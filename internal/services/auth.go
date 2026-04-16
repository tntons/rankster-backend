package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type AuthService struct {
	db              *gorm.DB
	users           *repositories.UserRepository
	googleClientID  string
	authTokenSecret string
}

type GoogleIdentity struct {
	Subject       string
	Email         string
	Name          string
	PictureURL    string
	EmailVerified bool
}

func NewAuthService(db *gorm.DB, users *repositories.UserRepository, authTokenSecret string, googleClientID string) *AuthService {
	return &AuthService{
		db:              db,
		users:           users,
		authTokenSecret: strings.TrimSpace(authTokenSecret),
		googleClientID:  strings.TrimSpace(googleClientID),
	}
}

func (s *AuthService) BuildAuthResponse(user models.User) (views.AuthResponse, error) {
	accessToken, err := auth.IssueUserToken(user.ID, s.authTokenSecret, 30*24*time.Hour)
	if err != nil {
		return views.AuthResponse{}, err
	}

	return views.AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		User:        views.BuildUser(user),
	}, nil
}

func (s *AuthService) UserFromAuthorization(header string) (*views.User, error) {
	authCtx := auth.FromAuthorization(header, s.authTokenSecret)
	if authCtx.Kind != "user" {
		return nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(authCtx.UserID)
	if err != nil {
		return nil, err
	}
	view := views.BuildUser(user)
	return &view, nil
}

func (s *AuthService) VerifyGoogleCredential(credential string) (GoogleIdentity, error) {
	payload, err := idtoken.Validate(context.Background(), credential, s.googleClientID)
	if err != nil {
		return GoogleIdentity{}, err
	}

	identity := GoogleIdentity{
		Subject:       payload.Subject,
		Email:         claimString(payload.Claims, "email"),
		Name:          claimString(payload.Claims, "name"),
		PictureURL:    claimString(payload.Claims, "picture"),
		EmailVerified: claimBool(payload.Claims, "email_verified"),
	}
	if strings.TrimSpace(identity.Subject) == "" || strings.TrimSpace(identity.Email) == "" {
		return GoogleIdentity{}, errors.New("missing google identity claims")
	}

	return identity, nil
}

func (s *AuthService) FindOrCreateGoogleUser(identity GoogleIdentity) (models.User, error) {
	var user models.User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		authRecord, err := s.users.LookupGoogleAuth(tx, identity.Subject)
		if err == nil {
			return s.hydrateAndRefreshGoogleUser(tx, authRecord.UserID, identity, &user)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		authRecord, err = s.users.LookupAuthByEmail(tx, identity.Email)
		if err == nil {
			return s.attachGoogleIdentityToExistingUser(tx, authRecord, identity, &user)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return s.createGoogleUser(tx, identity, &user)
	})

	return user, err
}

func (s *AuthService) hydrateAndRefreshGoogleUser(tx *gorm.DB, userID string, identity GoogleIdentity, out *models.User) error {
	if err := s.refreshGoogleUser(tx, userID, identity); err != nil {
		return err
	}

	user, err := repositories.FindUserByID(tx, userID)
	if err != nil {
		return err
	}

	*out = user
	return nil
}

func (s *AuthService) attachGoogleIdentityToExistingUser(tx *gorm.DB, authRecord models.UserAuth, identity GoogleIdentity, out *models.User) error {
	email := strings.ToLower(strings.TrimSpace(identity.Email))

	updates := map[string]any{
		"provider":     "GOOGLE",
		"provider_sub": identity.Subject,
		"email":        email,
	}
	if err := tx.Model(&models.UserAuth{}).Where("id = ?", authRecord.ID).Updates(updates).Error; err != nil {
		return err
	}

	return s.hydrateAndRefreshGoogleUser(tx, authRecord.UserID, identity, out)
}

func (s *AuthService) createGoogleUser(tx *gorm.DB, identity GoogleIdentity, out *models.User) error {
	userID := uuid.NewString()
	authID := uuid.NewString()
	profileID := uuid.NewString()
	statsID := uuid.NewString()
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	displayName := chooseDisplayName(identity)
	bio := "Signed in with Google"
	avatar := strings.TrimSpace(identity.PictureURL)
	username, err := s.uniqueUsername(tx, seedUsername(identity))
	if err != nil {
		return err
	}

	user := models.User{ID: userID}
	if err := tx.Create(&user).Error; err != nil {
		return err
	}

	authRecord := models.UserAuth{
		ID:          authID,
		UserID:      userID,
		Provider:    "GOOGLE",
		Email:       stringPtr(email),
		ProviderSub: stringPtr(identity.Subject),
	}
	if err := tx.Create(&authRecord).Error; err != nil {
		return err
	}

	profile := models.UserProfile{
		ID:          profileID,
		UserID:      userID,
		Username:    username,
		DisplayName: stringPtr(displayName),
		Bio:         stringPtr(bio),
		AvatarURL:   optionalStringPtr(avatar),
		Verified:    identity.EmailVerified,
	}
	if err := tx.Create(&profile).Error; err != nil {
		return err
	}

	stats := models.UserStats{
		ID:                statsID,
		UserID:            userID,
		RanksCreatedCount: 0,
		FollowersCount:    0,
		FollowingCount:    0,
		UpdatedAt:         time.Now(),
	}
	if err := tx.Create(&stats).Error; err != nil {
		return err
	}

	return s.hydrateAndRefreshGoogleUser(tx, userID, identity, out)
}

func (s *AuthService) refreshGoogleUser(tx *gorm.DB, userID string, identity GoogleIdentity) error {
	displayName := chooseDisplayName(identity)
	updates := map[string]any{
		"display_name": displayName,
		"verified":     identity.EmailVerified,
	}
	if avatar := strings.TrimSpace(identity.PictureURL); avatar != "" {
		updates["avatar_url"] = avatar
	}

	return tx.Model(&models.UserProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (s *AuthService) uniqueUsername(tx *gorm.DB, base string) (string, error) {
	candidate := seedUsernameValue(base)
	for suffix := 0; suffix < 100; suffix++ {
		username := candidate
		if suffix > 0 {
			username = fmt.Sprintf("%s-%d", candidate, suffix+1)
		}

		exists, err := s.users.UsernameExists(tx, username)
		if err != nil {
			return "", err
		}
		if !exists {
			return username, nil
		}
	}
	return "", errors.New("failed to generate unique username")
}

func claimString(claims map[string]any, key string) string {
	raw, ok := claims[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func claimBool(claims map[string]any, key string) bool {
	raw, ok := claims[key]
	if !ok {
		return false
	}
	value, ok := raw.(bool)
	if !ok {
		return false
	}
	return value
}

func chooseDisplayName(identity GoogleIdentity) string {
	if name := strings.TrimSpace(identity.Name); name != "" {
		return name
	}

	email := strings.TrimSpace(identity.Email)
	if email == "" {
		return "Rankster User"
	}

	localPart := email
	if at := strings.Index(localPart, "@"); at >= 0 {
		localPart = localPart[:at]
	}

	localPart = strings.ReplaceAll(localPart, ".", " ")
	localPart = strings.ReplaceAll(localPart, "_", " ")
	localPart = strings.ReplaceAll(localPart, "-", " ")
	localPart = strings.TrimSpace(localPart)
	if localPart == "" {
		return "Rankster User"
	}

	return titleWords(localPart)
}

func seedUsername(identity GoogleIdentity) string {
	if email := strings.TrimSpace(identity.Email); email != "" {
		localPart := email
		if at := strings.Index(localPart, "@"); at >= 0 {
			localPart = localPart[:at]
		}
		if strings.TrimSpace(localPart) != "" {
			return localPart
		}
	}
	if name := strings.TrimSpace(identity.Name); name != "" {
		return name
	}
	return "rankster-user"
}

func seedUsernameValue(base string) string {
	username := slugify(strings.ToLower(base))
	if username == "" {
		return "rankster-user"
	}
	return username
}

func titleWords(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

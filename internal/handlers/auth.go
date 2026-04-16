package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
)

func (h *FrontendHandler) MockLogin(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	_ = c.ShouldBindJSON(&body)

	username := strings.TrimSpace(body.Username)
	if username == "" {
		username = "me"
	}

	user, err := h.lookupUserByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load user"})
		return
	}

	session, err := h.buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to create auth session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *FrontendHandler) GoogleLogin(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	if h.googleClientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "GOOGLE_AUTH_NOT_CONFIGURED", "message": "google auth is not configured"})
		return
	}

	var body frontendGoogleAuthRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Credential) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "google credential is required"})
		return
	}

	identity, err := h.verifyGoogleCredential(body.Credential)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_GOOGLE_TOKEN", "message": "failed to validate google credential"})
		return
	}

	user, err := h.findOrCreateGoogleUser(identity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to sign in with google"})
		return
	}

	session, err := h.buildAuthResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to create auth session"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *FrontendHandler) GetAuthMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

type googleIdentity struct {
	Subject       string
	Email         string
	Name          string
	PictureURL    string
	EmailVerified bool
}

func (h *FrontendHandler) verifyGoogleCredential(credential string) (googleIdentity, error) {
	payload, err := idtoken.Validate(context.Background(), credential, h.googleClientID)
	if err != nil {
		return googleIdentity{}, err
	}

	identity := googleIdentity{
		Subject:       payload.Subject,
		Email:         claimString(payload.Claims, "email"),
		Name:          claimString(payload.Claims, "name"),
		PictureURL:    claimString(payload.Claims, "picture"),
		EmailVerified: claimBool(payload.Claims, "email_verified"),
	}
	if strings.TrimSpace(identity.Subject) == "" || strings.TrimSpace(identity.Email) == "" {
		return googleIdentity{}, errors.New("missing google identity claims")
	}

	return identity, nil
}

func (h *FrontendHandler) findOrCreateGoogleUser(identity googleIdentity) (models.User, error) {
	var user models.User

	err := h.db.Transaction(func(tx *gorm.DB) error {
		authRecord, err := h.lookupGoogleAuth(tx, identity.Subject)
		if err == nil {
			return h.hydrateAndRefreshGoogleUser(tx, authRecord.UserID, identity, &user)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		authRecord, err = h.lookupAuthByEmail(tx, identity.Email)
		if err == nil {
			return h.attachGoogleIdentityToExistingUser(tx, authRecord, identity, &user)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return h.createGoogleUser(tx, identity, &user)
	})

	return user, err
}

func (h *FrontendHandler) buildAuthResponse(user models.User) (frontendAuthResponse, error) {
	accessToken, err := auth.IssueUserToken(user.ID, h.authTokenSecret, 30*24*time.Hour)
	if err != nil {
		return frontendAuthResponse{}, err
	}

	return frontendAuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		User:        buildFrontendUser(user),
	}, nil
}

func (h *FrontendHandler) optionalUser(c *gin.Context) *frontendUserView {
	if h.db == nil {
		return nil
	}
	authCtx := auth.FromAuthorization(c.GetHeader("Authorization"), h.authTokenSecret)
	if authCtx.Kind != "user" {
		return nil
	}

	user, err := h.lookupUserByID(authCtx.UserID)
	if err != nil {
		return nil
	}
	view := buildFrontendUser(user)
	return &view
}

func (h *FrontendHandler) requireUser(c *gin.Context) (frontendUserView, bool) {
	if !h.ensureDB(c) {
		return frontendUserView{}, false
	}

	authCtx := auth.FromAuthorization(c.GetHeader("Authorization"), h.authTokenSecret)
	if authCtx.Kind != "user" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "missing bearer token"})
		return frontendUserView{}, false
	}

	user, err := h.lookupUserByID(authCtx.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "invalid bearer token"})
		return frontendUserView{}, false
	}

	return buildFrontendUser(user), true
}

func (h *FrontendHandler) lookupUserByID(userID string) (models.User, error) {
	var user models.User
	err := h.db.Preload("Profile").Preload("Stats").Where("id = ?", userID).First(&user).Error
	return user, err
}

func (h *FrontendHandler) lookupUserByUsername(username string) (models.User, error) {
	var profile models.UserProfile
	if err := h.db.Where("username = ?", username).First(&profile).Error; err != nil {
		return models.User{}, err
	}
	return h.lookupUserByID(profile.UserID)
}

func (h *FrontendHandler) lookupGoogleAuth(tx *gorm.DB, subject string) (models.UserAuth, error) {
	var authRecord models.UserAuth
	err := tx.Where("provider = ? AND provider_sub = ?", "GOOGLE", subject).First(&authRecord).Error
	return authRecord, err
}

func (h *FrontendHandler) lookupAuthByEmail(tx *gorm.DB, email string) (models.UserAuth, error) {
	var authRecord models.UserAuth
	err := tx.Where("LOWER(email) = LOWER(?)", email).First(&authRecord).Error
	return authRecord, err
}

func (h *FrontendHandler) hydrateAndRefreshGoogleUser(tx *gorm.DB, userID string, identity googleIdentity, out *models.User) error {
	if err := h.refreshGoogleUser(tx, userID, identity); err != nil {
		return err
	}

	user, err := h.lookupUserByIDWithDB(tx, userID)
	if err != nil {
		return err
	}

	*out = user
	return nil
}

func (h *FrontendHandler) attachGoogleIdentityToExistingUser(tx *gorm.DB, authRecord models.UserAuth, identity googleIdentity, out *models.User) error {
	email := strings.ToLower(strings.TrimSpace(identity.Email))

	updates := map[string]any{
		"provider":     "GOOGLE",
		"provider_sub": identity.Subject,
		"email":        email,
	}
	if err := tx.Model(&models.UserAuth{}).Where("id = ?", authRecord.ID).Updates(updates).Error; err != nil {
		return err
	}

	return h.hydrateAndRefreshGoogleUser(tx, authRecord.UserID, identity, out)
}

func (h *FrontendHandler) createGoogleUser(tx *gorm.DB, identity googleIdentity, out *models.User) error {
	userID := uuid.NewString()
	authID := uuid.NewString()
	profileID := uuid.NewString()
	statsID := uuid.NewString()
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	displayName := chooseDisplayName(identity)
	bio := "Signed in with Google"
	avatar := strings.TrimSpace(identity.PictureURL)
	username, err := h.uniqueUsername(tx, seedUsername(identity))
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

	return h.hydrateAndRefreshGoogleUser(tx, userID, identity, out)
}

func (h *FrontendHandler) refreshGoogleUser(tx *gorm.DB, userID string, identity googleIdentity) error {
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

func (h *FrontendHandler) lookupUserByIDWithDB(tx *gorm.DB, userID string) (models.User, error) {
	var user models.User
	err := tx.Preload("Profile").Preload("Stats").Where("id = ?", userID).First(&user).Error
	return user, err
}

func (h *FrontendHandler) uniqueUsername(tx *gorm.DB, base string) (string, error) {
	candidate := seedUsernameValue(base)
	for suffix := 0; suffix < 100; suffix++ {
		username := candidate
		if suffix > 0 {
			username = fmt.Sprintf("%s-%d", candidate, suffix+1)
		}

		var count int64
		if err := tx.Model(&models.UserProfile{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
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

func chooseDisplayName(identity googleIdentity) string {
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

func seedUsername(identity googleIdentity) string {
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

package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/config"
	"rankster-backend/internal/models"
)

type frontendUserView struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	DisplayName   string `json:"displayName"`
	Avatar        string `json:"avatar"`
	Bio           string `json:"bio"`
	Followers     int    `json:"followers"`
	Following     int    `json:"following"`
	TotalRankings int    `json:"totalRankings"`
	Verified      bool   `json:"verified"`
}

type frontendTierItem struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Emoji *string `json:"emoji,omitempty"`
}

type frontendTierData struct {
	S []frontendTierItem `json:"S"`
	A []frontendTierItem `json:"A"`
	B []frontendTierItem `json:"B"`
	C []frontendTierItem `json:"C"`
	D []frontendTierItem `json:"D"`
}

type frontendCommentView struct {
	ID        string           `json:"id"`
	User      frontendUserView `json:"user"`
	Text      string           `json:"text"`
	CreatedAt string           `json:"createdAt"`
	Likes     int              `json:"likes"`
}

type frontendRankPostView struct {
	ID               string                `json:"id"`
	User             frontendUserView      `json:"user"`
	Title            string                `json:"title"`
	Category         string                `json:"category"`
	CoverImage       string                `json:"coverImage"`
	Tiers            frontendTierData      `json:"tiers"`
	AllItems         []frontendTierItem    `json:"allItems"`
	Description      string                `json:"description"`
	Tags             []string              `json:"tags"`
	Likes            int                   `json:"likes"`
	IsLiked          bool                  `json:"isLiked"`
	Comments         []frontendCommentView `json:"comments"`
	Shares           int                   `json:"shares"`
	CreatedAt        string                `json:"createdAt"`
	IsPublic         bool                  `json:"isPublic"`
	ParticipantCount int                   `json:"participantCount"`
}

type frontendMessageView struct {
	ID          string           `json:"id"`
	User        frontendUserView `json:"user"`
	LastMessage string           `json:"lastMessage"`
	Timestamp   string           `json:"timestamp"`
	Unread      int              `json:"unread"`
}

type frontendChatMessageView struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Mine      bool   `json:"mine"`
	Timestamp string `json:"timestamp"`
}

type frontendMessageThreadDetailView struct {
	ID       string                    `json:"id"`
	User     frontendUserView          `json:"user"`
	Messages []frontendChatMessageView `json:"messages"`
}

type frontendTrendingTopicView struct {
	ID               string   `json:"id"`
	PostID           *string  `json:"postId,omitempty"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	CoverImage       string   `json:"coverImage"`
	ParticipantCount int      `json:"participantCount"`
	Tags             []string `json:"tags"`
}

type frontendCategoryView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Color string `json:"color"`
}

type frontendProfileResponse struct {
	User               frontendUserView              `json:"user"`
	Rankings           []frontendRankPostView        `json:"rankings"`
	LikedPosts         []frontendRankPostView        `json:"likedPosts"`
	PinnedPostID       *string                       `json:"pinnedPostId"`
	Stats              frontendProfileStatsView      `json:"stats"`
	FavoriteCategories []frontendProfileCategoryView `json:"favoriteCategories"`
	IsFollowing        bool                          `json:"isFollowing"`
}

type frontendProfileStatsView struct {
	TotalRankings int `json:"totalRankings"`
	Followers     int `json:"followers"`
	Following     int `json:"following"`
	TotalLikes    int `json:"totalLikes"`
}

type frontendProfileCategoryView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Pct   int    `json:"pct"`
}

type frontendSearchResponse struct {
	Users      []frontendUserView          `json:"users"`
	Topics     []frontendTrendingTopicView `json:"topics"`
	Categories []frontendCategoryView      `json:"categories"`
}

type frontendFeedResponse struct {
	Items      []frontendRankPostView `json:"items"`
	NextCursor any                    `json:"nextCursor"`
}

type frontendAuthResponse struct {
	AccessToken string           `json:"accessToken"`
	TokenType   string           `json:"tokenType"`
	User        frontendUserView `json:"user"`
}

type frontendGoogleAuthRequest struct {
	Credential string `json:"credential"`
}

type frontendLeaderboardEntry struct {
	Rank   int              `json:"rank"`
	User   frontendUserView `json:"user"`
	Score  int              `json:"score"`
	Change string           `json:"change"`
}

type frontendCreateRankRequest struct {
	Title       string             `json:"title"`
	Category    string             `json:"category"`
	Description string             `json:"description"`
	Tags        []string           `json:"tags"`
	Tiers       frontendTierData   `json:"tiers"`
	AllItems    []frontendTierItem `json:"allItems"`
	IsPublic    *bool              `json:"isPublic"`
}

type FrontendHandler struct {
	db              *gorm.DB
	googleClientID  string
	authTokenSecret string
}

func NewFrontendHandler(db *gorm.DB, cfg config.Config) *FrontendHandler {
	return &FrontendHandler{
		db:              db,
		googleClientID:  strings.TrimSpace(cfg.GoogleClientID),
		authTokenSecret: strings.TrimSpace(cfg.AuthTokenSecret),
	}
}

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

func (h *FrontendHandler) GetMainFeed(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	authUser := h.optionalUser(c)
	scope := strings.TrimSpace(strings.ToLower(c.DefaultQuery("scope", "for-you")))
	limit := parseIntWithDefault(c.Query("limit"), 20)
	if limit < 1 {
		limit = 20
	}

	offset := decodeCursor(c.Query("cursor"))
	var (
		lists      []models.TierListPost
		nextCursor any
		err        error
	)

	switch scope {
	case "following":
		if authUser == nil {
			c.JSON(http.StatusOK, frontendFeedResponse{Items: []frontendRankPostView{}, NextCursor: nil})
			return
		}
		lists, nextCursor, err = h.followingFeedTierLists(authUser.ID, offset, limit)
	default:
		lists, nextCursor, err = h.feedTierLists(offset, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load feed"})
		return
	}

	items, err := h.hydrateTierLists(lists, authUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to build feed"})
		return
	}

	c.JSON(http.StatusOK, frontendFeedResponse{Items: items, NextCursor: nextCursor})
}

func (h *FrontendHandler) GetProfileMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	profile, err := h.buildProfileResponse(user.ID, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *FrontendHandler) GetProfileByUsername(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	authUser := h.optionalUser(c)
	userRecord, err := h.lookupUserByUsername(c.Param("username"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	user := buildFrontendUser(userRecord)
	profile, err := h.buildProfileResponse(user.ID, authUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	profile.User = user
	c.JSON(http.StatusOK, profile)
}

func (h *FrontendHandler) FollowProfileUser(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	targetUser, err := h.lookupUserByUsername(c.Param("username"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to follow user"})
		return
	}

	if err := h.setFollowState(authUser.ID, targetUser.ID, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to follow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": true})
}

func (h *FrontendHandler) UnfollowProfileUser(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	targetUser, err := h.lookupUserByUsername(c.Param("username"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unfollow user"})
		return
	}

	if err := h.setFollowState(authUser.ID, targetUser.ID, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unfollow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": false})
}

func (h *FrontendHandler) PinProfilePost(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	postID := c.Param("postId")
	if err := h.setPinnedPost(authUser.ID, postID, true); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to pin post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pinnedPostId": postID})
}

func (h *FrontendHandler) UnpinProfilePost(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	if err := h.setPinnedPost(authUser.ID, c.Param("postId"), false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unpin post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pinnedPostId": nil})
}

func (h *FrontendHandler) SearchOverview(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	response, err := h.search(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to search"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *FrontendHandler) GetTrendingTopics(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	items, err := h.trendingTopics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load topics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) GetCategories(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	items, err := h.categories(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) GetMessages(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	items, err := h.messagesForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load messages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) GetMessageThread(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	thread, err := h.messageThreadDetail(user.ID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "THREAD_NOT_FOUND", "message": "thread not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load conversation"})
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *FrontendHandler) PostMessage(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "message text is required"})
		return
	}

	message, err := h.createMessage(user.ID, c.Param("id"), strings.TrimSpace(body.Text))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "THREAD_NOT_FOUND", "message": "thread not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to send message"})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *FrontendHandler) GetLeaderboard(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	timeframe := strings.TrimSpace(strings.ToLower(c.DefaultQuery("timeframe", "this-week")))
	category := strings.TrimSpace(strings.ToLower(c.DefaultQuery("category", "all")))

	items, err := h.leaderboard(timeframe, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load leaderboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) GetPost(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	authUser := h.optionalUser(c)
	post, err := h.postByID(c.Param("id"), authUser)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load post"})
		return
	}
	c.JSON(http.StatusOK, post)
}

func (h *FrontendHandler) CreateRank(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body frontendCreateRankRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "invalid create payload"})
		return
	}

	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Category) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "title and category are required"})
		return
	}

	post, err := h.createRank(user, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to create rank"})
		return
	}
	c.JSON(http.StatusCreated, post)
}

func (h *FrontendHandler) GetUserStats(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	stats, err := h.userStats(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId": user.ID,
		"totals": gin.H{
			"ranksCreated":     stats.RanksCreated,
			"likesReceived":    stats.LikesReceived,
			"commentsReceived": stats.CommentsReceived,
		},
		"engagement": gin.H{
			"followerCount":  stats.Followers,
			"followingCount": stats.Following,
		},
	})
}

func (h *FrontendHandler) ensureDB(c *gin.Context) bool {
	if h.db != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"code": "DATABASE_UNAVAILABLE", "message": "database is required"})
	return false
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

func (h *FrontendHandler) feedTierLists(offset, limit int) ([]models.TierListPost, any, error) {
	if offset < 0 {
		offset = 0
	}

	var lists []models.TierListPost
	err := h.db.
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("created_at desc").
		Offset(offset).
		Limit(limit + 1).
		Find(&lists).Error
	if err != nil {
		return nil, nil, err
	}

	var nextCursor any
	if len(lists) > limit {
		lists = lists[:limit]
		nextCursor = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset+limit)))
	}
	return lists, nextCursor, nil
}

func (h *FrontendHandler) followingFeedTierLists(userID string, offset, limit int) ([]models.TierListPost, any, error) {
	if offset < 0 {
		offset = 0
	}

	var lists []models.TierListPost
	err := h.db.
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Joins("JOIN follows ON follows.following_id = posts.creator_id").
		Where("follows.follower_id = ?", userID).
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("tier_list_posts.created_at desc").
		Offset(offset).
		Limit(limit + 1).
		Find(&lists).Error
	if err != nil {
		return nil, nil, err
	}

	var nextCursor any
	if len(lists) > limit {
		lists = lists[:limit]
		nextCursor = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset+limit)))
	}
	return lists, nextCursor, nil
}

func (h *FrontendHandler) rankingsForCreator(creatorID string, authUser *frontendUserView) ([]frontendRankPostView, error) {
	var lists []models.TierListPost
	err := h.db.
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Where("posts.creator_id = ?", creatorID).
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("tier_list_posts.created_at desc").
		Find(&lists).Error
	if err != nil {
		return nil, err
	}
	return h.hydrateTierLists(lists, authUser)
}

func (h *FrontendHandler) buildProfileResponse(profileUserID string, authUser *frontendUserView) (frontendProfileResponse, error) {
	userRecord, err := h.lookupUserByID(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	user := buildFrontendUser(userRecord)
	rankings, err := h.rankingsForCreator(profileUserID, authUser)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	likedPosts, err := h.likedRankingsForUser(profileUserID, authUser)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	stats, err := h.userStats(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	favoriteCategories, err := h.favoriteCategoriesForUser(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	pinnedPostID, err := h.pinnedPostIDForUser(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	isFollowing := false
	if authUser != nil && authUser.ID != profileUserID {
		isFollowing, err = h.followState(authUser.ID, profileUserID)
		if err != nil {
			return frontendProfileResponse{}, err
		}
	}

	return frontendProfileResponse{
		User:         user,
		Rankings:     rankings,
		LikedPosts:   likedPosts,
		PinnedPostID: pinnedPostID,
		Stats: frontendProfileStatsView{
			TotalRankings: stats.RanksCreated,
			Followers:     stats.Followers,
			Following:     stats.Following,
			TotalLikes:    stats.LikesReceived,
		},
		FavoriteCategories: favoriteCategories,
		IsFollowing:        isFollowing,
	}, nil
}

func (h *FrontendHandler) likedRankingsForUser(userID string, authUser *frontendUserView) ([]frontendRankPostView, error) {
	var lists []models.TierListPost
	err := h.db.
		Joins("JOIN post_likes ON post_likes.post_id = tier_list_posts.post_id").
		Where("post_likes.user_id = ?", userID).
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("post_likes.created_at desc").
		Find(&lists).Error
	if err != nil {
		return nil, err
	}

	return h.hydrateTierLists(lists, authUser)
}

func (h *FrontendHandler) favoriteCategoriesForUser(userID string) ([]frontendProfileCategoryView, error) {
	type categoryCountRow struct {
		ID    string
		Name  string
		Emoji string
		Count int
	}

	var rows []categoryCountRow
	err := h.db.Table("posts").
		Select("categories.slug AS id, categories.name AS name, COALESCE(categories.emoji, '') AS emoji, COUNT(*) AS count").
		Joins("JOIN categories ON categories.id = posts.category_id").
		Where("posts.creator_id = ?", userID).
		Group("categories.slug, categories.name, categories.emoji").
		Order("count DESC, categories.name ASC").
		Limit(4).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	total := 0
	for _, row := range rows {
		total += row.Count
	}

	out := make([]frontendProfileCategoryView, 0, len(rows))
	for _, row := range rows {
		pct := 0
		if total > 0 {
			pct = int(float64(row.Count) / float64(total) * 100)
		}
		out = append(out, frontendProfileCategoryView{
			ID:    row.ID,
			Name:  row.Name,
			Emoji: row.Emoji,
			Pct:   pct,
		})
	}
	return out, nil
}

func (h *FrontendHandler) pinnedPostIDForUser(userID string) (*string, error) {
	var pinned models.PinnedPost
	err := h.db.Where("user_id = ?", userID).Order("COALESCE(\"order\", 999999) asc, created_at asc").First(&pinned).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pinned.PostID, nil
}

func (h *FrontendHandler) followState(followerID, followingID string) (bool, error) {
	var count int64
	if err := h.db.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *FrontendHandler) setFollowState(followerID, followingID string, shouldFollow bool) error {
	if followerID == followingID {
		return nil
	}

	return h.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Follow
		err := tx.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if shouldFollow {
			if err == nil {
				return nil
			}
			follow := models.Follow{
				ID:          generateUUID(),
				FollowerID:  followerID,
				FollowingID: followingID,
				CreatedAt:   time.Now(),
			}
			if err := tx.Create(&follow).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.UserStats{}).Where("user_id = ?", followerID).
				Update("following_count", gorm.Expr("following_count + 1")).Error; err != nil {
				return err
			}
			return tx.Model(&models.UserStats{}).Where("user_id = ?", followingID).
				Update("followers_count", gorm.Expr("followers_count + 1")).Error
		}

		if err == gorm.ErrRecordNotFound {
			return nil
		}
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.UserStats{}).Where("user_id = ?", followerID).
			Update("following_count", gorm.Expr("GREATEST(following_count - 1, 0)")).Error; err != nil {
			return err
		}
		return tx.Model(&models.UserStats{}).Where("user_id = ?", followingID).
			Update("followers_count", gorm.Expr("GREATEST(followers_count - 1, 0)")).Error
	})
}

func (h *FrontendHandler) setPinnedPost(userID, postID string, shouldPin bool) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.PinnedPost{}).Error; err != nil {
			return err
		}
		if !shouldPin {
			return nil
		}

		var post models.Post
		if err := tx.Where("id = ? AND creator_id = ?", postID, userID).First(&post).Error; err != nil {
			return err
		}

		pinned := models.PinnedPost{
			ID:        generateUUID(),
			UserID:    userID,
			PostID:    postID,
			Order:     intPtrValue(1),
			CreatedAt: time.Now(),
		}
		return tx.Create(&pinned).Error
	})
}

func (h *FrontendHandler) search(query string) (frontendSearchResponse, error) {
	categories, err := h.categories(query)
	if err != nil {
		return frontendSearchResponse{}, err
	}

	topics, err := h.trendingTopicsFiltered(query, 6)
	if err != nil {
		return frontendSearchResponse{}, err
	}

	var users []models.User
	userQuery := h.db.Preload("Profile").Preload("Stats").Joins("JOIN user_profiles ON user_profiles.user_id = users.id")
	if query != "" {
		like := "%" + query + "%"
		userQuery = userQuery.Where("LOWER(user_profiles.username) LIKE ? OR LOWER(COALESCE(user_profiles.display_name, '')) LIKE ? OR LOWER(COALESCE(user_profiles.bio, '')) LIKE ?", like, like, like)
	}
	if err := userQuery.Limit(5).Find(&users).Error; err != nil {
		return frontendSearchResponse{}, err
	}

	response := frontendSearchResponse{
		Users:      make([]frontendUserView, 0, len(users)),
		Topics:     topics,
		Categories: categories,
	}
	for _, user := range users {
		response.Users = append(response.Users, buildFrontendUser(user))
	}
	return response, nil
}

func (h *FrontendHandler) trendingTopics() ([]frontendTrendingTopicView, error) {
	return h.trendingTopicsFiltered("", 100)
}

func (h *FrontendHandler) trendingTopicsFiltered(query string, limit int) ([]frontendTrendingTopicView, error) {
	var topics []models.TrendingTopic
	db := h.db.Preload("Category").Preload("CoverAsset").Preload("SourcePost").Order("participant_count desc")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(title) LIKE ? OR EXISTS (SELECT 1 FROM unnest(tags) tag WHERE LOWER(tag) LIKE ?)", like, like)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&topics).Error; err != nil {
		return nil, err
	}

	items := make([]frontendTrendingTopicView, 0, len(topics))
	for _, topic := range topics {
		items = append(items, frontendTrendingTopicView{
			ID:               topic.ID,
			PostID:           topic.SourcePostID,
			Title:            topic.Title,
			Category:         topic.Category.Slug,
			CoverImage:       assetOrFallback(topic.CoverAsset, "ranks", slugify(topic.Title)),
			ParticipantCount: topic.ParticipantCount,
			Tags:             append([]string{}, topic.Tags...),
		})
	}
	return items, nil
}

func (h *FrontendHandler) categories(query string) ([]frontendCategoryView, error) {
	var categories []models.Category
	db := h.db.Order("name asc")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ?", like, like)
	}
	if err := db.Find(&categories).Error; err != nil {
		return nil, err
	}

	items := make([]frontendCategoryView, 0, len(categories))
	for _, category := range categories {
		emoji := ""
		if category.Emoji != nil {
			emoji = *category.Emoji
		}
		color := ""
		if category.Color != nil {
			color = *category.Color
		}
		items = append(items, frontendCategoryView{
			ID:    category.Slug,
			Name:  category.Name,
			Emoji: emoji,
			Color: color,
		})
	}
	if len(items) > 6 && query != "" {
		items = items[:6]
	}
	return items, nil
}

func (h *FrontendHandler) messagesForUser(userID string) ([]frontendMessageView, error) {
	var threads []models.MessageThread
	err := h.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Where("owner_user_id = ?", userID).
		Order("updated_at desc").
		Find(&threads).Error
	if err != nil {
		return nil, err
	}

	items := make([]frontendMessageView, 0, len(threads))
	for _, thread := range threads {
		items = append(items, frontendMessageView{
			ID:          thread.ID,
			User:        buildFrontendUser(thread.PeerUser),
			LastMessage: thread.LastMessage,
			Timestamp:   relativeTime(thread.UpdatedAt),
			Unread:      thread.UnreadCount,
		})
	}
	return items, nil
}

func (h *FrontendHandler) messageThreadDetail(userID, threadID string) (frontendMessageThreadDetailView, error) {
	var thread models.MessageThread
	err := h.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Preload("Messages", func(db *gorm.DB) *gorm.DB { return db.Order("created_at asc") }).
		Where("id = ? AND owner_user_id = ?", threadID, userID).
		First(&thread).Error
	if err != nil {
		return frontendMessageThreadDetailView{}, err
	}

	messages := make([]frontendChatMessageView, 0, len(thread.Messages))
	for _, message := range thread.Messages {
		messages = append(messages, frontendChatMessageView{
			ID:        message.ID,
			Text:      message.Body,
			Mine:      message.SenderUserID == userID,
			Timestamp: chatTimestamp(message.CreatedAt),
		})
	}

	return frontendMessageThreadDetailView{
		ID:       thread.ID,
		User:     buildFrontendUser(thread.PeerUser),
		Messages: messages,
	}, nil
}

func (h *FrontendHandler) createMessage(userID, threadID, text string) (frontendChatMessageView, error) {
	now := time.Now()
	messageID := generateUUID()

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var thread models.MessageThread
		if err := tx.Where("id = ? AND owner_user_id = ?", threadID, userID).First(&thread).Error; err != nil {
			return err
		}

		message := models.DirectMessage{
			ID:           messageID,
			ThreadID:     thread.ID,
			SenderUserID: userID,
			Body:         text,
			CreatedAt:    now,
		}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}

		return tx.Model(&models.MessageThread{}).Where("id = ?", thread.ID).Updates(map[string]any{
			"last_message": text,
			"updated_at":   now,
			"unread_count": 0,
		}).Error
	})
	if err != nil {
		return frontendChatMessageView{}, err
	}

	return frontendChatMessageView{
		ID:        messageID,
		Text:      text,
		Mine:      true,
		Timestamp: "Now",
	}, nil
}

func (h *FrontendHandler) leaderboard(timeframe string, category string) ([]frontendLeaderboardEntry, error) {
	var entries []models.LeaderboardEntry
	err := h.db.
		Preload("User.Profile").
		Preload("User.Stats").
		Order("rank asc").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}

	items := make([]frontendLeaderboardEntry, 0, len(entries))
	for _, entry := range entries {
		score := adjustedLeaderboardScore(entry, timeframe, category)
		change := adjustedLeaderboardChange(entry, timeframe, category)
		items = append(items, frontendLeaderboardEntry{
			Rank:   entry.Rank,
			User:   buildFrontendUser(entry.User),
			Score:  score,
			Change: change,
		})
	}

	slices.SortFunc(items, func(a, b frontendLeaderboardEntry) int {
		if a.Score == b.Score {
			return strings.Compare(a.User.Username, b.User.Username)
		}
		return b.Score - a.Score
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return items, nil
}

func (h *FrontendHandler) postByID(postID string, authUser *frontendUserView) (frontendRankPostView, error) {
	list, err := h.lookupTierListPost(postID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		resolvedPostID, resolveErr := h.resolveTrendingTopicPostID(postID)
		if resolveErr != nil {
			return frontendRankPostView{}, resolveErr
		}
		if resolvedPostID != nil {
			list, err = h.lookupTierListPost(*resolvedPostID)
		}
	}
	if err != nil {
		return frontendRankPostView{}, err
	}

	items, err := h.hydrateTierLists([]models.TierListPost{list}, authUser)
	if err != nil {
		return frontendRankPostView{}, err
	}
	if len(items) == 0 {
		return frontendRankPostView{}, gorm.ErrRecordNotFound
	}
	return items[0], nil
}

func (h *FrontendHandler) lookupTierListPost(postID string) (models.TierListPost, error) {
	var list models.TierListPost
	err := h.db.
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Where("post_id = ?", postID).
		First(&list).Error
	return list, err
}

func (h *FrontendHandler) resolveTrendingTopicPostID(topicID string) (*string, error) {
	var topic models.TrendingTopic
	err := h.db.Where("id = ?", topicID).First(&topic).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return topic.SourcePostID, nil
}

func (h *FrontendHandler) createRank(user frontendUserView, body frontendCreateRankRequest) (frontendRankPostView, error) {
	now := time.Now()
	postID := ""

	err := h.db.Transaction(func(tx *gorm.DB) error {
		category, err := ensureCategory(tx, body.Category, now)
		if err != nil {
			return err
		}

		coverURL := fmt.Sprintf("http://localhost:8000/assets/ranks/%s.svg", slugify(body.Title))
		asset := models.Asset{ID: "", URL: ""}
		if err := tx.Where("url = ?", coverURL).First(&asset).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			asset = models.Asset{ID: generateUUID(), URL: coverURL, CreatedAt: now}
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
		}

		postID = generateUUID()
		visibility := "PUBLIC"
		if body.IsPublic != nil && !*body.IsPublic {
			visibility = "PRIVATE"
		}

		post := models.Post{
			ID:         postID,
			Type:       "RANK",
			Visibility: visibility,
			CreatorID:  user.ID,
			CategoryID: category.ID,
			Caption:    stringPtr(body.Description),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}

		tags := body.Tags
		if len(tags) == 0 {
			tags = []string{body.Category}
		}
		tierPost := models.TierListPost{
			PostID:           postID,
			Title:            body.Title,
			Description:      stringPtr(body.Description),
			CoverAssetID:     &asset.ID,
			Tags:             pq.StringArray(tags),
			ParticipantCount: max(1, len(body.AllItems)),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&tierPost).Error; err != nil {
			return err
		}

		tierLookup := map[string]struct {
			Key      string
			Position int
			Emoji    *string
		}{}
		recordTierItems := func(key string, items []frontendTierItem) {
			for index, item := range items {
				tierLookup[item.ID] = struct {
					Key      string
					Position int
					Emoji    *string
				}{Key: key, Position: index, Emoji: item.Emoji}
			}
		}
		recordTierItems("S", body.Tiers.S)
		recordTierItems("A", body.Tiers.A)
		recordTierItems("B", body.Tiers.B)
		recordTierItems("C", body.Tiers.C)
		recordTierItems("D", body.Tiers.D)

		for index, item := range body.AllItems {
			tierMeta := tierLookup[item.ID]
			entry := models.TierListItem{
				ID:             generateUUID(),
				TierListPostID: postID,
				ExternalID:     item.ID,
				Name:           item.Name,
				Emoji:          coalesceEmoji(item.Emoji, tierMeta.Emoji),
				TierKey:        tierMeta.Key,
				TierPosition:   tierMeta.Position,
				ListPosition:   index,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}

		metrics := models.PostMetrics{PostID: postID, UpdatedAt: now}
		if err := tx.Create(&metrics).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.UserStats{}).Where("user_id = ?", user.ID).
			Update("ranks_created_count", gorm.Expr("ranks_created_count + ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return frontendRankPostView{}, err
	}

	return h.postByID(postID, &user)
}

type computedUserStats struct {
	RanksCreated     int
	LikesReceived    int
	CommentsReceived int
	Followers        int
	Following        int
}

func (h *FrontendHandler) userStats(userID string) (computedUserStats, error) {
	var user models.User
	if err := h.db.Preload("Stats").Where("id = ?", userID).First(&user).Error; err != nil {
		return computedUserStats{}, err
	}

	var likesReceived int64
	if err := h.db.Model(&models.PostMetrics{}).
		Joins("JOIN posts ON posts.id = post_metrics.post_id").
		Where("posts.creator_id = ?", userID).
		Select("COALESCE(SUM(post_metrics.like_count), 0)").
		Scan(&likesReceived).Error; err != nil {
		return computedUserStats{}, err
	}

	var commentsReceived int64
	if err := h.db.Model(&models.Comment{}).
		Joins("JOIN posts ON posts.id = comments.post_id").
		Where("posts.creator_id = ?", userID).
		Count(&commentsReceived).Error; err != nil {
		return computedUserStats{}, err
	}

	stats := computedUserStats{
		LikesReceived:    int(likesReceived),
		CommentsReceived: int(commentsReceived),
	}
	if user.Stats != nil {
		stats.RanksCreated = user.Stats.RanksCreatedCount
		stats.Followers = user.Stats.FollowersCount
		stats.Following = user.Stats.FollowingCount
	}
	return stats, nil
}

func (h *FrontendHandler) hydrateTierLists(lists []models.TierListPost, authUser *frontendUserView) ([]frontendRankPostView, error) {
	postIDs := make([]string, 0, len(lists))
	for _, list := range lists {
		postIDs = append(postIDs, list.PostID)
	}

	commentsByPost, err := h.loadComments(postIDs)
	if err != nil {
		return nil, err
	}
	likedByPost, err := h.loadLikedPosts(postIDs, authUser)
	if err != nil {
		return nil, err
	}

	items := make([]frontendRankPostView, 0, len(lists))
	for _, list := range lists {
		items = append(items, hydrateTierList(list, commentsByPost[list.PostID], likedByPost[list.PostID]))
	}
	return items, nil
}

func (h *FrontendHandler) loadComments(postIDs []string) (map[string][]frontendCommentView, error) {
	out := map[string][]frontendCommentView{}
	if len(postIDs) == 0 {
		return out, nil
	}

	var comments []models.Comment
	err := h.db.
		Preload("Author.Profile").
		Preload("Author.Stats").
		Where("post_id IN ?", postIDs).
		Order("created_at desc").
		Find(&comments).Error
	if err != nil {
		return nil, err
	}

	for _, comment := range comments {
		out[comment.PostID] = append(out[comment.PostID], frontendCommentView{
			ID:        comment.ID,
			User:      buildFrontendUser(comment.Author),
			Text:      comment.Body,
			CreatedAt: relativeTime(comment.CreatedAt),
			Likes:     comment.LikeCount,
		})
	}
	return out, nil
}

func (h *FrontendHandler) loadLikedPosts(postIDs []string, authUser *frontendUserView) (map[string]bool, error) {
	out := map[string]bool{}
	if authUser == nil || len(postIDs) == 0 {
		return out, nil
	}

	var likes []models.PostLike
	if err := h.db.Where("user_id = ? AND post_id IN ?", authUser.ID, postIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, like := range likes {
		out[like.PostID] = true
	}
	return out, nil
}

func hydrateTierList(list models.TierListPost, comments []frontendCommentView, isLiked bool) frontendRankPostView {
	if comments == nil {
		comments = []frontendCommentView{}
	}
	return frontendRankPostView{
		ID:               list.PostID,
		User:             buildFrontendUser(list.Post.Creator),
		Title:            list.Title,
		Category:         list.Post.Category.Slug,
		CoverImage:       assetOrFallback(list.CoverAsset, "ranks", slugify(list.Title)),
		Tiers:            buildTierData(list.Items),
		AllItems:         buildAllItems(list.Items),
		Description:      derefString(list.Description),
		Tags:             append([]string{}, list.Tags...),
		Likes:            metricLikeCount(list.Post.Metrics),
		IsLiked:          isLiked,
		Comments:         comments,
		Shares:           metricShareCount(list.Post.Metrics),
		CreatedAt:        relativeTime(list.CreatedAt),
		IsPublic:         list.Post.Visibility == "PUBLIC",
		ParticipantCount: list.ParticipantCount,
	}
}

func buildFrontendUser(user models.User) frontendUserView {
	profile := models.UserProfile{}
	if user.Profile != nil {
		profile = *user.Profile
	}
	stats := models.UserStats{}
	if user.Stats != nil {
		stats = *user.Stats
	}

	displayName := profile.Username
	if profile.DisplayName != nil && strings.TrimSpace(*profile.DisplayName) != "" {
		displayName = *profile.DisplayName
	}

	avatar := assetURL("avatars", profile.Username)
	if profile.AvatarURL != nil && strings.TrimSpace(*profile.AvatarURL) != "" {
		avatar = *profile.AvatarURL
	}

	return frontendUserView{
		ID:            user.ID,
		Username:      profile.Username,
		DisplayName:   displayName,
		Avatar:        avatar,
		Bio:           derefString(profile.Bio),
		Followers:     stats.FollowersCount,
		Following:     stats.FollowingCount,
		TotalRankings: stats.RanksCreatedCount,
		Verified:      profile.Verified,
	}
}

func buildTierData(items []models.TierListItem) frontendTierData {
	data := frontendTierData{S: []frontendTierItem{}, A: []frontendTierItem{}, B: []frontendTierItem{}, C: []frontendTierItem{}, D: []frontendTierItem{}}
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		if a.TierKey == b.TierKey {
			return a.TierPosition - b.TierPosition
		}
		return a.ListPosition - b.ListPosition
	})

	for _, item := range sorted {
		view := frontendTierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji}
		switch item.TierKey {
		case "S":
			data.S = append(data.S, view)
		case "A":
			data.A = append(data.A, view)
		case "B":
			data.B = append(data.B, view)
		case "C":
			data.C = append(data.C, view)
		case "D":
			data.D = append(data.D, view)
		}
	}
	return data
}

func buildAllItems(items []models.TierListItem) []frontendTierItem {
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		return a.ListPosition - b.ListPosition
	})

	out := make([]frontendTierItem, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, frontendTierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji})
	}
	return out
}

func ensureCategory(tx *gorm.DB, slug string, now time.Time) (models.Category, error) {
	slug = slugify(slug)
	var category models.Category
	err := tx.Where("slug = ?", slug).First(&category).Error
	if err == nil {
		return category, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.Category{}, err
	}

	name := titleizeSlug(slug)
	category = models.Category{
		ID:        generateUUID(),
		Slug:      slug,
		Name:      name,
		Tags:      pq.StringArray{slug},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return category, tx.Create(&category).Error
}

func titleizeSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func decodeCursor(raw string) int {
	if raw == "" {
		return 0
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0
	}
	return value
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	diff := time.Since(t)
	if diff < time.Minute {
		return "Just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
}

func chatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("3:04 PM")
}

func adjustedLeaderboardScore(entry models.LeaderboardEntry, timeframe string, category string) int {
	score := entry.Score

	switch timeframe {
	case "this-month":
		score += 4200 - (entry.Rank * 350)
	case "all-time":
		score += 9100 - (entry.Rank * 500)
	default:
		score += 1800 - (entry.Rank * 200)
	}

	if category != "" && category != "all" {
		score += stableValue(entry.UserID+":"+category) % 6000
	}

	return max(score, 1)
}

func adjustedLeaderboardChange(entry models.LeaderboardEntry, timeframe string, category string) string {
	base := stableValue(entry.UserID + ":" + timeframe + ":" + category)
	delta := (base % 7) - 3
	if delta > 0 {
		return fmt.Sprintf("+%d", delta)
	}
	if delta < 0 {
		return fmt.Sprintf("%d", delta)
	}
	return "0"
}

func assetOrFallback(asset *models.Asset, kind, slug string) string {
	if asset != nil && strings.TrimSpace(asset.URL) != "" {
		return asset.URL
	}
	return assetURL(kind, slug)
}

func assetURL(kind string, slug string) string {
	return fmt.Sprintf("http://localhost:8000/assets/%s/%s.svg", kind, safeSlug(slug))
}

func metricLikeCount(metrics *models.PostMetrics) int {
	if metrics == nil {
		return 0
	}
	return metrics.LikeCount
}

func metricShareCount(metrics *models.PostMetrics) int {
	if metrics == nil {
		return 0
	}
	return metrics.ShareCount
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func generateUUID() string {
	return uuid.NewString()
}

func stableValue(input string) int {
	total := 0
	for _, char := range input {
		total += int(char)
	}
	return total
}

func intPtrValue(value int) *int {
	return &value
}

func coalesceEmoji(primary, secondary *string) *string {
	if primary != nil {
		return primary
	}
	return secondary
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "'", "")
	value = strings.ReplaceAll(value, "&", "and")

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "rankster"
	}
	return result
}

func safeSlug(raw string) string {
	value := strings.Trim(strings.ToLower(raw), "/ ")
	if value == "" {
		return "rankster"
	}

	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "rankster"
	}
	return builder.String()
}

func parseIntWithDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func stringPtr(value string) *string {
	return &value
}

func optionalStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
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

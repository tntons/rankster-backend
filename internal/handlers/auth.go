package handlers

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/models"
)

func (h *Handler) MockLogin(c *gin.Context) {
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

func (h *Handler) GoogleLogin(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	if h.googleClientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "GOOGLE_AUTH_NOT_CONFIGURED", "message": "google auth is not configured"})
		return
	}

	var body googleAuthRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Credential) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "google credential is required"})
		return
	}

	identity, err := h.authService.VerifyGoogleCredential(body.Credential)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_GOOGLE_TOKEN", "message": "failed to validate google credential"})
		return
	}

	user, err := h.authService.FindOrCreateGoogleUser(identity)
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

func (h *Handler) GetAuthMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) buildAuthResponse(user models.User) (authResponse, error) {
	return h.authService.BuildAuthResponse(user)
}

func (h *Handler) optionalUser(c *gin.Context) *userView {
	if h.db == nil {
		return nil
	}
	user, err := h.authService.UserFromAuthorization(c.GetHeader("Authorization"))
	if err != nil {
		return nil
	}
	return user
}

func (h *Handler) requireUser(c *gin.Context) (userView, bool) {
	if !h.ensureDB(c) {
		return userView{}, false
	}

	user, err := h.authService.UserFromAuthorization(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "invalid bearer token"})
		return userView{}, false
	}

	return *user, true
}

func (h *Handler) lookupUserByID(userID string) (models.User, error) {
	return h.userRepo.FindByID(userID)
}

func (h *Handler) lookupUserByUsername(username string) (models.User, error) {
	return h.userRepo.FindByUsername(username)
}

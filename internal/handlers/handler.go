package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/config"
)

func NewFrontendHandler(db *gorm.DB, cfg config.Config) *FrontendHandler {
	publicBaseURL := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	if publicBaseURL == "" {
		publicBaseURL = "http://localhost:8000"
	}
	uploadDir := strings.TrimSpace(cfg.UploadDir)
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	return &FrontendHandler{
		db:              db,
		publicBaseURL:   publicBaseURL,
		uploadDir:       uploadDir,
		googleClientID:  strings.TrimSpace(cfg.GoogleClientID),
		authTokenSecret: strings.TrimSpace(cfg.AuthTokenSecret),
		chatHub:         newFrontendChatHub(),
		messageInboxHub: newFrontendMessageInboxHub(),
		notificationHub: newFrontendNotificationHub(),
	}
}

func (h *FrontendHandler) UploadDir() string {
	return h.uploadDir
}

func (h *FrontendHandler) ensureDB(c *gin.Context) bool {
	if h.db != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"code": "DATABASE_UNAVAILABLE", "message": "database is required"})
	return false
}

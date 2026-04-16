package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/config"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/services"
)

type Handler struct {
	db                  *gorm.DB
	publicBaseURL       string
	uploadDir           string
	googleClientID      string
	authTokenSecret     string
	userRepo            *repositories.UserRepository
	authService         *services.AuthService
	feedService         *services.FeedService
	rankPostService     *services.RankPostService
	profileService      *services.ProfileService
	messageService      *services.MessageService
	notificationService *services.NotificationService
	searchService       *services.SearchService
	leaderboardService  *services.LeaderboardService
	chatHub             *chatHub
	messageInboxHub     *messageInboxHub
	notificationHub     *notificationHub
}

func NewHandler(db *gorm.DB, cfg config.Config) *Handler {
	publicBaseURL := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	if publicBaseURL == "" {
		publicBaseURL = "http://localhost:8000"
	}
	uploadDir := strings.TrimSpace(cfg.UploadDir)
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	chatHub := newChatHub()
	userRepo := repositories.NewUserRepository(db)
	tierListRepo := repositories.NewTierListRepository(db)
	interactionRepo := repositories.NewInteractionRepository(db)
	messageRepo := repositories.NewMessageRepository(db)
	profileRepo := repositories.NewProfileRepository(db)
	notificationRepo := repositories.NewNotificationRepository(db)
	searchRepo := repositories.NewSearchRepository(db)
	leaderboardRepo := repositories.NewLeaderboardRepository(db)

	notificationService := services.NewNotificationService(notificationRepo)
	rankPostService := services.NewRankPostService(db, tierListRepo, interactionRepo, notificationService)

	return &Handler{
		db:                  db,
		publicBaseURL:       publicBaseURL,
		uploadDir:           uploadDir,
		googleClientID:      strings.TrimSpace(cfg.GoogleClientID),
		authTokenSecret:     strings.TrimSpace(cfg.AuthTokenSecret),
		userRepo:            userRepo,
		authService:         services.NewAuthService(db, userRepo, strings.TrimSpace(cfg.AuthTokenSecret), strings.TrimSpace(cfg.GoogleClientID)),
		feedService:         services.NewFeedService(tierListRepo, rankPostService),
		rankPostService:     rankPostService,
		profileService:      services.NewProfileService(db, userRepo, profileRepo, rankPostService, notificationService),
		messageService:      services.NewMessageService(db, messageRepo, chatHub.hasSubscribers),
		notificationService: notificationService,
		searchService:       services.NewSearchService(searchRepo),
		leaderboardService:  services.NewLeaderboardService(leaderboardRepo),
		chatHub:             chatHub,
		messageInboxHub:     newMessageInboxHub(),
		notificationHub:     newNotificationHub(),
	}
}

func (h *Handler) UploadDir() string {
	return h.uploadDir
}

func (h *Handler) ensureDB(c *gin.Context) bool {
	if h.db != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"code": "DATABASE_UNAVAILABLE", "message": "database is required"})
	return false
}

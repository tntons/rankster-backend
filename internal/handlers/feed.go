package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
)

type FeedHandler struct {
	db *gorm.DB
}

func NewFeedHandler(db *gorm.DB) *FeedHandler {
	return &FeedHandler{db: db}
}

type feedCursor struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

func (h *FeedHandler) GetMainFeed(c *gin.Context) {
	limit := parseIntWithDefault(c.Query("limit"), 20)
	if limit < 1 {
		limit = 1
	}
	if limit > 50 {
		limit = 50
	}

	var cursor *feedCursor
	if raw := c.Query("cursor"); raw != "" {
		parsed, err := decodeCursor(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_CURSOR", "message": "cursor is invalid"})
			return
		}
		cursor = &parsed
	}

	authCtx := auth.FromAuthorization(c.GetHeader("Authorization"))

	query := h.db.Model(&models.Post{}).
		Where("visibility = ?", "PUBLIC").
		Where("type = ?", "RANK")

	if cursor != nil {
		createdAt, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
		if err == nil {
			query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", createdAt, createdAt, cursor.ID)
		}
	}

	var posts []models.Post
	if err := applyPostPreloads(query).
		Order("created_at desc, id desc").
		Limit(limit + 1).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": "failed to load feed"})
		return
	}

	hasMore := len(posts) > limit
	page := posts
	if hasMore {
		page = posts[:limit]
	}

	var nextCursor string
	if hasMore && len(page) > 0 {
		last := page[len(page)-1]
		nextCursor = encodeCursor(feedCursor{
			CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339Nano),
			ID:        last.ID,
		})
	}

	response, err := buildFrontendFeedResponse(h.db, page, authCtx, nextCursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FEED_BUILD_ERROR", "message": "failed to build feed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func applyPostPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Creator.Profile").
		Preload("Creator.Stats").
		Preload("Category").
		Preload("Metrics").
		Preload("Rank.Image").
		Preload("Survey.SponsorOrg").
		Preload("Survey.Campaign").
		Preload("Survey.Questions", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("\"order\" asc")
		}).
		Preload("Survey.Questions.Options", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("\"order\" asc")
		})
}

func encodeCursor(cursor feedCursor) string {
	payload, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(raw string) (feedCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return feedCursor{}, err
	}
	var cursor feedCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return feedCursor{}, err
	}
	return cursor, nil
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

func emptyToNull(value string) any {
	if value == "" {
		return nil
	}
	return value
}

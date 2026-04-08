package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
)

type RankHandler struct {
	db *gorm.DB
}

func NewRankHandler(db *gorm.DB) *RankHandler {
	return &RankHandler{db: db}
}

type createRankRequest struct {
	CategoryID   string  `json:"categoryId" binding:"required"`
	TemplateID   string  `json:"templateId" binding:"required"`
	TierKey      string  `json:"tierKey" binding:"required"`
	ImageAssetID string  `json:"imageAssetId" binding:"required"`
	Caption      *string `json:"caption"`
	SubjectTitle *string `json:"subjectTitle"`
	SubjectURL   *string `json:"subjectUrl"`
	Visibility   *string `json:"visibility"`
}

func (h *RankHandler) CreateRank(c *gin.Context) {
	authCtx := auth.FromAuthorization(c.GetHeader("Authorization"))
	if authCtx.Kind != "user" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Missing bearer token"})
		return
	}

	var req createRankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "Invalid body"})
		return
	}

	visibility := "PUBLIC"
	if req.Visibility != nil {
		visibility = *req.Visibility
	}

	var created models.Post
	var handledErr *handlerError
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var template models.TierListTemplate
		if err := tx.Where("id = ? AND category_id = ?", req.TemplateID, req.CategoryID).
			Select("id").
			First(&template).Error; err != nil {
			handledErr = &handlerError{status: http.StatusBadRequest, code: "TEMPLATE_NOT_FOUND", message: "Template does not belong to category"}
			return err
		}

		var tier models.TierDefinition
		if err := tx.Where("template_id = ? AND key = ?", req.TemplateID, req.TierKey).
			Select("id").
			First(&tier).Error; err != nil {
			handledErr = &handlerError{status: http.StatusBadRequest, code: "INVALID_TIER", message: "tierKey is not defined on template"}
			return err
		}

		var asset models.Asset
		if err := tx.Where("id = ?", req.ImageAssetID).Select("id").First(&asset).Error; err != nil {
			handledErr = &handlerError{status: http.StatusBadRequest, code: "ASSET_NOT_FOUND", message: "imageAssetId not found"}
			return err
		}

		post := models.Post{
			ID:         uuid.NewString(),
			Type:       "RANK",
			Visibility: visibility,
			CreatorID:  authCtx.UserID,
			CategoryID: req.CategoryID,
			Caption:    req.Caption,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}

		rank := models.RankPost{
			PostID:       post.ID,
			TemplateID:   req.TemplateID,
			TierKey:      req.TierKey,
			ImageAssetID: req.ImageAssetID,
			SubjectTitle: req.SubjectTitle,
			SubjectURL:   req.SubjectURL,
		}
		if err := tx.Create(&rank).Error; err != nil {
			return err
		}

		metrics := models.PostMetrics{
			PostID:    post.ID,
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(&metrics).Error; err != nil {
			return err
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{"ranks_created_count": gorm.Expr("\"user_stats\".\"ranks_created_count\" + EXCLUDED.\"ranks_created_count\"")}),
		}).Create(&models.UserStats{
			ID:                uuid.NewString(),
			UserID:            authCtx.UserID,
			RanksCreatedCount: 1,
			UpdatedAt:         time.Now(),
		}).Error; err != nil {
			return err
		}

		if err := applyPostPreloads(tx).Where("posts.id = ?", post.ID).First(&created).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if handledErr != nil {
			c.JSON(handledErr.status, gin.H{"code": handledErr.code, "message": handledErr.message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": "failed to create rank"})
		return
	}

	c.JSON(http.StatusCreated, toPostView(created))
}

type handlerError struct {
	status  int
	code    string
	message string
}

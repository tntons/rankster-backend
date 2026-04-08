package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/models"
)

type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

func (h *SearchHandler) SearchCategories(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "q is required"})
		return
	}
	limit := parseIntWithDefault(c.Query("limit"), 20)
	if limit < 1 {
		limit = 1
	}
	if limit > 50 {
		limit = 50
	}

	var categories []models.Category
	if err := h.db.
		Where("name ILIKE ? OR slug ILIKE ? OR ? = ANY(tags)", "%"+q+"%", "%"+q+"%", q).
		Order("name asc").
		Limit(limit).
		Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": "failed to search categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": categories})
}


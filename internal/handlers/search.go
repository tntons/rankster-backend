package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) SearchOverview(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	response, err := h.searchService.Search(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to search"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetTrendingTopics(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	items, err := h.searchService.TrendingTopics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load topics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) GetCategories(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	items, err := h.searchService.Categories(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) GetLeaderboard(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	timeframe := strings.TrimSpace(strings.ToLower(c.DefaultQuery("timeframe", "this-week")))
	category := strings.TrimSpace(strings.ToLower(c.DefaultQuery("category", "all")))

	items, err := h.leaderboardService.Leaderboard(timeframe, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load leaderboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

package handlers

import (
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"net/http"

	"rankster-backend/internal/models"
)

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

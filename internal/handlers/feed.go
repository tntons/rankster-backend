package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) GetMainFeed(c *gin.Context) {
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
	response, err := h.feedService.MainFeed(scope, offset, limit, authUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load feed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

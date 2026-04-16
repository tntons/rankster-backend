package handlers

import (
	"fmt"
	"strings"

	"encoding/base64"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) GetMainFeed(c *gin.Context) {
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
	var (
		lists      []models.TierListPost
		nextCursor any
		err        error
	)

	switch scope {
	case "following":
		if authUser == nil {
			c.JSON(http.StatusOK, frontendFeedResponse{Items: []frontendRankPostView{}, NextCursor: nil})
			return
		}
		lists, nextCursor, err = h.followingFeedTierLists(authUser.ID, offset, limit)
	default:
		lists, nextCursor, err = h.feedTierLists(offset, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load feed"})
		return
	}

	items, err := h.hydrateTierLists(lists, authUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to build feed"})
		return
	}

	c.JSON(http.StatusOK, frontendFeedResponse{Items: items, NextCursor: nextCursor})
}

func (h *FrontendHandler) feedTierLists(offset, limit int) ([]models.TierListPost, any, error) {
	if offset < 0 {
		offset = 0
	}

	var lists []models.TierListPost
	err := h.db.
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("created_at desc").
		Offset(offset).
		Limit(limit + 1).
		Find(&lists).Error
	if err != nil {
		return nil, nil, err
	}

	var nextCursor any
	if len(lists) > limit {
		lists = lists[:limit]
		nextCursor = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset+limit)))
	}
	return lists, nextCursor, nil
}

func (h *FrontendHandler) followingFeedTierLists(userID string, offset, limit int) ([]models.TierListPost, any, error) {
	if offset < 0 {
		offset = 0
	}

	var lists []models.TierListPost
	err := h.db.
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Joins("JOIN follows ON follows.following_id = posts.creator_id").
		Where("follows.follower_id = ?", userID).
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("tier_list_posts.created_at desc").
		Offset(offset).
		Limit(limit + 1).
		Find(&lists).Error
	if err != nil {
		return nil, nil, err
	}

	var nextCursor any
	if len(lists) > limit {
		lists = lists[:limit]
		nextCursor = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset+limit)))
	}
	return lists, nextCursor, nil
}

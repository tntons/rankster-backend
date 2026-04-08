package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetStats(c *gin.Context) {
	authCtx := auth.FromAuthorization(c.GetHeader("Authorization"))
	if authCtx.Kind != "user" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Missing bearer token"})
		return
	}

	var subscription models.Subscription
	if err := h.db.Where("user_id = ? AND status = ? AND plan IN ?", authCtx.UserID, "ACTIVE", []string{"PRO", "BUSINESS"}).
		Select("id").
		First(&subscription).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Subscription required"})
		return
	}

	var ranksCreated int64
	_ = h.db.Model(&models.Post{}).Where("creator_id = ? AND type = ?", authCtx.UserID, "RANK").Count(&ranksCreated).Error

	var likesReceived int64
	_ = h.db.Model(&models.PostMetrics{}).
		Select("COALESCE(SUM(like_count),0)").
		Joins("JOIN posts ON posts.id = post_metrics.post_id").
		Where("posts.creator_id = ?", authCtx.UserID).
		Scan(&likesReceived).Error

	var commentsReceived int64
	_ = h.db.Model(&models.Comment{}).
		Joins("JOIN posts ON posts.id = comments.post_id").
		Where("posts.creator_id = ?", authCtx.UserID).
		Count(&commentsReceived).Error

	var followerCount int64
	_ = h.db.Model(&models.Follow{}).Where("following_id = ?", authCtx.UserID).Count(&followerCount).Error

	var followingCount int64
	_ = h.db.Model(&models.Follow{}).Where("follower_id = ?", authCtx.UserID).Count(&followingCount).Error

	type rankRow struct {
		CategoryID string
		TierKey    string
	}
	var rankRows []rankRow
	_ = h.db.Table("rank_posts").
		Select("rank_posts.tier_key as tier_key, posts.category_id as category_id").
		Joins("JOIN posts ON posts.id = rank_posts.post_id").
		Where("posts.creator_id = ?", authCtx.UserID).
		Scan(&rankRows).Error

	type agg struct {
		TotalScore int
		SampleSize int
	}
	byCategoryMap := map[string]agg{}
	for _, row := range rankRows {
		score := tierKeyToScore(row.TierKey)
		if score <= 0 {
			continue
		}
		current := byCategoryMap[row.CategoryID]
		current.TotalScore += score
		current.SampleSize++
		byCategoryMap[row.CategoryID] = current
	}

	byCategory := make([]gin.H, 0, len(byCategoryMap))
	for categoryID, v := range byCategoryMap {
		avg := 0.0
		if v.SampleSize > 0 {
			avg = float64(v.TotalScore) / float64(v.SampleSize)
		}
		byCategory = append(byCategory, gin.H{
			"categoryId":       categoryID,
			"averageTierScore": avg,
			"sampleSize":       v.SampleSize,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"userId": authCtx.UserID,
		"totals": gin.H{
			"ranksCreated":     ranksCreated,
			"likesReceived":    likesReceived,
			"commentsReceived": commentsReceived,
		},
		"byCategory": byCategory,
		"engagement": gin.H{
			"followerCount":  followerCount,
			"followingCount": followingCount,
		},
	})
}

func tierKeyToScore(tierKey string) int {
	switch tierKey {
	case "S":
		return 5
	case "A":
		return 4
	case "B":
		return 3
	case "C":
		return 2
	case "D":
		return 1
	default:
		return 0
	}
}

package handlers

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/services"
)

func (h *Handler) GetPost(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	authUser := h.optionalUser(c)
	post, err := h.postByID(c.Param("id"), authUser)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load post"})
		return
	}
	c.JSON(http.StatusOK, post)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body createRankRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "invalid update payload"})
		return
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Category) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "title and category are required"})
		return
	}

	post, err := h.updateRankPost(user, c.Param("id"), body)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		if errors.Is(err, errForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "you can only edit your own post"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to update post"})
		return
	}
	c.JSON(http.StatusOK, post)
}

func (h *Handler) DeletePost(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	if err := h.deleteRankPost(user.ID, c.Param("id")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		if errors.Is(err, errForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "you can only delete your own post"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to delete post"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) PostComment(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body createCommentRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "comment text is required"})
		return
	}

	comment, notificationRecipientID, notification, err := h.createComment(user, c.Param("id"), strings.TrimSpace(body.Text))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to create comment"})
		return
	}

	if notification != nil {
		h.broadcastNotification(notificationRecipientID, *notification)
	}

	c.JSON(http.StatusCreated, comment)
}

func (h *Handler) LikeComment(c *gin.Context) {
	h.setCommentLike(c, true)
}

func (h *Handler) UnlikeComment(c *gin.Context) {
	h.setCommentLike(c, false)
}

func (h *Handler) CreateRank(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body createRankRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "invalid create payload"})
		return
	}

	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Category) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "title and category are required"})
		return
	}

	post, err := h.createRank(user, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to create rank"})
		return
	}
	c.JSON(http.StatusCreated, post)
}

func (h *Handler) GetUserStats(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	stats, err := h.userStats(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId": user.ID,
		"totals": gin.H{
			"ranksCreated":     stats.RanksCreated,
			"likesReceived":    stats.LikesReceived,
			"commentsReceived": stats.CommentsReceived,
		},
		"engagement": gin.H{
			"followerCount":  stats.Followers,
			"followingCount": stats.Following,
		},
	})
}

func (h *Handler) postByID(postID string, authUser *userView) (rankPostView, error) {
	return h.rankPostService.GetPost(postID, authUser)
}

func (h *Handler) createRank(user userView, body createRankRequest) (rankPostView, error) {
	return h.rankPostService.CreateRank(user, body)
}

func (h *Handler) updateRankPost(user userView, postID string, body createRankRequest) (rankPostView, error) {
	return h.rankPostService.UpdateRankPost(user, postID, body)
}

func (h *Handler) deleteRankPost(userID string, postID string) error {
	return h.rankPostService.DeleteRankPost(userID, postID)
}

func (h *Handler) createComment(user userView, postID string, text string) (commentView, string, *notificationView, error) {
	result, err := h.rankPostService.CreateComment(user, postID, text)
	return result.Comment, result.NotificationRecipientID, result.Notification, err
}

func (h *Handler) setCommentLike(c *gin.Context, liked bool) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	response, err := h.updateCommentLike(c.Param("id"), user.ID, liked)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "COMMENT_NOT_FOUND", "message": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to update comment like"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) updateCommentLike(commentID string, userID string, liked bool) (commentLikeResponse, error) {
	return h.rankPostService.UpdateCommentLike(commentID, userID, liked)
}

type computedUserStats = services.ComputedUserStats

func (h *Handler) userStats(userID string) (computedUserStats, error) {
	return h.rankPostService.UserStats(userID)
}

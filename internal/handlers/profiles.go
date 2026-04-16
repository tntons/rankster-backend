package handlers

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func (h *Handler) GetProfileMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	profile, err := h.profileService.BuildProfile(user.ID, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateProfileMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body updateProfileRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "invalid profile payload"})
		return
	}

	displayName := strings.TrimSpace(body.DisplayName)
	bio := strings.TrimSpace(body.Bio)
	avatar := strings.TrimSpace(body.Avatar)
	if displayName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "display name is required"})
		return
	}
	if len(displayName) > 40 || len(bio) > 160 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "profile fields are too long"})
		return
	}

	if err := h.profileService.UpdateCurrentProfile(user.ID, displayName, bio, avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to update profile"})
		return
	}

	profile, err := h.profileService.BuildProfile(user.ID, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) GetProfileByUsername(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	authUser := h.optionalUser(c)
	userRecord, err := h.lookupUserByUsername(c.Param("username"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	profile, err := h.profileService.BuildProfile(userRecord.ID, authUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) FollowProfileUser(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	targetUser, err := h.lookupUserByUsername(c.Param("username"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to follow user"})
		return
	}

	result, err := h.profileService.Follow(authUser, targetUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to follow user"})
		return
	}
	if result.Notification != nil {
		h.broadcastNotification(result.NotificationRecipientID, *result.Notification)
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": true})
}

func (h *Handler) UnfollowProfileUser(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	targetUser, err := h.lookupUserByUsername(c.Param("username"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unfollow user"})
		return
	}

	if _, err := h.profileService.Unfollow(authUser.ID, targetUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unfollow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": false})
}

func (h *Handler) PinProfilePost(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	postID := c.Param("postId")
	if err := h.profileService.SetPinnedPost(authUser.ID, postID, true); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to pin post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pinnedPostId": postID})
}

func (h *Handler) UnpinProfilePost(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	if err := h.profileService.SetPinnedPost(authUser.ID, c.Param("postId"), false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unpin post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pinnedPostId": nil})
}

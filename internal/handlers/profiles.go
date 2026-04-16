package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) GetProfileMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	profile, err := h.buildProfileResponse(user.ID, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *FrontendHandler) UpdateProfileMe(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body frontendUpdateProfileRequest
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

	if err := h.updateCurrentProfile(user.ID, displayName, bio, avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to update profile"})
		return
	}

	profile, err := h.buildProfileResponse(user.ID, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *FrontendHandler) GetProfileByUsername(c *gin.Context) {
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

	user := buildFrontendUser(userRecord)
	profile, err := h.buildProfileResponse(user.ID, authUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load profile"})
		return
	}

	profile.User = user
	c.JSON(http.StatusOK, profile)
}

func (h *FrontendHandler) FollowProfileUser(c *gin.Context) {
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

	changed, err := h.setFollowState(authUser.ID, targetUser.ID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to follow user"})
		return
	}
	var notification *frontendNotificationView
	if changed {
		notification, err = h.createNotification(h.db, targetUser.ID, &authUser.ID, "follow", "New follower", fmt.Sprintf("%s started following you.", authUser.DisplayName), "/profile/"+authUser.Username, time.Now())
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to create notification"})
		return
	}
	if notification != nil {
		h.broadcastNotification(targetUser.ID, *notification)
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": true})
}

func (h *FrontendHandler) UnfollowProfileUser(c *gin.Context) {
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

	if _, err := h.setFollowState(authUser.ID, targetUser.ID, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unfollow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isFollowing": false})
}

func (h *FrontendHandler) PinProfilePost(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	postID := c.Param("postId")
	if err := h.setPinnedPost(authUser.ID, postID, true); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "POST_NOT_FOUND", "message": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to pin post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pinnedPostId": postID})
}

func (h *FrontendHandler) UnpinProfilePost(c *gin.Context) {
	authUser, ok := h.requireUser(c)
	if !ok {
		return
	}

	if err := h.setPinnedPost(authUser.ID, c.Param("postId"), false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to unpin post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pinnedPostId": nil})
}

func (h *FrontendHandler) rankingsForCreator(creatorID string, authUser *frontendUserView) ([]frontendRankPostView, error) {
	var lists []models.TierListPost
	err := h.db.
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Where("posts.creator_id = ?", creatorID).
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("tier_list_posts.created_at desc").
		Find(&lists).Error
	if err != nil {
		return nil, err
	}
	return h.hydrateTierLists(lists, authUser)
}

func (h *FrontendHandler) buildProfileResponse(profileUserID string, authUser *frontendUserView) (frontendProfileResponse, error) {
	userRecord, err := h.lookupUserByID(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	user := buildFrontendUser(userRecord)
	rankings, err := h.rankingsForCreator(profileUserID, authUser)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	likedPosts, err := h.likedRankingsForUser(profileUserID, authUser)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	stats, err := h.userStats(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	favoriteCategories, err := h.favoriteCategoriesForUser(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	pinnedPostID, err := h.pinnedPostIDForUser(profileUserID)
	if err != nil {
		return frontendProfileResponse{}, err
	}

	isFollowing := false
	if authUser != nil && authUser.ID != profileUserID {
		isFollowing, err = h.followState(authUser.ID, profileUserID)
		if err != nil {
			return frontendProfileResponse{}, err
		}
	}

	return frontendProfileResponse{
		User:         user,
		Rankings:     rankings,
		LikedPosts:   likedPosts,
		PinnedPostID: pinnedPostID,
		Stats: frontendProfileStatsView{
			TotalRankings: stats.RanksCreated,
			Followers:     stats.Followers,
			Following:     stats.Following,
			TotalLikes:    stats.LikesReceived,
		},
		FavoriteCategories: favoriteCategories,
		IsFollowing:        isFollowing,
	}, nil
}

func (h *FrontendHandler) likedRankingsForUser(userID string, authUser *frontendUserView) ([]frontendRankPostView, error) {
	var lists []models.TierListPost
	err := h.db.
		Joins("JOIN post_likes ON post_likes.post_id = tier_list_posts.post_id").
		Where("post_likes.user_id = ?", userID).
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Order("post_likes.created_at desc").
		Find(&lists).Error
	if err != nil {
		return nil, err
	}

	return h.hydrateTierLists(lists, authUser)
}

func (h *FrontendHandler) favoriteCategoriesForUser(userID string) ([]frontendProfileCategoryView, error) {
	type categoryCountRow struct {
		ID    string
		Name  string
		Emoji string
		Count int
	}

	var rows []categoryCountRow
	err := h.db.Table("posts").
		Select("categories.slug AS id, categories.name AS name, COALESCE(categories.emoji, '') AS emoji, COUNT(*) AS count").
		Joins("JOIN categories ON categories.id = posts.category_id").
		Where("posts.creator_id = ?", userID).
		Group("categories.slug, categories.name, categories.emoji").
		Order("count DESC, categories.name ASC").
		Limit(4).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	total := 0
	for _, row := range rows {
		total += row.Count
	}

	out := make([]frontendProfileCategoryView, 0, len(rows))
	for _, row := range rows {
		pct := 0
		if total > 0 {
			pct = int(float64(row.Count) / float64(total) * 100)
		}
		out = append(out, frontendProfileCategoryView{
			ID:    row.ID,
			Name:  row.Name,
			Emoji: row.Emoji,
			Pct:   pct,
		})
	}
	return out, nil
}

func (h *FrontendHandler) pinnedPostIDForUser(userID string) (*string, error) {
	var pinned models.PinnedPost
	err := h.db.Where("user_id = ?", userID).Order("COALESCE(\"order\", 999999) asc, created_at asc").First(&pinned).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pinned.PostID, nil
}

func (h *FrontendHandler) followState(followerID, followingID string) (bool, error) {
	var count int64
	if err := h.db.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *FrontendHandler) updateCurrentProfile(userID string, displayName string, bio string, avatar string) error {
	updates := map[string]any{
		"display_name": displayName,
		"bio":          bio,
		"updated_at":   time.Now(),
	}
	if avatar == "" {
		updates["avatar_url"] = nil
	} else {
		updates["avatar_url"] = avatar
	}

	return h.db.Model(&models.UserProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (h *FrontendHandler) setFollowState(followerID, followingID string, shouldFollow bool) (bool, error) {
	if followerID == followingID {
		return false, nil
	}

	changed := false
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Follow
		err := tx.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if shouldFollow {
			if err == nil {
				return nil
			}
			changed = true
			follow := models.Follow{
				ID:          generateUUID(),
				FollowerID:  followerID,
				FollowingID: followingID,
				CreatedAt:   time.Now(),
			}
			if err := tx.Create(&follow).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.UserStats{}).Where("user_id = ?", followerID).
				Update("following_count", gorm.Expr("following_count + 1")).Error; err != nil {
				return err
			}
			return tx.Model(&models.UserStats{}).Where("user_id = ?", followingID).
				Update("followers_count", gorm.Expr("followers_count + 1")).Error
		}

		if err == gorm.ErrRecordNotFound {
			return nil
		}
		changed = true
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.UserStats{}).Where("user_id = ?", followerID).
			Update("following_count", gorm.Expr("GREATEST(following_count - 1, 0)")).Error; err != nil {
			return err
		}
		return tx.Model(&models.UserStats{}).Where("user_id = ?", followingID).
			Update("followers_count", gorm.Expr("GREATEST(followers_count - 1, 0)")).Error
	})
	return changed, err
}

func (h *FrontendHandler) setPinnedPost(userID, postID string, shouldPin bool) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.PinnedPost{}).Error; err != nil {
			return err
		}
		if !shouldPin {
			return nil
		}

		var post models.Post
		if err := tx.Where("id = ? AND creator_id = ?", postID, userID).First(&post).Error; err != nil {
			return err
		}

		pinned := models.PinnedPost{
			ID:        generateUUID(),
			UserID:    userID,
			PostID:    postID,
			Order:     intPtrValue(1),
			CreatedAt: time.Now(),
		}
		return tx.Create(&pinned).Error
	})
}

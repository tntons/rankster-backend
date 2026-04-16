package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) GetPost(c *gin.Context) {
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

func (h *FrontendHandler) UpdatePost(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body frontendCreateRankRequest
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

func (h *FrontendHandler) DeletePost(c *gin.Context) {
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

func (h *FrontendHandler) PostComment(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body frontendCreateCommentRequest
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

func (h *FrontendHandler) LikeComment(c *gin.Context) {
	h.setCommentLike(c, true)
}

func (h *FrontendHandler) UnlikeComment(c *gin.Context) {
	h.setCommentLike(c, false)
}

func (h *FrontendHandler) CreateRank(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body frontendCreateRankRequest
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

func (h *FrontendHandler) GetUserStats(c *gin.Context) {
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

func (h *FrontendHandler) postByID(postID string, authUser *frontendUserView) (frontendRankPostView, error) {
	list, err := h.lookupTierListPost(postID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		resolvedPostID, resolveErr := h.resolveTrendingTopicPostID(postID)
		if resolveErr != nil {
			return frontendRankPostView{}, resolveErr
		}
		if resolvedPostID != nil {
			list, err = h.lookupTierListPost(*resolvedPostID)
		}
	}
	if err != nil {
		return frontendRankPostView{}, err
	}

	items, err := h.hydrateTierLists([]models.TierListPost{list}, authUser)
	if err != nil {
		return frontendRankPostView{}, err
	}
	if len(items) == 0 {
		return frontendRankPostView{}, gorm.ErrRecordNotFound
	}
	return items[0], nil
}

func (h *FrontendHandler) lookupTierListPost(postID string) (models.TierListPost, error) {
	var list models.TierListPost
	err := h.db.
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") }).
		Where("post_id = ?", postID).
		First(&list).Error
	return list, err
}

func (h *FrontendHandler) resolveTrendingTopicPostID(topicID string) (*string, error) {
	var topic models.TrendingTopic
	err := h.db.Where("id = ?", topicID).First(&topic).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return topic.SourcePostID, nil
}

func (h *FrontendHandler) createRank(user frontendUserView, body frontendCreateRankRequest) (frontendRankPostView, error) {
	now := time.Now()
	postID := ""
	sourcePostID := strings.TrimSpace(body.SourcePostID)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		if sourcePostID != "" {
			var source models.TierListPost
			if err := tx.Where("post_id = ?", sourcePostID).First(&source).Error; err != nil {
				return err
			}
		}

		category, err := ensureCategory(tx, body.Category, now)
		if err != nil {
			return err
		}

		coverURL := fmt.Sprintf("http://localhost:8000/assets/ranks/%s.svg", slugify(body.Title))
		asset := models.Asset{ID: "", URL: ""}
		if err := tx.Where("url = ?", coverURL).First(&asset).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			asset = models.Asset{ID: generateUUID(), URL: coverURL, CreatedAt: now}
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
		}

		postID = generateUUID()
		visibility := "PUBLIC"
		if body.IsPublic != nil && !*body.IsPublic {
			visibility = "PRIVATE"
		}

		post := models.Post{
			ID:         postID,
			Type:       "RANK",
			Visibility: visibility,
			CreatorID:  user.ID,
			CategoryID: category.ID,
			Caption:    stringPtr(body.Description),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}

		tags := body.Tags
		if len(tags) == 0 {
			tags = []string{body.Category}
		}
		tierPost := models.TierListPost{
			PostID:           postID,
			Title:            body.Title,
			Description:      stringPtr(body.Description),
			CoverAssetID:     &asset.ID,
			Tags:             pq.StringArray(tags),
			ParticipantCount: max(1, len(body.AllItems)),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&tierPost).Error; err != nil {
			return err
		}

		tierLookup := map[string]struct {
			Key      string
			Position int
			Emoji    *string
			ImageURL *string
		}{}
		recordTierItems := func(key string, items []frontendTierItem) {
			for index, item := range items {
				tierLookup[item.ID] = struct {
					Key      string
					Position int
					Emoji    *string
					ImageURL *string
				}{Key: key, Position: index, Emoji: item.Emoji, ImageURL: item.ImageURL}
			}
		}
		recordTierItems("S", body.Tiers.S)
		recordTierItems("A", body.Tiers.A)
		recordTierItems("B", body.Tiers.B)
		recordTierItems("C", body.Tiers.C)
		recordTierItems("D", body.Tiers.D)

		for index, item := range body.AllItems {
			tierMeta := tierLookup[item.ID]
			entry := models.TierListItem{
				ID:             generateUUID(),
				TierListPostID: postID,
				ExternalID:     item.ID,
				Name:           item.Name,
				Emoji:          coalesceEmoji(item.Emoji, tierMeta.Emoji),
				ImageURL:       coalesceImageURL(item.ImageURL, tierMeta.ImageURL),
				TierKey:        tierMeta.Key,
				TierPosition:   tierMeta.Position,
				ListPosition:   index,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}

		metrics := models.PostMetrics{PostID: postID, UpdatedAt: now}
		if err := tx.Create(&metrics).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.UserStats{}).Where("user_id = ?", user.ID).
			Update("ranks_created_count", gorm.Expr("ranks_created_count + ?", 1)).Error; err != nil {
			return err
		}

		if sourcePostID != "" {
			if err := tx.Model(&models.TierListPost{}).
				Where("post_id = ?", sourcePostID).
				Update("participant_count", gorm.Expr("participant_count + ?", 1)).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.TrendingTopic{}).
				Where("source_post_id = ?", sourcePostID).
				Update("participant_count", gorm.Expr("participant_count + ?", 1)).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return frontendRankPostView{}, err
	}

	return h.postByID(postID, &user)
}

func (h *FrontendHandler) updateRankPost(user frontendUserView, postID string, body frontendCreateRankRequest) (frontendRankPostView, error) {
	now := time.Now()

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var list models.TierListPost
		if err := tx.
			Preload("Post").
			Where("post_id = ?", postID).
			First(&list).Error; err != nil {
			return err
		}
		if list.Post.CreatorID != user.ID {
			return errForbidden
		}

		category, err := ensureCategory(tx, body.Category, now)
		if err != nil {
			return err
		}

		coverURL := fmt.Sprintf("http://localhost:8000/assets/ranks/%s.svg", slugify(body.Title))
		asset := models.Asset{ID: "", URL: ""}
		if err := tx.Where("url = ?", coverURL).First(&asset).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			asset = models.Asset{ID: generateUUID(), URL: coverURL, CreatedAt: now}
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
		}

		visibility := list.Post.Visibility
		if body.IsPublic != nil {
			if *body.IsPublic {
				visibility = "PUBLIC"
			} else {
				visibility = "PRIVATE"
			}
		}

		tags := body.Tags
		if len(tags) == 0 {
			tags = []string{body.Category}
		}

		if err := tx.Model(&models.Post{}).
			Where("id = ?", postID).
			Updates(map[string]any{
				"visibility":  visibility,
				"category_id": category.ID,
				"caption":     stringPtr(body.Description),
				"updated_at":  now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.TierListPost{}).
			Where("post_id = ?", postID).
			Updates(map[string]any{
				"title":          body.Title,
				"description":    stringPtr(body.Description),
				"cover_asset_id": &asset.ID,
				"tags":           pq.StringArray(tags),
				"updated_at":     now,
			}).Error; err != nil {
			return err
		}

		if len(body.AllItems) == 0 {
			return nil
		}

		if err := tx.Where("tier_list_post_id = ?", postID).Delete(&models.TierListItem{}).Error; err != nil {
			return err
		}

		tierLookup := map[string]struct {
			Key      string
			Position int
			Emoji    *string
			ImageURL *string
		}{}
		recordTierItems := func(key string, items []frontendTierItem) {
			for index, item := range items {
				tierLookup[item.ID] = struct {
					Key      string
					Position int
					Emoji    *string
					ImageURL *string
				}{Key: key, Position: index, Emoji: item.Emoji, ImageURL: item.ImageURL}
			}
		}
		recordTierItems("S", body.Tiers.S)
		recordTierItems("A", body.Tiers.A)
		recordTierItems("B", body.Tiers.B)
		recordTierItems("C", body.Tiers.C)
		recordTierItems("D", body.Tiers.D)

		for index, item := range body.AllItems {
			tierMeta := tierLookup[item.ID]
			entry := models.TierListItem{
				ID:             generateUUID(),
				TierListPostID: postID,
				ExternalID:     item.ID,
				Name:           item.Name,
				Emoji:          coalesceEmoji(item.Emoji, tierMeta.Emoji),
				ImageURL:       coalesceImageURL(item.ImageURL, tierMeta.ImageURL),
				TierKey:        tierMeta.Key,
				TierPosition:   tierMeta.Position,
				ListPosition:   index,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return frontendRankPostView{}, err
	}

	return h.postByID(postID, &user)
}

func (h *FrontendHandler) deleteRankPost(userID string, postID string) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		var list models.TierListPost
		if err := tx.
			Preload("Post").
			Where("post_id = ?", postID).
			First(&list).Error; err != nil {
			return err
		}
		if list.Post.CreatorID != userID {
			return errForbidden
		}

		commentIDQuery := tx.Model(&models.Comment{}).Select("id").Where("post_id = ?", postID)
		if err := tx.Where("comment_id IN (?)", commentIDQuery).Delete(&models.CommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.PostLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.PostShare{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.PinnedPost{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tier_list_post_id = ?", postID).Delete(&models.TierListItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.PostMetrics{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.TrendingTopic{}).
			Where("source_post_id = ?", postID).
			Updates(map[string]any{"source_post_id": nil}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.TierListPost{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&models.RankPost{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Post{ID: postID}).Error; err != nil {
			return err
		}
		return tx.Model(&models.UserStats{}).Where("user_id = ?", userID).
			Update("ranks_created_count", gorm.Expr("GREATEST(ranks_created_count - 1, 0)")).Error
	})
}

func (h *FrontendHandler) createComment(user frontendUserView, postID string, text string) (frontendCommentView, string, *frontendNotificationView, error) {
	now := time.Now()
	var (
		comment                 frontendCommentView
		notification            *frontendNotificationView
		notificationRecipientID string
	)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var list models.TierListPost
		if err := tx.
			Preload("Post.Creator.Profile").
			Preload("Post.Creator.Stats").
			Where("post_id = ?", postID).
			First(&list).Error; err != nil {
			return err
		}

		created := models.Comment{
			ID:        generateUUID(),
			PostID:    postID,
			AuthorID:  user.ID,
			Body:      text,
			LikeCount: 0,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PostMetrics{}).
			Where("post_id = ?", postID).
			Updates(map[string]any{
				"comment_count": gorm.Expr("comment_count + ?", 1),
				"hot_score":     gorm.Expr("hot_score + ?", 1.5),
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}

		comment = frontendCommentView{
			ID:        created.ID,
			User:      user,
			Text:      created.Body,
			CreatedAt: relativeTime(created.CreatedAt),
			Likes:     created.LikeCount,
			IsLiked:   false,
		}

		if list.Post.CreatorID != user.ID {
			notificationRecipientID = list.Post.CreatorID
			var err error
			notification, err = h.createNotification(
				tx,
				list.Post.CreatorID,
				&user.ID,
				"comment",
				"New comment",
				fmt.Sprintf("%s commented on your ranking.", user.DisplayName),
				"/topic/"+postID,
				now,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
	return comment, notificationRecipientID, notification, err
}

func (h *FrontendHandler) setCommentLike(c *gin.Context, liked bool) {
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

func (h *FrontendHandler) updateCommentLike(commentID string, userID string, liked bool) (frontendCommentLikeResponse, error) {
	var response frontendCommentLikeResponse
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var comment models.Comment
		if err := tx.Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}

		var existing models.CommentLike
		err := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).First(&existing).Error
		if liked {
			if err == nil {
				response = frontendCommentLikeResponse{Likes: comment.LikeCount, IsLiked: true}
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&models.CommentLike{
				ID:        generateUUID(),
				CommentID: commentID,
				UserID:    userID,
				CreatedAt: time.Now(),
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Comment{}).
				Where("id = ?", commentID).
				Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
				return err
			}
			comment.LikeCount++
			response = frontendCommentLikeResponse{Likes: comment.LikeCount, IsLiked: true}
			return nil
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			response = frontendCommentLikeResponse{Likes: comment.LikeCount, IsLiked: false}
			return nil
		}
		if err != nil {
			return err
		}
		result := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).Delete(&models.CommentLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if err := tx.Model(&models.Comment{}).
				Where("id = ?", commentID).
				Update("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
				return err
			}
			if comment.LikeCount > 0 {
				comment.LikeCount--
			}
		}
		response = frontendCommentLikeResponse{Likes: comment.LikeCount, IsLiked: false}
		return nil
	})
	return response, err
}

type computedUserStats struct {
	RanksCreated     int
	LikesReceived    int
	CommentsReceived int
	Followers        int
	Following        int
}

func (h *FrontendHandler) userStats(userID string) (computedUserStats, error) {
	var user models.User
	if err := h.db.Preload("Stats").Where("id = ?", userID).First(&user).Error; err != nil {
		return computedUserStats{}, err
	}

	var likesReceived int64
	if err := h.db.Model(&models.PostMetrics{}).
		Joins("JOIN posts ON posts.id = post_metrics.post_id").
		Where("posts.creator_id = ?", userID).
		Select("COALESCE(SUM(post_metrics.like_count), 0)").
		Scan(&likesReceived).Error; err != nil {
		return computedUserStats{}, err
	}

	var commentsReceived int64
	if err := h.db.Model(&models.Comment{}).
		Joins("JOIN posts ON posts.id = comments.post_id").
		Where("posts.creator_id = ?", userID).
		Count(&commentsReceived).Error; err != nil {
		return computedUserStats{}, err
	}

	stats := computedUserStats{
		LikesReceived:    int(likesReceived),
		CommentsReceived: int(commentsReceived),
	}
	if user.Stats != nil {
		stats.RanksCreated = user.Stats.RanksCreatedCount
		stats.Followers = user.Stats.FollowersCount
		stats.Following = user.Stats.FollowingCount
	}
	return stats, nil
}

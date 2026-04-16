package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type RankPostService struct {
	db            *gorm.DB
	tierLists     *repositories.TierListRepository
	interactions  *repositories.InteractionRepository
	notifications *NotificationService
}

type CreatedComment struct {
	Comment                 views.Comment
	NotificationRecipientID string
	Notification            *views.Notification
}

func NewRankPostService(
	db *gorm.DB,
	tierLists *repositories.TierListRepository,
	interactions *repositories.InteractionRepository,
	notifications *NotificationService,
) *RankPostService {
	return &RankPostService{
		db:            db,
		tierLists:     tierLists,
		interactions:  interactions,
		notifications: notifications,
	}
}

func (s *RankPostService) HydrateTierLists(lists []models.TierListPost, authUser *views.User) ([]views.RankPost, error) {
	postIDs := make([]string, 0, len(lists))
	for _, list := range lists {
		postIDs = append(postIDs, list.PostID)
	}

	commentsByPost, err := s.commentsByPost(postIDs, authUser)
	if err != nil {
		return nil, err
	}

	var authUserID string
	if authUser != nil {
		authUserID = authUser.ID
	}
	likedByPost, err := s.interactions.LikedPostIDs(authUserID, postIDs)
	if err != nil {
		return nil, err
	}

	items := make([]views.RankPost, 0, len(lists))
	for _, list := range lists {
		canEdit := authUser != nil && list.Post.CreatorID == authUser.ID
		items = append(items, views.BuildRankPost(list, commentsByPost[list.PostID], likedByPost[list.PostID], canEdit))
	}
	return items, nil
}

func (s *RankPostService) GetPost(postID string, authUser *views.User) (views.RankPost, error) {
	list, err := s.tierLists.FindByPostID(postID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		resolvedPostID, resolveErr := s.tierLists.SourcePostIDForTopic(postID)
		if resolveErr != nil {
			return views.RankPost{}, resolveErr
		}
		if resolvedPostID != nil {
			list, err = s.tierLists.FindByPostID(*resolvedPostID)
		}
	}
	if err != nil {
		return views.RankPost{}, err
	}

	items, err := s.HydrateTierLists([]models.TierListPost{list}, authUser)
	if err != nil {
		return views.RankPost{}, err
	}
	if len(items) == 0 {
		return views.RankPost{}, gorm.ErrRecordNotFound
	}
	return items[0], nil
}

func (s *RankPostService) RankingsForCreator(creatorID string, authUser *views.User) ([]views.RankPost, error) {
	lists, err := s.tierLists.ByCreator(creatorID)
	if err != nil {
		return nil, err
	}
	return s.HydrateTierLists(lists, authUser)
}

func (s *RankPostService) LikedRankingsForUser(userID string, authUser *views.User) ([]views.RankPost, error) {
	lists, err := s.tierLists.LikedByUser(userID)
	if err != nil {
		return nil, err
	}
	return s.HydrateTierLists(lists, authUser)
}

func (s *RankPostService) CreateRank(user views.User, body views.CreateRankRequest) (views.RankPost, error) {
	now := time.Now()
	postID := ""
	sourcePostID := strings.TrimSpace(body.SourcePostID)

	err := s.db.Transaction(func(tx *gorm.DB) error {
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

		coverURL := rankCoverURL(body.Title)
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

		if err := createTierListItems(tx, postID, body.Tiers, body.AllItems, now); err != nil {
			return err
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
		return views.RankPost{}, err
	}

	return s.GetPost(postID, &user)
}

func (s *RankPostService) UpdateRankPost(user views.User, postID string, body views.CreateRankRequest) (views.RankPost, error) {
	now := time.Now()

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var list models.TierListPost
		if err := tx.
			Preload("Post").
			Where("post_id = ?", postID).
			First(&list).Error; err != nil {
			return err
		}
		if list.Post.CreatorID != user.ID {
			return ErrForbidden
		}

		category, err := ensureCategory(tx, body.Category, now)
		if err != nil {
			return err
		}

		coverURL := rankCoverURL(body.Title)
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

		return createTierListItems(tx, postID, body.Tiers, body.AllItems, now)
	})
	if err != nil {
		return views.RankPost{}, err
	}

	return s.GetPost(postID, &user)
}

func (s *RankPostService) DeleteRankPost(userID string, postID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var list models.TierListPost
		if err := tx.
			Preload("Post").
			Where("post_id = ?", postID).
			First(&list).Error; err != nil {
			return err
		}
		if list.Post.CreatorID != userID {
			return ErrForbidden
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

func (s *RankPostService) CreateComment(user views.User, postID string, text string) (CreatedComment, error) {
	now := time.Now()
	var result CreatedComment

	err := s.db.Transaction(func(tx *gorm.DB) error {
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

		result.Comment = views.Comment{
			ID:        created.ID,
			User:      user,
			Text:      created.Body,
			CreatedAt: views.RelativeTime(created.CreatedAt),
			Likes:     created.LikeCount,
			IsLiked:   false,
		}

		if list.Post.CreatorID != user.ID && s.notifications != nil {
			result.NotificationRecipientID = list.Post.CreatorID
			notification, err := s.notifications.Create(
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
			result.Notification = notification
		}

		return nil
	})
	return result, err
}

func (s *RankPostService) UpdateCommentLike(commentID string, userID string, liked bool) (views.CommentLikeResponse, error) {
	var response views.CommentLikeResponse
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var comment models.Comment
		if err := tx.Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}

		var existing models.CommentLike
		err := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).First(&existing).Error
		if liked {
			if err == nil {
				response = views.CommentLikeResponse{Likes: comment.LikeCount, IsLiked: true}
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
			response = views.CommentLikeResponse{Likes: comment.LikeCount, IsLiked: true}
			return nil
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			response = views.CommentLikeResponse{Likes: comment.LikeCount, IsLiked: false}
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
		response = views.CommentLikeResponse{Likes: comment.LikeCount, IsLiked: false}
		return nil
	})
	return response, err
}

func (s *RankPostService) UserStats(userID string) (ComputedUserStats, error) {
	var user models.User
	if err := s.db.Preload("Stats").Where("id = ?", userID).First(&user).Error; err != nil {
		return ComputedUserStats{}, err
	}

	var likesReceived int64
	if err := s.db.Model(&models.PostMetrics{}).
		Joins("JOIN posts ON posts.id = post_metrics.post_id").
		Where("posts.creator_id = ?", userID).
		Select("COALESCE(SUM(post_metrics.like_count), 0)").
		Scan(&likesReceived).Error; err != nil {
		return ComputedUserStats{}, err
	}

	var commentsReceived int64
	if err := s.db.Model(&models.Comment{}).
		Joins("JOIN posts ON posts.id = comments.post_id").
		Where("posts.creator_id = ?", userID).
		Count(&commentsReceived).Error; err != nil {
		return ComputedUserStats{}, err
	}

	stats := ComputedUserStats{
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

func (s *RankPostService) commentsByPost(postIDs []string, authUser *views.User) (map[string][]views.Comment, error) {
	out := map[string][]views.Comment{}
	if len(postIDs) == 0 {
		return out, nil
	}

	comments, err := s.interactions.CommentsByPostIDs(postIDs)
	if err != nil {
		return nil, err
	}

	var authUserID string
	if authUser != nil {
		authUserID = authUser.ID
	}
	commentIDs := make([]string, 0, len(comments))
	for _, comment := range comments {
		commentIDs = append(commentIDs, comment.ID)
	}
	likedByComment, err := s.interactions.LikedCommentIDs(authUserID, commentIDs)
	if err != nil {
		return nil, err
	}

	for _, comment := range comments {
		out[comment.PostID] = append(out[comment.PostID], views.Comment{
			ID:        comment.ID,
			User:      views.BuildUser(comment.Author),
			Text:      comment.Body,
			CreatedAt: views.RelativeTime(comment.CreatedAt),
			Likes:     comment.LikeCount,
			IsLiked:   likedByComment[comment.ID],
		})
	}
	return out, nil
}

func createTierListItems(tx *gorm.DB, postID string, tiers views.TierData, allItems []views.TierItem, now time.Time) error {
	tierLookup := map[string]struct {
		Key      string
		Position int
		Emoji    *string
		ImageURL *string
	}{}
	recordTierItems := func(key string, items []views.TierItem) {
		for index, item := range items {
			tierLookup[item.ID] = struct {
				Key      string
				Position int
				Emoji    *string
				ImageURL *string
			}{Key: key, Position: index, Emoji: item.Emoji, ImageURL: item.ImageURL}
		}
	}
	recordTierItems("S", tiers.S)
	recordTierItems("A", tiers.A)
	recordTierItems("B", tiers.B)
	recordTierItems("C", tiers.C)
	recordTierItems("D", tiers.D)

	for index, item := range allItems {
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
}

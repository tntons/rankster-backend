package repositories

import (
	"errors"

	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type TierListRepository struct {
	db *gorm.DB
}

func NewTierListRepository(db *gorm.DB) *TierListRepository {
	return &TierListRepository{db: db}
}

func (r *TierListRepository) Feed(offset, limit int) ([]models.TierListPost, bool, error) {
	if offset < 0 {
		offset = 0
	}

	var lists []models.TierListPost
	err := preloadTierList(r.db).
		Order("created_at desc").
		Offset(offset).
		Limit(limit + 1).
		Find(&lists).Error
	if err != nil {
		return nil, false, err
	}

	hasMore := len(lists) > limit
	if hasMore {
		lists = lists[:limit]
	}
	return lists, hasMore, nil
}

func (r *TierListRepository) FollowingFeed(userID string, offset, limit int) ([]models.TierListPost, bool, error) {
	if offset < 0 {
		offset = 0
	}

	var lists []models.TierListPost
	err := preloadTierList(r.db).
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Joins("JOIN follows ON follows.following_id = posts.creator_id").
		Where("follows.follower_id = ?", userID).
		Order("tier_list_posts.created_at desc").
		Offset(offset).
		Limit(limit + 1).
		Find(&lists).Error
	if err != nil {
		return nil, false, err
	}

	hasMore := len(lists) > limit
	if hasMore {
		lists = lists[:limit]
	}
	return lists, hasMore, nil
}

func (r *TierListRepository) FindByPostID(postID string) (models.TierListPost, error) {
	var list models.TierListPost
	err := preloadTierList(r.db).
		Where("post_id = ?", postID).
		First(&list).Error
	return list, err
}

func (r *TierListRepository) SourcePostIDForTopic(topicID string) (*string, error) {
	var topic models.TrendingTopic
	err := r.db.Where("id = ?", topicID).First(&topic).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return topic.SourcePostID, nil
}

func (r *TierListRepository) ByCreator(creatorID string) ([]models.TierListPost, error) {
	var lists []models.TierListPost
	err := preloadTierList(r.db).
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Where("posts.creator_id = ?", creatorID).
		Order("tier_list_posts.created_at desc").
		Find(&lists).Error
	return lists, err
}

func (r *TierListRepository) LikedByUser(userID string) ([]models.TierListPost, error) {
	var lists []models.TierListPost
	err := preloadTierList(r.db).
		Joins("JOIN post_likes ON post_likes.post_id = tier_list_posts.post_id").
		Where("post_likes.user_id = ?", userID).
		Order("post_likes.created_at desc").
		Find(&lists).Error
	return lists, err
}

func preloadTierList(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Post.Creator.Profile").
		Preload("Post.Creator.Stats").
		Preload("Post.Category").
		Preload("Post.Metrics").
		Preload("CoverAsset").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("list_position asc") })
}

package repositories

import (
	"errors"
	"time"

	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type ProfileRepository struct {
	db *gorm.DB
}

type FavoriteCategoryRow struct {
	ID    string
	Name  string
	Emoji string
	Count int
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) FavoriteCategories(userID string, limit int) ([]FavoriteCategoryRow, error) {
	var rows []FavoriteCategoryRow
	err := r.db.Table("posts").
		Select("categories.slug AS id, categories.name AS name, COALESCE(categories.emoji, '') AS emoji, COUNT(*) AS count").
		Joins("JOIN categories ON categories.id = posts.category_id").
		Where("posts.creator_id = ?", userID).
		Group("categories.slug, categories.name, categories.emoji").
		Order("count DESC, categories.name ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *ProfileRepository) PinnedPostID(userID string) (*string, error) {
	var pinned models.PinnedPost
	err := r.db.Where("user_id = ?", userID).Order("COALESCE(\"order\", 999999) asc, created_at asc").First(&pinned).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pinned.PostID, nil
}

func (r *ProfileRepository) FollowState(followerID, followingID string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ProfileRepository) UpdateCurrentProfile(userID string, displayName string, bio string, avatar string) error {
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

	return r.db.Model(&models.UserProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (r *ProfileRepository) SetFollowState(followerID, followingID string, shouldFollow bool, idFactory func() string) (bool, error) {
	if followerID == followingID {
		return false, nil
	}

	changed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Follow
		err := tx.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if shouldFollow {
			if err == nil {
				return nil
			}
			changed = true
			follow := models.Follow{
				ID:          idFactory(),
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

		if errors.Is(err, gorm.ErrRecordNotFound) {
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

func (r *ProfileRepository) SetPinnedPost(userID, postID string, shouldPin bool, idFactory func() string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
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

		order := 1
		pinned := models.PinnedPost{
			ID:        idFactory(),
			UserID:    userID,
			PostID:    postID,
			Order:     &order,
			CreatedAt: time.Now(),
		}
		return tx.Create(&pinned).Error
	})
}

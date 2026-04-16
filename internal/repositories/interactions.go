package repositories

import (
	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type InteractionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

func (r *InteractionRepository) CommentsByPostIDs(postIDs []string) ([]models.Comment, error) {
	if len(postIDs) == 0 {
		return []models.Comment{}, nil
	}

	var comments []models.Comment
	err := r.db.
		Preload("Author.Profile").
		Preload("Author.Stats").
		Where("post_id IN ?", postIDs).
		Order("created_at desc").
		Find(&comments).Error
	return comments, err
}

func (r *InteractionRepository) LikedCommentIDs(userID string, commentIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if userID == "" || len(commentIDs) == 0 {
		return out, nil
	}

	var likes []models.CommentLike
	if err := r.db.Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, like := range likes {
		out[like.CommentID] = true
	}
	return out, nil
}

func (r *InteractionRepository) LikedPostIDs(userID string, postIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if userID == "" || len(postIDs) == 0 {
		return out, nil
	}

	var likes []models.PostLike
	if err := r.db.Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, like := range likes {
		out[like.PostID] = true
	}
	return out, nil
}

package repositories

import (
	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type SearchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

func (r *SearchRepository) Users(query string, limit int) ([]models.User, error) {
	var users []models.User
	db := r.db.Preload("Profile").Preload("Stats").Joins("JOIN user_profiles ON user_profiles.user_id = users.id")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(user_profiles.username) LIKE ? OR LOWER(COALESCE(user_profiles.display_name, '')) LIKE ? OR LOWER(COALESCE(user_profiles.bio, '')) LIKE ?", like, like, like)
	}
	err := db.Limit(limit).Find(&users).Error
	return users, err
}

func (r *SearchRepository) TrendingTopics(query string, limit int) ([]models.TrendingTopic, error) {
	var topics []models.TrendingTopic
	db := r.db.Preload("Category").Preload("CoverAsset").Preload("SourcePost").Order("participant_count desc")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(title) LIKE ? OR EXISTS (SELECT 1 FROM unnest(tags) tag WHERE LOWER(tag) LIKE ?)", like, like)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&topics).Error
	return topics, err
}

func (r *SearchRepository) Categories(query string) ([]models.Category, error) {
	var categories []models.Category
	db := r.db.Order("name asc")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ?", like, like)
	}
	err := db.Find(&categories).Error
	return categories, err
}

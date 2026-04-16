package repositories

import (
	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type LeaderboardRepository struct {
	db *gorm.DB
}

func NewLeaderboardRepository(db *gorm.DB) *LeaderboardRepository {
	return &LeaderboardRepository{db: db}
}

func (r *LeaderboardRepository) Entries() ([]models.LeaderboardEntry, error) {
	var entries []models.LeaderboardEntry
	err := r.db.
		Preload("User.Profile").
		Preload("User.Stats").
		Order("rank asc").
		Find(&entries).Error
	return entries, err
}

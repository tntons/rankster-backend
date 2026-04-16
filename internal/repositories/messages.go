package repositories

import (
	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) ThreadsForUser(userID string) ([]models.MessageThread, error) {
	var threads []models.MessageThread
	err := r.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Where("owner_user_id = ?", userID).
		Order("updated_at desc").
		Find(&threads).Error
	return threads, err
}

func (r *MessageRepository) ThreadForOwner(userID, threadID string) (models.MessageThread, error) {
	var thread models.MessageThread
	err := r.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Where("id = ? AND owner_user_id = ?", threadID, userID).
		First(&thread).Error
	return thread, err
}

func (r *MessageRepository) ThreadDetail(userID, threadID string) (models.MessageThread, error) {
	var thread models.MessageThread
	err := r.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Preload("Messages", func(db *gorm.DB) *gorm.DB { return db.Order("created_at asc") }).
		Where("id = ? AND owner_user_id = ?", threadID, userID).
		First(&thread).Error
	return thread, err
}

func (r *MessageRepository) MarkThreadRead(userID, threadID string) error {
	return r.db.Model(&models.MessageThread{}).
		Where("id = ? AND owner_user_id = ?", threadID, userID).
		Update("unread_count", 0).Error
}

func (r *MessageRepository) UnreadCount(userID string) (int, error) {
	var unreadCount int64
	if err := r.db.Model(&models.MessageThread{}).
		Where("owner_user_id = ?", userID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&unreadCount).Error; err != nil {
		return 0, err
	}
	return int(unreadCount), nil
}

package repositories

import (
	"time"

	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) List(userID string) ([]models.Notification, error) {
	var notifications []models.Notification
	err := r.db.
		Preload("ActorUser.Profile").
		Preload("ActorUser.Stats").
		Where("user_id = ? AND type <> ?", userID, "message").
		Order("created_at desc").
		Limit(50).
		Find(&notifications).Error
	return notifications, err
}

func (r *NotificationRepository) UnreadCount(userID string) (int, error) {
	var unreadCount int64
	if err := r.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL AND type <> ?", userID, "message").
		Count(&unreadCount).Error; err != nil {
		return 0, err
	}
	return int(unreadCount), nil
}

func (r *NotificationRepository) MarkRead(userID, notificationID string, readAt time.Time) error {
	result := r.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND type <> ?", notificationID, userID, "message").
		Update("read_at", &readAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(userID string, readAt time.Time) error {
	return r.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL AND type <> ?", userID, "message").
		Update("read_at", &readAt).Error
}

func (r *NotificationRepository) FindForUser(userID, notificationID string) (models.Notification, error) {
	var notification models.Notification
	err := r.db.
		Preload("ActorUser.Profile").
		Preload("ActorUser.Stats").
		Where("id = ? AND user_id = ? AND type <> ?", notificationID, userID, "message").
		First(&notification).Error
	return notification, err
}

func (r *NotificationRepository) FindByID(tx *gorm.DB, notificationID string) (models.Notification, error) {
	var notification models.Notification
	err := tx.
		Preload("ActorUser.Profile").
		Preload("ActorUser.Stats").
		Where("id = ?", notificationID).
		First(&notification).Error
	return notification, err
}

func (r *NotificationRepository) Create(tx *gorm.DB, notification models.Notification) error {
	return tx.Create(&notification).Error
}

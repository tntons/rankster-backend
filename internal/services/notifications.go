package services

import (
	"time"

	"gorm.io/gorm"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type NotificationService struct {
	notifications *repositories.NotificationRepository
}

func NewNotificationService(notifications *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

func (s *NotificationService) NotificationsForUser(userID string) (views.NotificationsResponse, error) {
	notifications, err := s.notifications.List(userID)
	if err != nil {
		return views.NotificationsResponse{}, err
	}

	unreadCount, err := s.notifications.UnreadCount(userID)
	if err != nil {
		return views.NotificationsResponse{}, err
	}

	items := make([]views.Notification, 0, len(notifications))
	for _, notification := range notifications {
		items = append(items, views.BuildNotification(notification))
	}

	return views.NotificationsResponse{
		Items:       items,
		UnreadCount: unreadCount,
	}, nil
}

func (s *NotificationService) UnreadCount(userID string) (int, error) {
	return s.notifications.UnreadCount(userID)
}

func (s *NotificationService) MarkNotificationRead(userID, notificationID string) (views.Notification, error) {
	now := time.Now()
	if err := s.notifications.MarkRead(userID, notificationID, now); err != nil {
		return views.Notification{}, err
	}

	notification, err := s.notifications.FindForUser(userID, notificationID)
	if err != nil {
		return views.Notification{}, err
	}

	return views.BuildNotification(notification), nil
}

func (s *NotificationService) MarkAllNotificationsRead(userID string) error {
	return s.notifications.MarkAllRead(userID, time.Now())
}

func (s *NotificationService) Create(
	tx *gorm.DB,
	userID string,
	actorUserID *string,
	notificationType string,
	title string,
	body string,
	actionHref string,
	createdAt time.Time,
) (*views.Notification, error) {
	if actorUserID != nil && *actorUserID == userID {
		return nil, nil
	}

	notification := models.Notification{
		ID:          generateUUID(),
		UserID:      userID,
		ActorUserID: actorUserID,
		Type:        notificationType,
		Title:       title,
		Body:        body,
		ActionHref:  actionHref,
		CreatedAt:   createdAt,
	}
	if err := s.notifications.Create(tx, notification); err != nil {
		return nil, err
	}
	if notificationType == "message" {
		return nil, nil
	}

	created, err := s.notifications.FindByID(tx, notification.ID)
	if err != nil {
		return nil, err
	}

	view := views.BuildNotification(created)
	return &view, nil
}

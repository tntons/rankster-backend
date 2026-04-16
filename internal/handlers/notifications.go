package handlers

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) GetNotifications(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	response, err := h.notificationsForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load notifications"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *FrontendHandler) MarkNotificationRead(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	notification, err := h.markNotificationRead(user.ID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOTIFICATION_NOT_FOUND", "message": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *FrontendHandler) MarkAllNotificationsRead(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	if err := h.markAllNotificationsRead(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to mark notifications as read"})
		return
	}

	response, err := h.notificationsForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load notifications"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *FrontendHandler) notificationsForUser(userID string) (frontendNotificationsResponse, error) {
	var notifications []models.Notification
	if err := h.db.
		Preload("ActorUser.Profile").
		Preload("ActorUser.Stats").
		Where("user_id = ? AND type <> ?", userID, "message").
		Order("created_at desc").
		Limit(50).
		Find(&notifications).Error; err != nil {
		return frontendNotificationsResponse{}, err
	}

	unreadCount, err := h.notificationUnreadCount(userID)
	if err != nil {
		return frontendNotificationsResponse{}, err
	}

	items := make([]frontendNotificationView, 0, len(notifications))
	for _, notification := range notifications {
		items = append(items, buildFrontendNotification(notification))
	}

	return frontendNotificationsResponse{
		Items:       items,
		UnreadCount: unreadCount,
	}, nil
}

func (h *FrontendHandler) notificationUnreadCount(userID string) (int, error) {
	var unreadCount int64
	if err := h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL AND type <> ?", userID, "message").
		Count(&unreadCount).Error; err != nil {
		return 0, err
	}
	return int(unreadCount), nil
}

func (h *FrontendHandler) markNotificationRead(userID, notificationID string) (frontendNotificationView, error) {
	now := time.Now()
	result := h.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND type <> ?", notificationID, userID, "message").
		Update("read_at", &now)
	if result.Error != nil {
		return frontendNotificationView{}, result.Error
	}
	if result.RowsAffected == 0 {
		return frontendNotificationView{}, gorm.ErrRecordNotFound
	}

	var notification models.Notification
	if err := h.db.
		Preload("ActorUser.Profile").
		Preload("ActorUser.Stats").
		Where("id = ? AND user_id = ? AND type <> ?", notificationID, userID, "message").
		First(&notification).Error; err != nil {
		return frontendNotificationView{}, err
	}

	return buildFrontendNotification(notification), nil
}

func (h *FrontendHandler) markAllNotificationsRead(userID string) error {
	now := time.Now()
	return h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL AND type <> ?", userID, "message").
		Update("read_at", &now).Error
}

func (h *FrontendHandler) createNotification(tx *gorm.DB, userID string, actorUserID *string, notificationType string, title string, body string, actionHref string, createdAt time.Time) (*frontendNotificationView, error) {
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
	if err := tx.Create(&notification).Error; err != nil {
		return nil, err
	}
	if notificationType == "message" {
		return nil, nil
	}

	var created models.Notification
	if err := tx.
		Preload("ActorUser.Profile").
		Preload("ActorUser.Stats").
		Where("id = ?", notification.ID).
		First(&created).Error; err != nil {
		return nil, err
	}

	view := buildFrontendNotification(created)
	return &view, nil
}

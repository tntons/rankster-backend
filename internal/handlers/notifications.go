package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func (h *Handler) GetNotifications(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	response, err := h.notificationService.NotificationsForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load notifications"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) MarkNotificationRead(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	notification, err := h.notificationService.MarkNotificationRead(user.ID, c.Param("id"))
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

func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	if err := h.notificationService.MarkAllNotificationsRead(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to mark notifications as read"})
		return
	}

	response, err := h.notificationService.NotificationsForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load notifications"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) notificationUnreadCount(userID string) (int, error) {
	return h.notificationService.UnreadCount(userID)
}

package handlers

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

func (h *Handler) GetMessages(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	items, err := h.messageService.ThreadsForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load messages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) GetMessageUnreadCount(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	unreadCount, err := h.messageService.UnreadCount(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load unread messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unreadCount": unreadCount})
}

func (h *Handler) GetMessageThread(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	thread, err := h.messageService.ThreadDetail(user.ID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "THREAD_NOT_FOUND", "message": "thread not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load conversation"})
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *Handler) PostMessage(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "message text is required"})
		return
	}

	created, err := h.messageService.CreateMessage(user.ID, c.Param("id"), strings.TrimSpace(body.Text))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "THREAD_NOT_FOUND", "message": "thread not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to send message"})
		return
	}

	h.broadcastCreatedMessage(c.Param("id"), created)
	c.JSON(http.StatusCreated, created.Sender)
}

func (h *Handler) messageUnreadCount(userID string) (int, error) {
	return h.messageService.UnreadCount(userID)
}

func (h *Handler) messageThreadDetail(userID, threadID string) (messageThreadDetailView, error) {
	return h.messageService.ThreadDetail(userID, threadID)
}

func (h *Handler) createMessage(userID, threadID, text string) (createdMessageView, error) {
	return h.messageService.CreateMessage(userID, threadID, text)
}

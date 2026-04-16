package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) GetMessages(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	items, err := h.messagesForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load messages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) GetMessageUnreadCount(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	unreadCount, err := h.messageUnreadCount(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load unread messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unreadCount": unreadCount})
}

func (h *FrontendHandler) GetMessageThread(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	thread, err := h.messageThreadDetail(user.ID, c.Param("id"))
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

func (h *FrontendHandler) PostMessage(c *gin.Context) {
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

	created, err := h.createMessage(user.ID, c.Param("id"), strings.TrimSpace(body.Text))
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

func (h *FrontendHandler) messagesForUser(userID string) ([]frontendMessageView, error) {
	var threads []models.MessageThread
	err := h.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Where("owner_user_id = ?", userID).
		Order("updated_at desc").
		Find(&threads).Error
	if err != nil {
		return nil, err
	}

	items := make([]frontendMessageView, 0, len(threads))
	for _, thread := range threads {
		items = append(items, frontendMessageView{
			ID:          thread.ID,
			User:        buildFrontendUser(thread.PeerUser),
			LastMessage: thread.LastMessage,
			Timestamp:   relativeTime(thread.UpdatedAt),
			Unread:      thread.UnreadCount,
		})
	}
	return items, nil
}

func (h *FrontendHandler) messageThreadViewByID(userID, threadID string) (frontendMessageView, error) {
	var thread models.MessageThread
	err := h.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Where("id = ? AND owner_user_id = ?", threadID, userID).
		First(&thread).Error
	if err != nil {
		return frontendMessageView{}, err
	}

	return frontendMessageView{
		ID:          thread.ID,
		User:        buildFrontendUser(thread.PeerUser),
		LastMessage: thread.LastMessage,
		Timestamp:   relativeTime(thread.UpdatedAt),
		Unread:      thread.UnreadCount,
	}, nil
}

func (h *FrontendHandler) messageUnreadCount(userID string) (int, error) {
	var unreadCount int64
	if err := h.db.Model(&models.MessageThread{}).
		Where("owner_user_id = ?", userID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&unreadCount).Error; err != nil {
		return 0, err
	}
	return int(unreadCount), nil
}

func (h *FrontendHandler) nextThreadUnreadCount(threadID string) any {
	if h.chatHub.hasSubscribers(threadID) {
		return 0
	}
	return gorm.Expr("unread_count + ?", 1)
}

func (h *FrontendHandler) messageThreadDetail(userID, threadID string) (frontendMessageThreadDetailView, error) {
	var thread models.MessageThread
	err := h.db.
		Preload("PeerUser.Profile").
		Preload("PeerUser.Stats").
		Preload("Messages", func(db *gorm.DB) *gorm.DB { return db.Order("created_at asc") }).
		Where("id = ? AND owner_user_id = ?", threadID, userID).
		First(&thread).Error
	if err != nil {
		return frontendMessageThreadDetailView{}, err
	}

	if thread.UnreadCount > 0 {
		if err := h.db.Model(&models.MessageThread{}).
			Where("id = ? AND owner_user_id = ?", threadID, userID).
			Update("unread_count", 0).Error; err != nil {
			return frontendMessageThreadDetailView{}, err
		}
	}

	messages := make([]frontendChatMessageView, 0, len(thread.Messages))
	for _, message := range thread.Messages {
		messages = append(messages, frontendChatMessageView{
			ID:        message.ID,
			Text:      message.Body,
			Mine:      message.SenderUserID == userID,
			Timestamp: chatTimestamp(message.CreatedAt),
		})
	}

	return frontendMessageThreadDetailView{
		ID:       thread.ID,
		User:     buildFrontendUser(thread.PeerUser),
		Messages: messages,
	}, nil
}

func (h *FrontendHandler) createMessage(userID, threadID, text string) (frontendCreatedMessage, error) {
	now := time.Now()
	messageID := generateUUID()
	var recipientThreadID *string
	var recipientUserID *string

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var thread models.MessageThread
		if err := tx.Where("id = ? AND owner_user_id = ?", threadID, userID).First(&thread).Error; err != nil {
			return err
		}
		peerUserID := thread.PeerUserID
		recipientUserID = &peerUserID

		message := models.DirectMessage{
			ID:           messageID,
			ThreadID:     thread.ID,
			SenderUserID: userID,
			Body:         text,
			CreatedAt:    now,
		}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.MessageThread{}).Where("id = ?", thread.ID).Updates(map[string]any{
			"last_message": text,
			"updated_at":   now,
			"unread_count": 0,
		}).Error; err != nil {
			return err
		}

		var peerThread models.MessageThread
		err := tx.Where("owner_user_id = ? AND peer_user_id = ?", thread.PeerUserID, userID).First(&peerThread).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			peerThread = models.MessageThread{
				ID:          generateUUID(),
				OwnerUserID: thread.PeerUserID,
				PeerUserID:  userID,
				LastMessage: text,
				UnreadCount: 1,
				UpdatedAt:   now,
				CreatedAt:   now,
			}
			if err := tx.Create(&peerThread).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&models.MessageThread{}).Where("id = ?", peerThread.ID).Updates(map[string]any{
			"last_message": text,
			"updated_at":   now,
			"unread_count": h.nextThreadUnreadCount(peerThread.ID),
		}).Error; err != nil {
			return err
		}

		recipientThreadID = &peerThread.ID
		peerMessage := models.DirectMessage{
			ID:           generateUUID(),
			ThreadID:     peerThread.ID,
			SenderUserID: userID,
			Body:         text,
			CreatedAt:    now,
		}
		if err := tx.Create(&peerMessage).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return frontendCreatedMessage{}, err
	}

	var recipientThread *frontendMessageView
	if recipientUserID != nil && recipientThreadID != nil {
		threadView, err := h.messageThreadViewByID(*recipientUserID, *recipientThreadID)
		if err != nil {
			return frontendCreatedMessage{}, err
		}
		recipientThread = &threadView
	}

	senderMessage := frontendChatMessageView{
		ID:        messageID,
		Text:      text,
		Mine:      true,
		Timestamp: "Now",
	}
	recipientMessage := frontendChatMessageView{
		ID:        messageID,
		Text:      text,
		Mine:      false,
		Timestamp: "Now",
	}

	return frontendCreatedMessage{
		Sender:            senderMessage,
		Recipient:         &recipientMessage,
		RecipientThreadID: recipientThreadID,
		RecipientUserID:   recipientUserID,
		RecipientThread:   recipientThread,
	}, nil
}

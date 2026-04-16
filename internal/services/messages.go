package services

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type MessageService struct {
	db                   *gorm.DB
	messages             *repositories.MessageRepository
	threadHasSubscribers func(threadID string) bool
}

func NewMessageService(db *gorm.DB, messages *repositories.MessageRepository, threadHasSubscribers func(threadID string) bool) *MessageService {
	return &MessageService{
		db:                   db,
		messages:             messages,
		threadHasSubscribers: threadHasSubscribers,
	}
}

func (s *MessageService) ThreadsForUser(userID string) ([]views.MessageThread, error) {
	threads, err := s.messages.ThreadsForUser(userID)
	if err != nil {
		return nil, err
	}

	items := make([]views.MessageThread, 0, len(threads))
	for _, thread := range threads {
		items = append(items, views.BuildMessageThread(thread))
	}
	return items, nil
}

func (s *MessageService) ThreadViewByID(userID, threadID string) (views.MessageThread, error) {
	thread, err := s.messages.ThreadForOwner(userID, threadID)
	if err != nil {
		return views.MessageThread{}, err
	}
	return views.BuildMessageThread(thread), nil
}

func (s *MessageService) UnreadCount(userID string) (int, error) {
	return s.messages.UnreadCount(userID)
}

func (s *MessageService) ThreadDetail(userID, threadID string) (views.MessageThreadDetail, error) {
	thread, err := s.messages.ThreadDetail(userID, threadID)
	if err != nil {
		return views.MessageThreadDetail{}, err
	}

	if thread.UnreadCount > 0 {
		if err := s.messages.MarkThreadRead(userID, threadID); err != nil {
			return views.MessageThreadDetail{}, err
		}
	}

	return views.BuildMessageThreadDetail(thread, userID), nil
}

func (s *MessageService) CreateMessage(userID, threadID, text string) (views.CreatedMessage, error) {
	now := time.Now()
	messageID := generateUUID()
	var recipientThreadID *string
	var recipientUserID *string

	err := s.db.Transaction(func(tx *gorm.DB) error {
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
			"unread_count": s.nextThreadUnreadCount(peerThread.ID),
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
		return views.CreatedMessage{}, err
	}

	var recipientThread *views.MessageThread
	if recipientUserID != nil && recipientThreadID != nil {
		threadView, err := s.ThreadViewByID(*recipientUserID, *recipientThreadID)
		if err != nil {
			return views.CreatedMessage{}, err
		}
		recipientThread = &threadView
	}

	senderMessage := views.NewChatMessage(messageID, text, true)
	recipientMessage := views.NewChatMessage(messageID, text, false)

	return views.CreatedMessage{
		Sender:            senderMessage,
		Recipient:         &recipientMessage,
		RecipientThreadID: recipientThreadID,
		RecipientUserID:   recipientUserID,
		RecipientThread:   recipientThread,
	}, nil
}

func (s *MessageService) nextThreadUnreadCount(threadID string) any {
	if s.threadHasSubscribers != nil && s.threadHasSubscribers(threadID) {
		return 0
	}
	return gorm.Expr("unread_count + ?", 1)
}

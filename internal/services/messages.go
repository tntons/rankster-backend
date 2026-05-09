package services

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

var (
	ErrInvalidMessagePeer = errors.New("message peer username is required")
	ErrCannotMessageSelf  = errors.New("cannot message yourself")
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

func (s *MessageService) StartThreadByUsername(userID, username string) (views.MessageThreadDetail, error) {
	normalizedUsername := strings.TrimPrefix(strings.TrimSpace(username), "@")
	if normalizedUsername == "" {
		return views.MessageThreadDetail{}, ErrInvalidMessagePeer
	}

	now := time.Now()
	var threadID string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var peer models.User
		if err := tx.
			Preload("Profile").
			Preload("Stats").
			Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
			Where("user_profiles.username = ?", normalizedUsername).
			First(&peer).Error; err != nil {
			return err
		}
		if peer.ID == userID {
			return ErrCannotMessageSelf
		}

		var thread models.MessageThread
		err := tx.Where("owner_user_id = ? AND peer_user_id = ?", userID, peer.ID).First(&thread).Error
		if err == nil {
			threadID = thread.ID
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		thread = models.MessageThread{
			ID:          generateUUID(),
			OwnerUserID: userID,
			PeerUserID:  peer.ID,
			LastMessage: "",
			UnreadCount: 0,
			UpdatedAt:   now,
			CreatedAt:   now,
		}
		if err := tx.Create(&thread).Error; err != nil {
			return err
		}
		threadID = thread.ID
		return nil
	})
	if err != nil {
		return views.MessageThreadDetail{}, err
	}

	return s.ThreadDetail(userID, threadID)
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

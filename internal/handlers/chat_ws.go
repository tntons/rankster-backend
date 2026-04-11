package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const chatSocketBufferSize = 16

func newFrontendChatHub() *frontendChatHub {
	return &frontendChatHub{
		clients: map[string]map[*frontendChatClient]struct{}{},
	}
}

func newFrontendMessageInboxHub() *frontendMessageInboxHub {
	return &frontendMessageInboxHub{
		clients: map[string]map[*frontendMessageInboxClient]struct{}{},
	}
}

func (hub *frontendMessageInboxHub) subscribe(userID string) (*frontendMessageInboxClient, func()) {
	client := &frontendMessageInboxClient{
		userID: userID,
		send:   make(chan frontendMessageInboxSocketEvent, chatSocketBufferSize),
	}

	hub.mu.Lock()
	if hub.clients[userID] == nil {
		hub.clients[userID] = map[*frontendMessageInboxClient]struct{}{}
	}
	hub.clients[userID][client] = struct{}{}
	hub.mu.Unlock()

	unsubscribe := func() {
		hub.mu.Lock()
		if clients := hub.clients[userID]; clients != nil {
			if _, ok := clients[client]; ok {
				delete(clients, client)
				close(client.send)
			}
			if len(clients) == 0 {
				delete(hub.clients, userID)
			}
		}
		hub.mu.Unlock()
	}

	return client, unsubscribe
}

func (hub *frontendMessageInboxHub) broadcast(userID string, event frontendMessageInboxSocketEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for client := range hub.clients[userID] {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (hub *frontendChatHub) subscribe(threadID string) (*frontendChatClient, func()) {
	client := &frontendChatClient{
		threadID: threadID,
		send:     make(chan frontendChatSocketEvent, chatSocketBufferSize),
	}

	hub.mu.Lock()
	if hub.clients[threadID] == nil {
		hub.clients[threadID] = map[*frontendChatClient]struct{}{}
	}
	hub.clients[threadID][client] = struct{}{}
	hub.mu.Unlock()

	unsubscribe := func() {
		hub.mu.Lock()
		if clients := hub.clients[threadID]; clients != nil {
			if _, ok := clients[client]; ok {
				delete(clients, client)
				close(client.send)
			}
			if len(clients) == 0 {
				delete(hub.clients, threadID)
			}
		}
		hub.mu.Unlock()
	}

	return client, unsubscribe
}

func (hub *frontendChatHub) broadcast(threadID string, event frontendChatSocketEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for client := range hub.clients[threadID] {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (hub *frontendChatHub) hasSubscribers(threadID string) bool {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.clients[threadID]) > 0
}

func (h *FrontendHandler) WebSocketMessageInbox(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.userFromSocketToken(c)
	if !ok {
		return
	}

	upgrader := frontendSocketUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client, unsubscribe := h.messageInboxHub.subscribe(user.ID)
	defer unsubscribe()

	go func() {
		for event := range client.send {
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		}
	}()

	unreadCount, err := h.messageUnreadCount(user.ID)
	if err != nil {
		unreadCount = 0
	}
	client.send <- frontendMessageInboxSocketEvent{
		Type:        "ready",
		UnreadCount: unreadCount,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *FrontendHandler) WebSocketMessageThread(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.userFromSocketToken(c)
	if !ok {
		return
	}

	threadID := c.Param("id")
	if _, err := h.messageThreadDetail(user.ID, threadID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "THREAD_NOT_FOUND", "message": "thread not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load conversation"})
		return
	}

	upgrader := frontendSocketUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client, unsubscribe := h.chatHub.subscribe(threadID)
	defer unsubscribe()

	go func() {
		for event := range client.send {
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		}
	}()

	client.send <- frontendChatSocketEvent{
		Type:      "ready",
		ThreadID:  threadID,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	for {
		var incoming struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		if incoming.Type != "message" {
			continue
		}

		text := strings.TrimSpace(incoming.Text)
		if text == "" {
			message := "message text is required"
			client.send <- frontendChatSocketEvent{
				Type:      "error",
				ThreadID:  threadID,
				Error:     &message,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			continue
		}

		created, err := h.createMessage(user.ID, threadID, text)
		if err != nil {
			message := "failed to send message"
			client.send <- frontendChatSocketEvent{
				Type:      "error",
				ThreadID:  threadID,
				Error:     &message,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			continue
		}
		h.broadcastCreatedMessage(threadID, created)
	}
}

func (h *FrontendHandler) broadcastCreatedMessage(threadID string, created frontendCreatedMessage) {
	h.chatHub.broadcast(threadID, frontendChatSocketEvent{
		Type:      "message",
		ThreadID:  threadID,
		Message:   &created.Sender,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	if created.RecipientThreadID != nil && created.Recipient != nil {
		h.chatHub.broadcast(*created.RecipientThreadID, frontendChatSocketEvent{
			Type:      "message",
			ThreadID:  *created.RecipientThreadID,
			Message:   created.Recipient,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	if created.RecipientUserID != nil && created.RecipientThread != nil {
		unreadCount, err := h.messageUnreadCount(*created.RecipientUserID)
		if err != nil {
			unreadCount = created.RecipientThread.Unread
		}
		h.messageInboxHub.broadcast(*created.RecipientUserID, frontendMessageInboxSocketEvent{
			Type:        "message",
			Thread:      created.RecipientThread,
			UnreadCount: unreadCount,
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}
}

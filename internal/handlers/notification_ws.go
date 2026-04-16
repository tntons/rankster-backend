package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const notificationSocketBufferSize = 16

func newNotificationHub() *notificationHub {
	return &notificationHub{
		clients: map[string]map[*notificationClient]struct{}{},
	}
}

func (hub *notificationHub) subscribe(userID string) (*notificationClient, func()) {
	client := &notificationClient{
		userID: userID,
		send:   make(chan notificationSocketEvent, notificationSocketBufferSize),
	}

	hub.mu.Lock()
	if hub.clients[userID] == nil {
		hub.clients[userID] = map[*notificationClient]struct{}{}
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

func (hub *notificationHub) broadcast(userID string, event notificationSocketEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for client := range hub.clients[userID] {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (h *Handler) WebSocketNotifications(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	user, ok := h.userFromSocketToken(c)
	if !ok {
		return
	}

	upgrader := socketUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client, unsubscribe := h.notificationHub.subscribe(user.ID)
	defer unsubscribe()

	go func() {
		for event := range client.send {
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		}
	}()

	unreadCount, err := h.notificationUnreadCount(user.ID)
	if err != nil {
		unreadCount = 0
	}
	client.send <- notificationSocketEvent{
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

func (h *Handler) broadcastNotification(userID string, notification notificationView) {
	unreadCount, err := h.notificationUnreadCount(userID)
	if err != nil {
		unreadCount = 0
	}

	h.notificationHub.broadcast(userID, notificationSocketEvent{
		Type:         "notification",
		Notification: &notification,
		UnreadCount:  unreadCount,
		Timestamp:    time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) userFromSocketToken(c *gin.Context) (userView, bool) {
	token := strings.TrimSpace(c.Query("token"))
	user, err := h.authService.UserFromAuthorization("Bearer " + token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "invalid bearer token"})
		return userView{}, false
	}

	return *user, true
}

func socketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(request *http.Request) bool {
			origin := request.Header.Get("Origin")
			return origin == "" || strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
		},
	}
}

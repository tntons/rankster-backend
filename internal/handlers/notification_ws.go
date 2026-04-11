package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"rankster-backend/internal/auth"
)

const notificationSocketBufferSize = 16

func newFrontendNotificationHub() *frontendNotificationHub {
	return &frontendNotificationHub{
		clients: map[string]map[*frontendNotificationClient]struct{}{},
	}
}

func (hub *frontendNotificationHub) subscribe(userID string) (*frontendNotificationClient, func()) {
	client := &frontendNotificationClient{
		userID: userID,
		send:   make(chan frontendNotificationSocketEvent, notificationSocketBufferSize),
	}

	hub.mu.Lock()
	if hub.clients[userID] == nil {
		hub.clients[userID] = map[*frontendNotificationClient]struct{}{}
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

func (hub *frontendNotificationHub) broadcast(userID string, event frontendNotificationSocketEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for client := range hub.clients[userID] {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (h *FrontendHandler) WebSocketNotifications(c *gin.Context) {
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
	client.send <- frontendNotificationSocketEvent{
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

func (h *FrontendHandler) broadcastNotification(userID string, notification frontendNotificationView) {
	unreadCount, err := h.notificationUnreadCount(userID)
	if err != nil {
		unreadCount = 0
	}

	h.notificationHub.broadcast(userID, frontendNotificationSocketEvent{
		Type:         "notification",
		Notification: &notification,
		UnreadCount:  unreadCount,
		Timestamp:    time.Now().Format(time.RFC3339),
	})
}

func (h *FrontendHandler) userFromSocketToken(c *gin.Context) (frontendUserView, bool) {
	token := strings.TrimSpace(c.Query("token"))
	authCtx := auth.FromAuthorization("Bearer "+token, h.authTokenSecret)
	if authCtx.Kind != "user" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "missing bearer token"})
		return frontendUserView{}, false
	}

	user, err := h.lookupUserByID(authCtx.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "invalid bearer token"})
		return frontendUserView{}, false
	}

	return buildFrontendUser(user), true
}

func frontendSocketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(request *http.Request) bool {
			origin := request.Header.Get("Origin")
			return origin == "" || strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
		},
	}
}

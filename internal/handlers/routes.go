package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/config"
)

func RegisterRoutes(router *gin.Engine, db *gorm.DB, cfg config.Config) {
	frontendHandler := NewFrontendHandler(db, cfg)

	router.POST("/auth/mock-login", frontendHandler.MockLogin)
	router.POST("/auth/google", frontendHandler.GoogleLogin)
	router.GET("/auth/me", frontendHandler.GetAuthMe)

	router.GET("/feed/main", frontendHandler.GetMainFeed)
	router.GET("/feed/post/:id", frontendHandler.GetPost)
	router.POST("/rank/create", frontendHandler.CreateRank)

	router.GET("/profile/me", frontendHandler.GetProfileMe)
	router.POST("/profile/me/pinned/:postId", frontendHandler.PinProfilePost)
	router.DELETE("/profile/me/pinned/:postId", frontendHandler.UnpinProfilePost)
	router.GET("/profile/:username", frontendHandler.GetProfileByUsername)
	router.POST("/profile/:username/follow", frontendHandler.FollowProfileUser)
	router.DELETE("/profile/:username/follow", frontendHandler.UnfollowProfileUser)

	router.GET("/search/overview", frontendHandler.SearchOverview)
	router.GET("/search/trending", frontendHandler.GetTrendingTopics)
	router.GET("/search/categories", frontendHandler.GetCategories)

	router.GET("/messages/threads", frontendHandler.GetMessages)
	router.GET("/messages/unread-count", frontendHandler.GetMessageUnreadCount)
	router.GET("/messages/ws", frontendHandler.WebSocketMessageInbox)
	router.GET("/messages/threads/:id", frontendHandler.GetMessageThread)
	router.GET("/messages/threads/:id/ws", frontendHandler.WebSocketMessageThread)
	router.POST("/messages/threads/:id/messages", frontendHandler.PostMessage)
	router.GET("/notifications", frontendHandler.GetNotifications)
	router.GET("/notifications/ws", frontendHandler.WebSocketNotifications)
	router.POST("/notifications/:id/read", frontendHandler.MarkNotificationRead)
	router.POST("/notifications/read-all", frontendHandler.MarkAllNotificationsRead)
	router.GET("/leaderboard", frontendHandler.GetLeaderboard)
	router.GET("/user/stats", frontendHandler.GetUserStats)
}

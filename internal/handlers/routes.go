package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/config"
)

func RegisterRoutes(router *gin.Engine, db *gorm.DB, cfg config.Config) {
	apiHandler := NewHandler(db, cfg)
	router.StaticFS("/uploads", gin.Dir(apiHandler.UploadDir(), false))

	router.POST("/auth/mock-login", apiHandler.MockLogin)
	router.POST("/auth/google", apiHandler.GoogleLogin)
	router.GET("/auth/me", apiHandler.GetAuthMe)
	router.POST("/uploads/images", apiHandler.UploadImage)

	router.GET("/feed/main", apiHandler.GetMainFeed)
	router.GET("/feed/post/:id", apiHandler.GetPost)
	router.PATCH("/feed/post/:id", apiHandler.UpdatePost)
	router.DELETE("/feed/post/:id", apiHandler.DeletePost)
	router.POST("/feed/post/:id/comments", apiHandler.PostComment)
	router.POST("/feed/comments/:id/like", apiHandler.LikeComment)
	router.DELETE("/feed/comments/:id/like", apiHandler.UnlikeComment)
	router.POST("/rank/create", apiHandler.CreateRank)

	router.GET("/profile/me", apiHandler.GetProfileMe)
	router.PATCH("/profile/me", apiHandler.UpdateProfileMe)
	router.POST("/profile/me/pinned/:postId", apiHandler.PinProfilePost)
	router.DELETE("/profile/me/pinned/:postId", apiHandler.UnpinProfilePost)
	router.GET("/profile/:username", apiHandler.GetProfileByUsername)
	router.POST("/profile/:username/follow", apiHandler.FollowProfileUser)
	router.DELETE("/profile/:username/follow", apiHandler.UnfollowProfileUser)

	router.GET("/search/overview", apiHandler.SearchOverview)
	router.GET("/search/trending", apiHandler.GetTrendingTopics)
	router.GET("/search/categories", apiHandler.GetCategories)

	router.GET("/messages/threads", apiHandler.GetMessages)
	router.GET("/messages/unread-count", apiHandler.GetMessageUnreadCount)
	router.GET("/messages/ws", apiHandler.WebSocketMessageInbox)
	router.GET("/messages/threads/:id", apiHandler.GetMessageThread)
	router.GET("/messages/threads/:id/ws", apiHandler.WebSocketMessageThread)
	router.POST("/messages/threads/:id/messages", apiHandler.PostMessage)
	router.GET("/notifications", apiHandler.GetNotifications)
	router.GET("/notifications/ws", apiHandler.WebSocketNotifications)
	router.POST("/notifications/:id/read", apiHandler.MarkNotificationRead)
	router.POST("/notifications/read-all", apiHandler.MarkAllNotificationsRead)
	router.GET("/leaderboard", apiHandler.GetLeaderboard)
	router.GET("/user/stats", apiHandler.GetUserStats)
}

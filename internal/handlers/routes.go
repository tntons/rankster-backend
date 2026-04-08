package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, db *gorm.DB) {
	frontendHandler := NewFrontendHandler(db)

	router.POST("/auth/mock-login", frontendHandler.MockLogin)
	router.GET("/auth/me", frontendHandler.GetAuthMe)

	router.GET("/feed/main", frontendHandler.GetMainFeed)
	router.GET("/feed/post/:id", frontendHandler.GetPost)
	router.POST("/rank/create", frontendHandler.CreateRank)

	router.GET("/profile/me", frontendHandler.GetProfileMe)
	router.GET("/profile/:username", frontendHandler.GetProfileByUsername)

	router.GET("/search/overview", frontendHandler.SearchOverview)
	router.GET("/search/trending", frontendHandler.GetTrendingTopics)
	router.GET("/search/categories", frontendHandler.GetCategories)

	router.GET("/messages/threads", frontendHandler.GetMessages)
	router.GET("/leaderboard", frontendHandler.GetLeaderboard)
	router.GET("/user/stats", frontendHandler.GetUserStats)
}

package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, db *gorm.DB) {
	feedHandler := NewFeedHandler(db)
	rankHandler := NewRankHandler(db)
	searchHandler := NewSearchHandler(db)
	userHandler := NewUserHandler(db)

	router.GET("/feed/main", feedHandler.GetMainFeed)
	router.POST("/rank/create", rankHandler.CreateRank)
	router.GET("/search/categories", searchHandler.SearchCategories)
	router.GET("/user/stats", userHandler.GetStats)
}

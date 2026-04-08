package db

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"

	"rankster-backend/internal/models"
)

func AutoMigrate(database *gorm.DB) error {
	return database.AutoMigrate(
		&models.User{},
		&models.UserAuth{},
		&models.UserProfile{},
		&models.UserStats{},
		&models.Subscription{},
		&models.Follow{},
		&models.Category{},
		&models.TierListTemplate{},
		&models.TierDefinition{},
		&models.Asset{},
		&models.Organization{},
		&models.Post{},
		&models.RankPost{},
		&models.TierListPost{},
		&models.TierListItem{},
		&models.SurveyPost{},
		&models.SurveyQuestion{},
		&models.SurveyOption{},
		&models.SurveyCampaign{},
		&models.SurveyImpression{},
		&models.PostMetrics{},
		&models.Comment{},
		&models.PostLike{},
		&models.PostShare{},
		&models.PinnedPost{},
		&models.MessageThread{},
		&models.TrendingTopic{},
		&models.LeaderboardEntry{},
	)
}

func Seed(database *gorm.DB, publicBaseURL string) error {
	var existingUsers int64
	if err := database.Model(&models.User{}).Count(&existingUsers).Error; err != nil {
		return err
	}
	if existingUsers == 0 {
		if err := seedBaseDemo(database, publicBaseURL); err != nil {
			return err
		}
	}
	return seedFrontendDemo(database, publicBaseURL)
}

func EnsureDatabase(database *gorm.DB, publicBaseURL string) error {
	if database == nil {
		return errors.New("nil database")
	}
	if err := AutoMigrate(database); err != nil {
		return err
	}
	if err := Seed(database, publicBaseURL); err != nil {
		return err
	}
	return ensureLocalDevAssetURLs(database, publicBaseURL)
}

func ensureLocalDevAssetURLs(database *gorm.DB, publicBaseURL string) error {
	baseURL := strings.TrimRight(publicBaseURL, "/")

	userUpdates := map[string]string{
		"alice": fmt.Sprintf("%s/assets/avatars/alice.svg", baseURL),
		"bob":   fmt.Sprintf("%s/assets/avatars/bob.svg", baseURL),
	}
	for username, avatarURL := range userUpdates {
		if err := database.Model(&models.UserProfile{}).
			Where("username = ?", username).
			Update("avatar_url", avatarURL).Error; err != nil {
			return err
		}
	}

	assetUpdates := map[string]string{
		"latte":    fmt.Sprintf("%s/assets/ranks/latte.svg", baseURL),
		"espresso": fmt.Sprintf("%s/assets/ranks/espresso.svg", baseURL),
	}
	for title, assetURL := range assetUpdates {
		if err := database.Model(&models.Asset{}).
			Where("url LIKE ?", "%/"+title+".svg").
			Update("url", assetURL).Error; err != nil {
			return err
		}
		if err := database.Model(&models.Asset{}).
			Where("url = ?", "https://cdn.example.com/dev/"+title+".jpg").
			Update("url", assetURL).Error; err != nil {
			return err
		}
	}

	return nil
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

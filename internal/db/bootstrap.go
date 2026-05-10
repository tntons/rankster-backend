package db

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

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
		&models.CommentLike{},
		&models.PostLike{},
		&models.PostShare{},
		&models.PinnedPost{},
		&models.MessageThread{},
		&models.DirectMessage{},
		&models.Notification{},
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
		return seedFrontendDemo(database, publicBaseURL)
	}
	// Do not re-apply the frontend demo seed on every boot. It overwrites
	// editable profile fields such as avatar_url after production deploys.
	return nil
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
	if err := ensureLocalDevAssetURLs(database, publicBaseURL); err != nil {
		return err
	}
	return ensureFrontendDemoCoverAssets(database)
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

type frontendDemoPostCoverUpdate struct {
	Username  string
	Title     string
	CoverSlug string
}

var frontendDemoPostCoverUpdates = []frontendDemoPostCoverUpdate{
	{Username: "animequeen", Title: "Best Anime of Winter 2025", CoverSlug: "anime-winter-2025"},
	{Username: "tierqueen", Title: "Pizza Toppings Definitive Ranking", CoverSlug: "pizza-toppings"},
	{Username: "rankmaster99", Title: "NBA Players 2024-25 Season", CoverSlug: "nba-players-2025"},
	{Username: "drip_scholar", Title: "2024 Hip-Hop Albums", CoverSlug: "hiphop-albums-2024"},
	{Username: "tierqueen", Title: "Best Video Games of 2024", CoverSlug: "games-2024"},
	{Username: "me", Title: "Albums I Had On Repeat In 2024", CoverSlug: "albums-on-repeat-2024"},
	{Username: "me", Title: "Games I Couldn't Stop Playing In 2024", CoverSlug: "games-i-couldnt-stop-playing-2024"},
}

var frontendDemoTopicCoverUpdates = map[string]string{
	"Best Anime of Winter 2025": "anime-winter-2025",
	"Pizza Toppings Ranking":    "pizza-toppings",
	"NBA All-Stars 2025":        "nba-players-2025",
	"Best Albums of 2024":       "hiphop-albums-2024",
	"Video Games GOTY 2024":     "games-2024",
}

func ensureFrontendDemoCoverAssets(database *gorm.DB) error {
	now := time.Now()
	return database.Transaction(func(tx *gorm.DB) error {
		for _, update := range frontendDemoPostCoverUpdates {
			if err := updateFrontendDemoPostCoverAsset(tx, now, update); err != nil {
				return err
			}
		}
		for title, coverSlug := range frontendDemoTopicCoverUpdates {
			if err := updateFrontendDemoTopicCoverAsset(tx, now, title, coverSlug); err != nil {
				return err
			}
		}
		return nil
	})
}

func updateFrontendDemoPostCoverAsset(tx *gorm.DB, now time.Time, update frontendDemoPostCoverUpdate) error {
	coverURL := strings.TrimSpace(frontendDemoCoverURLs[update.CoverSlug])
	if coverURL == "" {
		return nil
	}

	var tierList models.TierListPost
	err := tx.Model(&models.TierListPost{}).
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Joins("JOIN user_profiles ON user_profiles.user_id = posts.creator_id").
		Where("user_profiles.username = ? AND tier_list_posts.title = ?", update.Username, update.Title).
		First(&tierList).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	coverAssetID, err := ensureAsset(tx, coverURL, now)
	if err != nil {
		return err
	}
	if tierList.CoverAssetID != nil && *tierList.CoverAssetID == coverAssetID {
		return nil
	}
	return tx.Model(&models.TierListPost{}).
		Where("post_id = ?", tierList.PostID).
		Updates(map[string]any{"cover_asset_id": coverAssetID, "updated_at": now}).Error
}

func updateFrontendDemoTopicCoverAsset(tx *gorm.DB, now time.Time, title string, coverSlug string) error {
	coverURL := strings.TrimSpace(frontendDemoCoverURLs[coverSlug])
	if coverURL == "" {
		return nil
	}

	var topic models.TrendingTopic
	err := tx.Where("title = ?", title).First(&topic).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	coverAssetID, err := ensureAsset(tx, coverURL, now)
	if err != nil {
		return err
	}
	if topic.CoverAssetID != nil && *topic.CoverAssetID == coverAssetID {
		return nil
	}
	return tx.Model(&models.TrendingTopic{}).
		Where("id = ?", topic.ID).
		Updates(map[string]any{"cover_asset_id": coverAssetID, "updated_at": now}).Error
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

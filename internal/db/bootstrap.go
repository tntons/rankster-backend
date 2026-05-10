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
		&models.DataMigration{},
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
	if err := runDataMigrations(database); err != nil {
		return err
	}
	if err := ensureLocalDevAssetURLs(database, publicBaseURL); err != nil {
		return err
	}
	return ensureFrontendDemoCoverAssets(database)
}

const (
	fixRankParticipantCountsMigrationID = "20260510_fix_rank_participant_counts_from_item_count"
	backfillRankTopicsMigrationID       = "20260510_backfill_rank_topics"
	removeFrontendDemoEngagementID      = "20260511_remove_frontend_demo_engagement"
)

func runDataMigrations(database *gorm.DB) error {
	return database.Transaction(func(tx *gorm.DB) error {
		migrations := []struct {
			id  string
			run func(*gorm.DB) error
		}{
			{fixRankParticipantCountsMigrationID, fixRankParticipantCountsFromItemCounts},
			{backfillRankTopicsMigrationID, backfillRankTopics},
			{removeFrontendDemoEngagementID, removeFrontendDemoEngagement},
		}

		for _, migration := range migrations {
			if err := runDataMigration(tx, migration.id, migration.run); err != nil {
				return err
			}
		}
		return nil
	})
}

func runDataMigration(tx *gorm.DB, id string, run func(*gorm.DB) error) error {
	var existing models.DataMigration
	err := tx.Where("id = ?", id).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := run(tx); err != nil {
		return err
	}
	return tx.Create(&models.DataMigration{
		ID:        id,
		AppliedAt: time.Now(),
	}).Error
}

func fixRankParticipantCountsFromItemCounts(tx *gorm.DB) error {
	var lists []models.TierListPost
	if err := tx.Preload("Items").
		Where("participant_count > ? AND participant_count <= ?", 1, 50).
		Find(&lists).Error; err != nil {
		return err
	}

	now := time.Now()
	for _, list := range lists {
		itemCount := len(list.Items)
		if itemCount == 0 || list.ParticipantCount < itemCount {
			continue
		}

		// Older create logic accidentally initialized participant_count from the
		// item count. Preserve any later "rank this" increments by subtracting
		// the item-count seed and keeping the creator as the first participant.
		correctedCount := list.ParticipantCount - itemCount + 1
		if correctedCount < 1 || correctedCount == list.ParticipantCount {
			continue
		}

		if err := tx.Model(&models.TierListPost{}).
			Where("post_id = ?", list.PostID).
			Updates(map[string]any{
				"participant_count": correctedCount,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillRankTopics(tx *gorm.DB) error {
	var lists []models.TierListPost
	if err := tx.
		Preload("Post").
		Order("created_at asc").
		Find(&lists).Error; err != nil {
		return err
	}

	type topicKey struct {
		title      string
		categoryID string
	}
	groups := map[topicKey][]models.TierListPost{}
	for _, list := range lists {
		key := topicKey{
			title:      normalizedTopicTitle(list.Title),
			categoryID: list.Post.CategoryID,
		}
		groups[key] = append(groups[key], list)
	}

	now := time.Now()
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		topicID := ""
		for _, list := range group {
			if list.TopicID != nil && strings.TrimSpace(*list.TopicID) != "" {
				topicID = *list.TopicID
				break
			}
		}
		if topicID == "" {
			topicID = group[0].PostID
		}

		publicCount := int64(0)
		postIDs := make([]string, 0, len(group))
		for _, list := range group {
			postIDs = append(postIDs, list.PostID)
			if list.Post.Visibility == "PUBLIC" {
				publicCount++
			}
		}
		if publicCount < 1 {
			publicCount = 1
		}

		if err := tx.Model(&models.TierListPost{}).
			Where("post_id IN ?", postIDs).
			Updates(map[string]any{
				"topic_id":          topicID,
				"participant_count": publicCount,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizedTopicTitle(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(title))), " ")
}

type frontendDemoPostKey struct {
	username string
	title    string
}

type frontendDemoSeededComment struct {
	frontendDemoPostKey
	commentUsername string
	body            string
}

var frontendDemoEngagementPostKeys = []frontendDemoPostKey{
	{username: "animequeen", title: "Best Anime of Winter 2025"},
	{username: "tierqueen", title: "Pizza Toppings Definitive Ranking"},
	{username: "rankmaster99", title: "NBA Players 2024-25 Season"},
	{username: "drip_scholar", title: "2024 Hip-Hop Albums"},
	{username: "tierqueen", title: "Best Video Games of 2024"},
	{username: "me", title: "Albums I Had On Repeat In 2024"},
	{username: "me", title: "Games I Couldn't Stop Playing In 2024"},
}

var frontendDemoSeededComments = []frontendDemoSeededComment{
	{frontendDemoPostKey: frontendDemoPostKey{username: "animequeen", title: "Best Anime of Winter 2025"}, commentUsername: "tierqueen", body: "Sakamoto Days in D is criminal"},
	{frontendDemoPostKey: frontendDemoPostKey{username: "animequeen", title: "Best Anime of Winter 2025"}, commentUsername: "rankmaster99", body: "Finally someone who appreciates Blue Box!"},
	{frontendDemoPostKey: frontendDemoPostKey{username: "tierqueen", title: "Pizza Toppings Definitive Ranking"}, commentUsername: "drip_scholar", body: "Pineapple supremacy!! D tier is wrong"},
	{frontendDemoPostKey: frontendDemoPostKey{username: "tierqueen", title: "Pizza Toppings Definitive Ranking"}, commentUsername: "rankmaster99", body: "Finally! A correct pizza tier list."},
	{frontendDemoPostKey: frontendDemoPostKey{username: "rankmaster99", title: "NBA Players 2024-25 Season"}, commentUsername: "animequeen", body: "Curry in C is absolutely disrespectful"},
	{frontendDemoPostKey: frontendDemoPostKey{username: "tierqueen", title: "Best Video Games of 2024"}, commentUsername: "rankmaster99", body: "Balatro S tier is absolutely based"},
	{frontendDemoPostKey: frontendDemoPostKey{username: "me", title: "Albums I Had On Repeat In 2024"}, commentUsername: "tierqueen", body: "Charm in A tier is so real"},
	{frontendDemoPostKey: frontendDemoPostKey{username: "me", title: "Games I Couldn't Stop Playing In 2024"}, commentUsername: "rankmaster99", body: "Balatro at S is the only truth"},
}

var frontendDemoUsernames = []string{"animequeen", "tierqueen", "rankmaster99", "drip_scholar", "me"}

func removeFrontendDemoEngagement(tx *gorm.DB) error {
	postIDsByKey := map[frontendDemoPostKey]string{}
	postIDs := make([]string, 0, len(frontendDemoEngagementPostKeys))

	for _, key := range frontendDemoEngagementPostKeys {
		postID, err := frontendDemoPostID(tx, key)
		if err != nil {
			return err
		}
		if postID == "" {
			continue
		}
		postIDsByKey[key] = postID
		postIDs = append(postIDs, postID)
	}
	if len(postIDs) == 0 {
		return nil
	}

	if err := removeSeededDemoComments(tx, postIDsByKey); err != nil {
		return err
	}

	demoUserIDs := tx.Model(&models.UserProfile{}).
		Select("user_id").
		Where("username IN ?", frontendDemoUsernames)
	if err := tx.Where("post_id IN ? AND user_id IN (?)", postIDs, demoUserIDs).
		Delete(&models.PostLike{}).Error; err != nil {
		return err
	}

	for _, postID := range postIDs {
		if err := refreshPostMetricCounts(tx, postID); err != nil {
			return err
		}
	}
	return nil
}

func frontendDemoPostID(tx *gorm.DB, key frontendDemoPostKey) (string, error) {
	var list models.TierListPost
	err := tx.Model(&models.TierListPost{}).
		Joins("JOIN posts ON posts.id = tier_list_posts.post_id").
		Joins("JOIN user_profiles ON user_profiles.user_id = posts.creator_id").
		Where("user_profiles.username = ? AND tier_list_posts.title = ?", key.username, key.title).
		First(&list).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return list.PostID, nil
}

func removeSeededDemoComments(tx *gorm.DB, postIDsByKey map[frontendDemoPostKey]string) error {
	for _, seed := range frontendDemoSeededComments {
		postID := postIDsByKey[seed.frontendDemoPostKey]
		if postID == "" {
			continue
		}

		commentIDsForLikes := tx.Model(&models.Comment{}).
			Select("comments.id").
			Joins("JOIN user_profiles ON user_profiles.user_id = comments.author_id").
			Where("comments.post_id = ? AND user_profiles.username = ? AND comments.body = ?", postID, seed.commentUsername, seed.body)
		if err := tx.Where("comment_id IN (?)", commentIDsForLikes).
			Delete(&models.CommentLike{}).Error; err != nil {
			return err
		}
		commentIDsForDelete := tx.Model(&models.Comment{}).
			Select("comments.id").
			Joins("JOIN user_profiles ON user_profiles.user_id = comments.author_id").
			Where("comments.post_id = ? AND user_profiles.username = ? AND comments.body = ?", postID, seed.commentUsername, seed.body)
		if err := tx.Where("id IN (?)", commentIDsForDelete).
			Delete(&models.Comment{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func refreshPostMetricCounts(tx *gorm.DB, postID string) error {
	var likes int64
	if err := tx.Model(&models.PostLike{}).Where("post_id = ?", postID).Count(&likes).Error; err != nil {
		return err
	}
	var comments int64
	if err := tx.Model(&models.Comment{}).Where("post_id = ?", postID).Count(&comments).Error; err != nil {
		return err
	}
	var shares int64
	if err := tx.Model(&models.PostShare{}).Where("post_id = ?", postID).Count(&shares).Error; err != nil {
		return err
	}

	hotScore := float64(likes) + float64(comments)*1.5 + float64(shares)*2
	return tx.Model(&models.PostMetrics{}).
		Where("post_id = ?", postID).
		Updates(map[string]any{
			"like_count":    likes,
			"comment_count": comments,
			"share_count":   shares,
			"hot_score":     hotScore,
			"updated_at":    time.Now(),
		}).Error
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

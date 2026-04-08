package db

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
	)
}

func Seed(database *gorm.DB) error {
	var existingUsers int64
	if err := database.Model(&models.User{}).Count(&existingUsers).Error; err != nil {
		return err
	}
	if existingUsers > 0 {
		return nil
	}

	now := time.Now()

	aliceID := uuid.NewString()
	bobID := uuid.NewString()
	categoryID := uuid.NewString()
	templateID := uuid.NewString()
	latteAssetID := uuid.NewString()
	espressoAssetID := uuid.NewString()
	orgID := uuid.NewString()
	surveyPostID := uuid.NewString()
	surveyCampaignID := uuid.NewString()

	return database.Transaction(func(tx *gorm.DB) error {
		aliceEmail := "alice@example.com"
		bobEmail := "bob@example.com"
		aliceName := "Alice"
		bobName := "Bob"
		aliceUsername := "alice"
		bobUsername := "bob"
		theme := "#0f766e"
		latteURL := "https://cdn.example.com/dev/latte.jpg"
		espressoURL := "https://cdn.example.com/dev/espresso.jpg"
		orgWebsite := "https://example.com"
		surveyDescription := "Anonymous research survey."

		users := []models.User{
			{ID: aliceID, CreatedAt: now, UpdatedAt: now},
			{ID: bobID, CreatedAt: now, UpdatedAt: now},
		}
		if err := tx.Create(&users).Error; err != nil {
			return err
		}

		auths := []models.UserAuth{
			{ID: uuid.NewString(), UserID: aliceID, Provider: "LOCAL", Email: &aliceEmail, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.NewString(), UserID: bobID, Provider: "LOCAL", Email: &bobEmail, CreatedAt: now, UpdatedAt: now},
		}
		if err := tx.Create(&auths).Error; err != nil {
			return err
		}

		profiles := []models.UserProfile{
			{ID: uuid.NewString(), UserID: aliceID, Username: aliceUsername, DisplayName: &aliceName, ThemeColor: &theme, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.NewString(), UserID: bobID, Username: bobUsername, DisplayName: &bobName, CreatedAt: now, UpdatedAt: now},
		}
		if err := tx.Create(&profiles).Error; err != nil {
			return err
		}

		stats := []models.UserStats{
			{ID: uuid.NewString(), UserID: aliceID, RanksCreatedCount: 1, FollowersCount: 1, FollowingCount: 0, UpdatedAt: now},
			{ID: uuid.NewString(), UserID: bobID, RanksCreatedCount: 1, FollowersCount: 0, FollowingCount: 1, UpdatedAt: now},
		}
		if err := tx.Create(&stats).Error; err != nil {
			return err
		}

		plans := []models.Subscription{
			{ID: uuid.NewString(), UserID: aliceID, Plan: "PRO", Status: "ACTIVE", StartedAt: now, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.NewString(), UserID: bobID, Plan: "FREE", Status: "ACTIVE", StartedAt: now, CreatedAt: now, UpdatedAt: now},
		}
		if err := tx.Create(&plans).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.Follow{ID: uuid.NewString(), FollowerID: bobID, FollowingID: aliceID, CreatedAt: now}).Error; err != nil {
			return err
		}

		category := models.Category{
			ID: categoryID, Slug: "coffee", Name: "Coffee", Tags: pq.StringArray{"drinks", "food"}, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&category).Error; err != nil {
			return err
		}

		template := models.TierListTemplate{
			ID: templateID, CategoryID: categoryID, IsMaster: true, Title: "Coffee Master Tier List", Visibility: "PUBLIC", CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&template).Error; err != nil {
			return err
		}

		tiers := []models.TierDefinition{
			{ID: uuid.NewString(), TemplateID: templateID, Key: "S", Label: "S", Order: 1},
			{ID: uuid.NewString(), TemplateID: templateID, Key: "A", Label: "A", Order: 2},
			{ID: uuid.NewString(), TemplateID: templateID, Key: "B", Label: "B", Order: 3},
			{ID: uuid.NewString(), TemplateID: templateID, Key: "C", Label: "C", Order: 4},
			{ID: uuid.NewString(), TemplateID: templateID, Key: "D", Label: "D", Order: 5},
		}
		if err := tx.Create(&tiers).Error; err != nil {
			return err
		}

		assets := []models.Asset{
			{ID: latteAssetID, URL: latteURL, CreatedAt: now},
			{ID: espressoAssetID, URL: espressoURL, CreatedAt: now},
		}
		if err := tx.Create(&assets).Error; err != nil {
			return err
		}

		rankPost1ID := uuid.NewString()
		rankPost2ID := uuid.NewString()

		posts := []models.Post{
			{ID: rankPost1ID, Type: "RANK", Visibility: "PUBLIC", CreatorID: aliceID, CategoryID: categoryID, Caption: stringPtr("Latte is S-tier for me"), CreatedAt: now, UpdatedAt: now},
			{ID: rankPost2ID, Type: "RANK", Visibility: "PUBLIC", CreatorID: bobID, CategoryID: categoryID, Caption: stringPtr("Espresso goes A-tier"), CreatedAt: now, UpdatedAt: now},
			{ID: surveyPostID, Type: "SURVEY", Visibility: "PUBLIC", CreatorID: aliceID, CategoryID: categoryID, Caption: stringPtr("Quick coffee preference survey"), CreatedAt: now, UpdatedAt: now},
		}
		if err := tx.Create(&posts).Error; err != nil {
			return err
		}

		ranks := []models.RankPost{
			{PostID: rankPost1ID, TemplateID: templateID, TierKey: "S", ImageAssetID: latteAssetID, SubjectTitle: stringPtr("Latte")},
			{PostID: rankPost2ID, TemplateID: templateID, TierKey: "A", ImageAssetID: espressoAssetID, SubjectTitle: stringPtr("Espresso")},
		}
		if err := tx.Create(&ranks).Error; err != nil {
			return err
		}

		org := models.Organization{ID: orgID, Name: "Acme Research Lab", Website: &orgWebsite, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}

		survey := models.SurveyPost{
			PostID: surveyPostID, SurveyType: "THESIS", SponsorOrgID: &orgID, Title: "Coffee Habits 2026", Description: &surveyDescription,
		}
		if err := tx.Create(&survey).Error; err != nil {
			return err
		}

		questionID := uuid.NewString()
		if err := tx.Create(&models.SurveyQuestion{
			ID: questionID, SurveyPostID: surveyPostID, Order: 1, Type: "SINGLE_CHOICE", Prompt: "How many coffees do you drink per day?", Required: true,
		}).Error; err != nil {
			return err
		}

		options := []models.SurveyOption{
			{ID: uuid.NewString(), QuestionID: questionID, Order: 1, Label: "0"},
			{ID: uuid.NewString(), QuestionID: questionID, Order: 2, Label: "1"},
			{ID: uuid.NewString(), QuestionID: questionID, Order: 3, Label: "2-3"},
			{ID: uuid.NewString(), QuestionID: questionID, Order: 4, Label: "4+"},
		}
		if err := tx.Create(&options).Error; err != nil {
			return err
		}

		targeting := `{"countries":["TH","US"],"interests":["coffee"]}`
		campaign := models.SurveyCampaign{
			ID: surveyCampaignID, SurveyPostID: surveyPostID, SponsorOrgID: orgID, StartAt: now, BudgetCents: 5000, SpentCents: 0, Targeting: &targeting, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&campaign).Error; err != nil {
			return err
		}

		metrics := []models.PostMetrics{
			{PostID: rankPost1ID, LikeCount: 1, CommentCount: 1, ShareCount: 1, HotScore: 4.5, UpdatedAt: now},
			{PostID: rankPost2ID, LikeCount: 0, CommentCount: 0, ShareCount: 0, HotScore: 2.0, UpdatedAt: now},
			{PostID: surveyPostID, LikeCount: 0, CommentCount: 0, ShareCount: 0, HotScore: 1.0, UpdatedAt: now},
		}
		if err := tx.Create(&metrics).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.Comment{ID: uuid.NewString(), PostID: rankPost1ID, AuthorID: bobID, Body: "Hard agree.", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.PostLike{ID: uuid.NewString(), PostID: rankPost1ID, UserID: bobID, CreatedAt: now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.PostShare{ID: uuid.NewString(), PostID: rankPost1ID, UserID: bobID, Channel: "copy_link", CreatedAt: now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.PinnedPost{ID: uuid.NewString(), UserID: aliceID, PostID: rankPost1ID, Order: intPtr(1), CreatedAt: now}).Error; err != nil {
			return err
		}

		return nil
	})
}

func EnsureDatabase(database *gorm.DB) error {
	if database == nil {
		return errors.New("nil database")
	}
	if err := AutoMigrate(database); err != nil {
		return err
	}
	return Seed(database)
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

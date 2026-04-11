package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"rankster-backend/internal/models"
)

type frontendDemoUser struct {
	Username      string
	DisplayName   string
	Bio           string
	Followers     int
	Following     int
	TotalRankings int
	Verified      bool
	Plan          string
}

type frontendDemoTierItem struct {
	ExternalID string
	Name       string
	Emoji      *string
}

type frontendDemoComment struct {
	Username string
	Text     string
	Age      time.Duration
	Likes    int
}

type frontendDemoPost struct {
	Username         string
	Title            string
	CategorySlug     string
	CoverSlug        string
	Description      string
	Tags             []string
	Likes            int
	LikedBy          []string
	Shares           int
	ParticipantCount int
	Age              time.Duration
	Tiers            map[string][]frontendDemoTierItem
	AllItems         []frontendDemoTierItem
	Comments         []frontendDemoComment
}

type frontendDemoMessage struct {
	OwnerUsername string
	PeerUsername  string
	LastMessage   string
	Age           time.Duration
	Unread        int
	History       []frontendDemoChatMessage
}

type frontendDemoChatMessage struct {
	SenderUsername string
	Text           string
	Age            time.Duration
}

type frontendDemoTopic struct {
	Title            string
	CategorySlug     string
	CoverSlug        string
	SourcePostTitle  string
	ParticipantCount int
	Tags             []string
}

type frontendDemoCategory struct {
	Slug  string
	Name  string
	Emoji string
	Color string
	Tags  []string
}

type frontendDemoLeaderboardEntry struct {
	Username string
	Rank     int
	Score    int
	Change   string
}

func seedBaseDemo(database *gorm.DB, publicBaseURL string) error {
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
		baseURL := strings.TrimRight(publicBaseURL, "/")
		latteURL := fmt.Sprintf("%s/assets/ranks/latte.svg", baseURL)
		espressoURL := fmt.Sprintf("%s/assets/ranks/espresso.svg", baseURL)
		aliceAvatarURL := fmt.Sprintf("%s/assets/avatars/alice.svg", baseURL)
		bobAvatarURL := fmt.Sprintf("%s/assets/avatars/bob.svg", baseURL)
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
			{ID: uuid.NewString(), UserID: aliceID, Username: aliceUsername, DisplayName: &aliceName, AvatarURL: &aliceAvatarURL, ThemeColor: &theme, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.NewString(), UserID: bobID, Username: bobUsername, DisplayName: &bobName, AvatarURL: &bobAvatarURL, CreatedAt: now, UpdatedAt: now},
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

		if err := tx.Create(&models.Comment{ID: uuid.NewString(), PostID: rankPost1ID, AuthorID: bobID, Body: "Hard agree.", LikeCount: 0, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
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

func seedFrontendDemo(database *gorm.DB, publicBaseURL string) error {
	now := time.Now()
	baseURL := strings.TrimRight(publicBaseURL, "/")

	emojiSword := "⚔️"
	emojiGhost := "👻"
	emojiBall := "🏀"
	emojiFire := "🔥"
	emojiSlice := "🍕"
	emojiCard := "🃏"

	users := []frontendDemoUser{
		{Username: "tierqueen", DisplayName: "Sophia Chen", Bio: "Ranking everything, one tier at a time | Film buff | Foodie", Followers: 12400, Following: 832, TotalRankings: 147, Verified: true, Plan: "PRO"},
		{Username: "rankmaster99", DisplayName: "Jordan Miles", Bio: "Sports and gaming tier lists | Hot takes only", Followers: 8930, Following: 421, TotalRankings: 89, Verified: false, Plan: "FREE"},
		{Username: "animequeen", DisplayName: "Yuki Tanaka", Bio: "Anime and manga enthusiast | Seasonal watcher | Music lover", Followers: 21050, Following: 1203, TotalRankings: 214, Verified: true, Plan: "PRO"},
		{Username: "drip_scholar", DisplayName: "Marcus Thompson", Bio: "Fashion and culture critic | Every fit is a tier list", Followers: 5670, Following: 309, TotalRankings: 62, Verified: false, Plan: "FREE"},
		{Username: "me", DisplayName: "Alex Rivera", Bio: "Just vibing and ranking | Music and Movies", Followers: 3210, Following: 891, TotalRankings: 38, Verified: false, Plan: "FREE"},
	}

	categories := []frontendDemoCategory{
		{Slug: "movies", Name: "Movies & TV", Emoji: "🎬", Color: "bg-purple-100 text-purple-700", Tags: []string{"movies", "tv"}},
		{Slug: "music", Name: "Music", Emoji: "🎵", Color: "bg-pink-100 text-pink-700", Tags: []string{"music"}},
		{Slug: "food", Name: "Food & Drinks", Emoji: "🍕", Color: "bg-orange-100 text-orange-700", Tags: []string{"food", "drinks"}},
		{Slug: "sports", Name: "Sports", Emoji: "🏀", Color: "bg-green-100 text-green-700", Tags: []string{"sports"}},
		{Slug: "gaming", Name: "Gaming", Emoji: "🎮", Color: "bg-blue-100 text-blue-700", Tags: []string{"gaming"}},
		{Slug: "anime", Name: "Anime", Emoji: "⛩️", Color: "bg-red-100 text-red-700", Tags: []string{"anime"}},
		{Slug: "travel", Name: "Travel", Emoji: "✈️", Color: "bg-cyan-100 text-cyan-700", Tags: []string{"travel"}},
		{Slug: "tech", Name: "Tech", Emoji: "💻", Color: "bg-slate-100 text-slate-700", Tags: []string{"tech"}},
		{Slug: "fashion", Name: "Fashion", Emoji: "👗", Color: "bg-rose-100 text-rose-700", Tags: []string{"fashion"}},
		{Slug: "books", Name: "Books", Emoji: "📚", Color: "bg-amber-100 text-amber-700", Tags: []string{"books"}},
	}

	posts := []frontendDemoPost{
		{
			Username: "animequeen", Title: "Best Anime of Winter 2025", CategorySlug: "anime", CoverSlug: "anime-winter-2025",
			Description: "My honest tier list for this season. Frieren is a masterpiece, change my mind.",
			Tags:        []string{"anime", "winter2025", "seasonal"}, Likes: 2847, LikedBy: []string{"me"}, Shares: 341, ParticipantCount: 1247, Age: 4 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "i1", Name: "Frieren", Emoji: &emojiSword}, {ExternalID: "i2", Name: "Dandadan", Emoji: &emojiGhost}},
				"A": {{ExternalID: "i3", Name: "Blue Box"}, {ExternalID: "i4", Name: "Solo Leveling S2"}},
				"B": {{ExternalID: "i5", Name: "Apothecary Diaries"}},
				"C": {{ExternalID: "i6", Name: "Wind Breaker"}},
				"D": {{ExternalID: "i7", Name: "Sakamoto Days"}},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "i1", Name: "Frieren"}, {ExternalID: "i2", Name: "Dandadan"}, {ExternalID: "i3", Name: "Blue Box"}, {ExternalID: "i4", Name: "Solo Leveling S2"}, {ExternalID: "i5", Name: "Apothecary Diaries"}, {ExternalID: "i6", Name: "Wind Breaker"}, {ExternalID: "i7", Name: "Sakamoto Days"}},
			Comments: []frontendDemoComment{{Username: "tierqueen", Text: "Sakamoto Days in D is criminal", Age: 2 * time.Hour, Likes: 142}, {Username: "rankmaster99", Text: "Finally someone who appreciates Blue Box!", Age: 3 * time.Hour, Likes: 87}},
		},
		{
			Username: "tierqueen", Title: "Pizza Toppings Definitive Ranking", CategorySlug: "food", CoverSlug: "pizza-toppings",
			Description: "The definitive pizza topping tier list. Pineapple deserves D and I will not be taking questions.",
			Tags:        []string{"food", "pizza", "hotTake"}, Likes: 5102, Shares: 892, ParticipantCount: 4321, Age: 6 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "j1", Name: "Pepperoni", Emoji: &emojiSlice}, {ExternalID: "j2", Name: "Mushrooms"}},
				"A": {{ExternalID: "j3", Name: "Olives"}, {ExternalID: "j4", Name: "BBQ Chicken"}},
				"B": {{ExternalID: "j5", Name: "Bell Peppers"}, {ExternalID: "j6", Name: "Sausage"}},
				"C": {{ExternalID: "j7", Name: "Anchovies"}},
				"D": {{ExternalID: "j8", Name: "Pineapple"}},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "j1", Name: "Pepperoni"}, {ExternalID: "j2", Name: "Mushrooms"}, {ExternalID: "j3", Name: "Olives"}, {ExternalID: "j4", Name: "BBQ Chicken"}, {ExternalID: "j5", Name: "Bell Peppers"}, {ExternalID: "j6", Name: "Sausage"}, {ExternalID: "j7", Name: "Anchovies"}, {ExternalID: "j8", Name: "Pineapple"}},
			Comments: []frontendDemoComment{{Username: "drip_scholar", Text: "Pineapple supremacy!! D tier is wrong", Age: 1 * time.Hour, Likes: 312}, {Username: "rankmaster99", Text: "Finally! A correct pizza tier list.", Age: 2 * time.Hour, Likes: 201}},
		},
		{
			Username: "rankmaster99", Title: "NBA Players 2024-25 Season", CategorySlug: "sports", CoverSlug: "nba-players-2025",
			Description: "Controversial? Maybe. Accurate? Absolutely. Let the arguments begin.",
			Tags:        []string{"nba", "basketball", "sports"}, Likes: 8912, Shares: 1204, ParticipantCount: 6789, Age: 8 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "k1", Name: "Nikola Jokic"}, {ExternalID: "k2", Name: "Shai Gilgeous-Alexander", Emoji: &emojiBall}},
				"A": {{ExternalID: "k3", Name: "LeBron James"}, {ExternalID: "k4", Name: "Luka Doncic", Emoji: &emojiFire}},
				"B": {{ExternalID: "k5", Name: "Giannis Antetokounmpo"}, {ExternalID: "k6", Name: "Kevin Durant"}},
				"C": {{ExternalID: "k7", Name: "Stephen Curry"}},
				"D": {{ExternalID: "k8", Name: "Kyrie Irving"}},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "k1", Name: "Nikola Jokic"}, {ExternalID: "k2", Name: "Shai Gilgeous-Alexander"}, {ExternalID: "k3", Name: "LeBron James"}, {ExternalID: "k4", Name: "Luka Doncic"}, {ExternalID: "k5", Name: "Giannis Antetokounmpo"}, {ExternalID: "k6", Name: "Kevin Durant"}, {ExternalID: "k7", Name: "Stephen Curry"}, {ExternalID: "k8", Name: "Kyrie Irving"}},
			Comments: []frontendDemoComment{{Username: "animequeen", Text: "Curry in C is absolutely disrespectful", Age: 30 * time.Minute, Likes: 876}},
		},
		{
			Username: "drip_scholar", Title: "2024 Hip-Hop Albums", CategorySlug: "music", CoverSlug: "hiphop-albums-2024",
			Description: "2024 was a legendary year for music. GNX and Chromakopia are instant classics.",
			Tags:        []string{"music", "hiphop", "2024", "albums"}, Likes: 3401, LikedBy: []string{"me"}, Shares: 567, ParticipantCount: 2109, Age: 24 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "l1", Name: "GNX - Kendrick"}, {ExternalID: "l2", Name: "Chromakopia - Tyler"}},
				"A": {{ExternalID: "l3", Name: "Bright Future - Adrianne"}},
				"B": {{ExternalID: "l4", Name: "Short n Sweet - Sabrina"}, {ExternalID: "l5", Name: "Hit Me Hard - Billie"}},
				"C": {{ExternalID: "l6", Name: "Radical Optimism - Dua"}},
				"D": {},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "l1", Name: "GNX - Kendrick"}, {ExternalID: "l2", Name: "Chromakopia - Tyler"}, {ExternalID: "l3", Name: "Bright Future - Adrianne"}, {ExternalID: "l4", Name: "Short n Sweet - Sabrina"}, {ExternalID: "l5", Name: "Hit Me Hard - Billie"}, {ExternalID: "l6", Name: "Radical Optimism - Dua"}},
			Comments: []frontendDemoComment{},
		},
		{
			Username: "tierqueen", Title: "Best Video Games of 2024", CategorySlug: "gaming", CoverSlug: "games-2024",
			Description: "2024 was packed with bangers. Balatro being S tier is the most correct take I've ever had.",
			Tags:        []string{"gaming", "2024", "videogames"}, Likes: 4231, Shares: 712, ParticipantCount: 3456, Age: 48 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "m1", Name: "Elden Ring DLC"}, {ExternalID: "m2", Name: "Balatro", Emoji: &emojiCard}},
				"A": {{ExternalID: "m3", Name: "Black Myth: Wukong"}, {ExternalID: "m4", Name: "Astro Bot"}},
				"B": {{ExternalID: "m5", Name: "Tekken 8"}, {ExternalID: "m6", Name: "Palworld"}},
				"C": {{ExternalID: "m7", Name: "Senua's Saga"}},
				"D": {},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "m1", Name: "Elden Ring DLC"}, {ExternalID: "m2", Name: "Balatro"}, {ExternalID: "m3", Name: "Black Myth: Wukong"}, {ExternalID: "m4", Name: "Astro Bot"}, {ExternalID: "m5", Name: "Tekken 8"}, {ExternalID: "m6", Name: "Palworld"}, {ExternalID: "m7", Name: "Senua's Saga"}},
			Comments: []frontendDemoComment{{Username: "rankmaster99", Text: "Balatro S tier is absolutely based", Age: 5 * time.Hour, Likes: 234}},
		},
		{
			Username: "me", Title: "Albums I Had On Repeat In 2024", CategorySlug: "music", CoverSlug: "albums-on-repeat-2024",
			Description: "My personal replay-heavy list from 2024. This one is all vibes and zero objectivity.",
			Tags:        []string{"music", "albums", "2024"}, Likes: 1940, Shares: 288, ParticipantCount: 904, Age: 18 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "n1", Name: "GNX - Kendrick"}, {ExternalID: "n2", Name: "Chromakopia - Tyler"}},
				"A": {{ExternalID: "n3", Name: "Charm - Clairo"}, {ExternalID: "n4", Name: "Cowboy Carter - Beyonce"}},
				"B": {{ExternalID: "n5", Name: "Hit Me Hard - Billie"}},
				"C": {{ExternalID: "n6", Name: "Short n Sweet - Sabrina"}},
				"D": {},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "n1", Name: "GNX - Kendrick"}, {ExternalID: "n2", Name: "Chromakopia - Tyler"}, {ExternalID: "n3", Name: "Charm - Clairo"}, {ExternalID: "n4", Name: "Cowboy Carter - Beyonce"}, {ExternalID: "n5", Name: "Hit Me Hard - Billie"}, {ExternalID: "n6", Name: "Short n Sweet - Sabrina"}},
			Comments: []frontendDemoComment{{Username: "tierqueen", Text: "Charm in A tier is so real", Age: 7 * time.Hour, Likes: 92}},
		},
		{
			Username: "me", Title: "Games I Couldn't Stop Playing In 2024", CategorySlug: "gaming", CoverSlug: "games-i-couldnt-stop-playing-2024",
			Description: "Less objective, more obsession. Balatro absolutely consumed my life.",
			Tags:        []string{"gaming", "2024", "favorites"}, Likes: 2315, Shares: 305, ParticipantCount: 1120, Age: 36 * time.Hour,
			Tiers: map[string][]frontendDemoTierItem{
				"S": {{ExternalID: "o1", Name: "Balatro", Emoji: &emojiCard}, {ExternalID: "o2", Name: "Astro Bot"}},
				"A": {{ExternalID: "o3", Name: "Elden Ring DLC"}, {ExternalID: "o4", Name: "Tekken 8"}},
				"B": {{ExternalID: "o5", Name: "Metaphor: ReFantazio"}},
				"C": {{ExternalID: "o6", Name: "Palworld"}},
				"D": {},
			},
			AllItems: []frontendDemoTierItem{{ExternalID: "o1", Name: "Balatro"}, {ExternalID: "o2", Name: "Astro Bot"}, {ExternalID: "o3", Name: "Elden Ring DLC"}, {ExternalID: "o4", Name: "Tekken 8"}, {ExternalID: "o5", Name: "Metaphor: ReFantazio"}, {ExternalID: "o6", Name: "Palworld"}},
			Comments: []frontendDemoComment{{Username: "rankmaster99", Text: "Balatro at S is the only truth", Age: 9 * time.Hour, Likes: 144}},
		},
	}

	messages := []frontendDemoMessage{
		{
			OwnerUsername: "me",
			PeerUsername:  "animequeen",
			LastMessage:   "Your NBA tier list is so wrong lmaoo 😭",
			Age:           2 * time.Minute,
			Unread:        3,
			History: []frontendDemoChatMessage{
				{SenderUsername: "animequeen", Text: "Your NBA tier list is so wrong lmaoo 😭", Age: 8 * time.Minute},
				{SenderUsername: "animequeen", Text: "Curry in C is disrespectful??", Age: 7 * time.Minute},
				{SenderUsername: "me", Text: "He's past his prime, I stand by it 😅", Age: 5 * time.Minute},
				{SenderUsername: "animequeen", Text: "Absolute crime. Anyway check my new anime tier list!", Age: 4 * time.Minute},
				{SenderUsername: "me", Text: "omg Frieren S tier?? finally someone with taste", Age: 2 * time.Minute},
			},
		},
		{
			OwnerUsername: "me",
			PeerUsername:  "tierqueen",
			LastMessage:   "omg same taste in anime!! ✨",
			Age:           15 * time.Minute,
			Unread:        1,
			History: []frontendDemoChatMessage{
				{SenderUsername: "tierqueen", Text: "Your winter anime ranking is basically perfect.", Age: 45 * time.Minute},
				{SenderUsername: "me", Text: "You get it. Frieren supremacy.", Age: 35 * time.Minute},
				{SenderUsername: "tierqueen", Text: "omg same taste in anime!! ✨", Age: 15 * time.Minute},
			},
		},
		{
			OwnerUsername: "me",
			PeerUsername:  "rankmaster99",
			LastMessage:   "Can you rank coffee shops next?",
			Age:           1 * time.Hour,
			Unread:        0,
			History: []frontendDemoChatMessage{
				{SenderUsername: "me", Text: "Your NBA comments section is on fire today.", Age: 95 * time.Minute},
				{SenderUsername: "rankmaster99", Text: "As it should be. I speak truth.", Age: 75 * time.Minute},
				{SenderUsername: "rankmaster99", Text: "Can you rank coffee shops next?", Age: 1 * time.Hour},
			},
		},
		{
			OwnerUsername: "me",
			PeerUsername:  "drip_scholar",
			LastMessage:   "The collab tier list is live!",
			Age:           3 * time.Hour,
			Unread:        0,
			History: []frontendDemoChatMessage{
				{SenderUsername: "drip_scholar", Text: "We need a fashion x music collab list.", Age: 5 * time.Hour},
				{SenderUsername: "me", Text: "Say less, I'll set it up.", Age: 4 * time.Hour},
				{SenderUsername: "me", Text: "The collab tier list is live!", Age: 3 * time.Hour},
			},
		},
	}

	topics := []frontendDemoTopic{
		{Title: "Best Anime of Winter 2025", CategorySlug: "anime", CoverSlug: "anime-winter-2025", SourcePostTitle: "Best Anime of Winter 2025", ParticipantCount: 12847, Tags: []string{"anime", "winter2025"}},
		{Title: "Pizza Toppings Ranking", CategorySlug: "food", CoverSlug: "pizza-toppings", SourcePostTitle: "Pizza Toppings Definitive Ranking", ParticipantCount: 48231, Tags: []string{"food", "pizza"}},
		{Title: "NBA All-Stars 2025", CategorySlug: "sports", CoverSlug: "nba-players-2025", SourcePostTitle: "NBA Players 2024-25 Season", ParticipantCount: 89012, Tags: []string{"nba", "basketball"}},
		{Title: "Best Albums of 2024", CategorySlug: "music", CoverSlug: "hiphop-albums-2024", SourcePostTitle: "2024 Hip-Hop Albums", ParticipantCount: 31456, Tags: []string{"music", "2024"}},
		{Title: "Video Games GOTY 2024", CategorySlug: "gaming", CoverSlug: "games-2024", SourcePostTitle: "Best Video Games of 2024", ParticipantCount: 67890, Tags: []string{"gaming", "goty"}},
	}

	leaderboard := []frontendDemoLeaderboardEntry{
		{Username: "animequeen", Rank: 1, Score: 98240, Change: "+2"},
		{Username: "tierqueen", Rank: 2, Score: 87103, Change: "-1"},
		{Username: "rankmaster99", Rank: 3, Score: 65892, Change: "+5"},
		{Username: "drip_scholar", Rank: 4, Score: 43210, Change: "+1"},
		{Username: "me", Rank: 5, Score: 28901, Change: "-2"},
	}

	return database.Transaction(func(tx *gorm.DB) error {
		userIDs := map[string]string{}
		for _, seed := range users {
			userID, err := upsertFrontendUser(tx, baseURL, now, seed)
			if err != nil {
				return err
			}
			userIDs[seed.Username] = userID
		}

		categoryIDs := map[string]string{}
		for _, seed := range categories {
			categoryID, err := upsertFrontendCategory(tx, now, seed)
			if err != nil {
				return err
			}
			categoryIDs[seed.Slug] = categoryID
		}

		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.DirectMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_user_id IN ?", valuesOf(userIDs)).Delete(&models.MessageThread{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.LeaderboardEntry{}).Error; err != nil {
			return err
		}

		for _, seed := range posts {
			if err := upsertFrontendPost(tx, baseURL, now, seed, userIDs, categoryIDs); err != nil {
				return err
			}
		}

		for _, seed := range topics {
			if err := upsertTrendingTopic(tx, baseURL, now, seed, categoryIDs); err != nil {
				return err
			}
		}

		for _, seed := range messages {
			if err := createDemoMessageThread(tx, now, seed, seed.OwnerUsername, seed.PeerUsername, seed.Unread, userIDs); err != nil {
				return err
			}
			if err := createDemoMessageThread(tx, now, seed, seed.PeerUsername, seed.OwnerUsername, 0, userIDs); err != nil {
				return err
			}
		}

		for _, seed := range leaderboard {
			entry := models.LeaderboardEntry{
				ID:        uuid.NewString(),
				UserID:    userIDs[seed.Username],
				Rank:      seed.Rank,
				Score:     seed.Score,
				Change:    seed.Change,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func createDemoMessageThread(tx *gorm.DB, now time.Time, seed frontendDemoMessage, ownerUsername string, peerUsername string, unread int, userIDs map[string]string) error {
	threadID := uuid.NewString()
	thread := models.MessageThread{
		ID:          threadID,
		OwnerUserID: userIDs[ownerUsername],
		PeerUserID:  userIDs[peerUsername],
		LastMessage: seed.LastMessage,
		UnreadCount: unread,
		UpdatedAt:   now.Add(-seed.Age),
		CreatedAt:   now.Add(-seed.Age),
	}
	if err := tx.Create(&thread).Error; err != nil {
		return err
	}
	for _, chat := range seed.History {
		message := models.DirectMessage{
			ID:           uuid.NewString(),
			ThreadID:     threadID,
			SenderUserID: userIDs[chat.SenderUsername],
			Body:         chat.Text,
			CreatedAt:    now.Add(-chat.Age),
		}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertFrontendUser(tx *gorm.DB, baseURL string, now time.Time, seed frontendDemoUser) (string, error) {
	var profile models.UserProfile
	err := tx.Where("username = ?", seed.Username).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	avatarURL := fmt.Sprintf("%s/assets/avatars/%s.svg", baseURL, seed.Username)
	displayName := seed.DisplayName
	bio := seed.Bio
	email := fmt.Sprintf("%s@rankster.local", seed.Username)
	userID := profile.UserID

	if err == gorm.ErrRecordNotFound {
		userID = uuid.NewString()
		user := models.User{ID: userID, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&user).Error; err != nil {
			return "", err
		}
		auth := models.UserAuth{ID: uuid.NewString(), UserID: userID, Provider: "LOCAL", Email: &email, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&auth).Error; err != nil {
			return "", err
		}
		profile = models.UserProfile{
			ID: uuid.NewString(), UserID: userID, Username: seed.Username, DisplayName: &displayName, Bio: &bio,
			AvatarURL: &avatarURL, Verified: seed.Verified, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return "", err
		}
		stats := models.UserStats{
			ID: uuid.NewString(), UserID: userID, RanksCreatedCount: seed.TotalRankings, FollowersCount: seed.Followers, FollowingCount: seed.Following, UpdatedAt: now,
		}
		if err := tx.Create(&stats).Error; err != nil {
			return "", err
		}
		sub := models.Subscription{
			ID: uuid.NewString(), UserID: userID, Plan: seed.Plan, Status: "ACTIVE", StartedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&sub).Error; err != nil {
			return "", err
		}
		return userID, nil
	}

	if err := tx.Model(&models.UserProfile{}).Where("user_id = ?", userID).Updates(map[string]any{
		"display_name": displayName,
		"bio":          bio,
		"avatar_url":   avatarURL,
		"verified":     seed.Verified,
		"updated_at":   now,
	}).Error; err != nil {
		return "", err
	}

	var stats models.UserStats
	if err := tx.Where("user_id = ?", userID).First(&stats).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return "", err
		}
		stats = models.UserStats{
			ID: uuid.NewString(), UserID: userID, RanksCreatedCount: seed.TotalRankings, FollowersCount: seed.Followers, FollowingCount: seed.Following, UpdatedAt: now,
		}
		if err := tx.Create(&stats).Error; err != nil {
			return "", err
		}
	} else if err := tx.Model(&models.UserStats{}).Where("user_id = ?", userID).Updates(map[string]any{
		"ranks_created_count": seed.TotalRankings,
		"followers_count":     seed.Followers,
		"following_count":     seed.Following,
		"updated_at":          now,
	}).Error; err != nil {
		return "", err
	}

	return userID, nil
}

func upsertFrontendCategory(tx *gorm.DB, now time.Time, seed frontendDemoCategory) (string, error) {
	var category models.Category
	err := tx.Where("slug = ?", seed.Slug).First(&category).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	emoji := seed.Emoji
	color := seed.Color
	if err == gorm.ErrRecordNotFound {
		category = models.Category{
			ID: uuid.NewString(), Slug: seed.Slug, Name: seed.Name, Emoji: &emoji, Color: &color, Tags: pq.StringArray(seed.Tags), CreatedAt: now, UpdatedAt: now,
		}
		return category.ID, tx.Create(&category).Error
	}

	return category.ID, tx.Model(&models.Category{}).Where("id = ?", category.ID).Updates(map[string]any{
		"name":       seed.Name,
		"emoji":      emoji,
		"color":      color,
		"tags":       pq.StringArray(seed.Tags),
		"updated_at": now,
	}).Error
}

func upsertFrontendPost(tx *gorm.DB, baseURL string, now time.Time, seed frontendDemoPost, userIDs map[string]string, categoryIDs map[string]string) error {
	var existing models.TierListPost
	if err := tx.Where("title = ?", seed.Title).First(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	} else if err == nil {
		return nil
	}

	createdAt := now.Add(-seed.Age)
	coverAssetID, err := ensureAsset(tx, fmt.Sprintf("%s/assets/ranks/%s.svg", baseURL, seed.CoverSlug), createdAt)
	if err != nil {
		return err
	}

	postID := uuid.NewString()
	post := models.Post{
		ID:         postID,
		Type:       "RANK",
		Visibility: "PUBLIC",
		CreatorID:  userIDs[seed.Username],
		CategoryID: categoryIDs[seed.CategorySlug],
		Caption:    stringPtr(seed.Description),
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
	if err := tx.Create(&post).Error; err != nil {
		return err
	}

	tierList := models.TierListPost{
		PostID:           postID,
		Title:            seed.Title,
		Description:      stringPtr(seed.Description),
		CoverAssetID:     &coverAssetID,
		Tags:             pq.StringArray(seed.Tags),
		ParticipantCount: seed.ParticipantCount,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
	if err := tx.Create(&tierList).Error; err != nil {
		return err
	}

	positionByExternalID := map[string]int{}
	for index, item := range seed.AllItems {
		positionByExternalID[item.ExternalID] = index
	}

	for tierKey, tierItems := range seed.Tiers {
		for index, item := range tierItems {
			listItem := models.TierListItem{
				ID:             uuid.NewString(),
				TierListPostID: postID,
				ExternalID:     item.ExternalID,
				Name:           item.Name,
				Emoji:          item.Emoji,
				TierKey:        tierKey,
				TierPosition:   index,
				ListPosition:   positionByExternalID[item.ExternalID],
				CreatedAt:      createdAt,
				UpdatedAt:      createdAt,
			}
			if err := tx.Create(&listItem).Error; err != nil {
				return err
			}
		}
	}

	metrics := models.PostMetrics{
		PostID:       postID,
		LikeCount:    seed.Likes,
		CommentCount: len(seed.Comments),
		ShareCount:   seed.Shares,
		HotScore:     float64(seed.Likes)/1000 + float64(seed.Shares)/100 + float64(seed.ParticipantCount)/10000,
		UpdatedAt:    createdAt,
	}
	if err := tx.Create(&metrics).Error; err != nil {
		return err
	}

	for _, username := range seed.LikedBy {
		like := models.PostLike{ID: uuid.NewString(), PostID: postID, UserID: userIDs[username], CreatedAt: createdAt}
		if err := tx.Create(&like).Error; err != nil {
			return err
		}
	}

	for _, commentSeed := range seed.Comments {
		comment := models.Comment{
			ID:        uuid.NewString(),
			PostID:    postID,
			AuthorID:  userIDs[commentSeed.Username],
			Body:      commentSeed.Text,
			LikeCount: commentSeed.Likes,
			CreatedAt: now.Add(-commentSeed.Age),
			UpdatedAt: now.Add(-commentSeed.Age),
		}
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
	}

	return nil
}

func upsertTrendingTopic(tx *gorm.DB, baseURL string, now time.Time, seed frontendDemoTopic, categoryIDs map[string]string) error {
	var existing models.TrendingTopic
	if err := tx.Where("title = ?", seed.Title).First(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	coverAssetID, err := ensureAsset(tx, fmt.Sprintf("%s/assets/ranks/%s.svg", baseURL, seed.CoverSlug), now)
	if err != nil {
		return err
	}

	sourcePostID, err := resolveTrendingTopicSourcePostID(tx, seed)
	if err != nil {
		return err
	}

	if existing.ID != "" {
		return tx.Model(&models.TrendingTopic{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"category_id":       categoryIDs[seed.CategorySlug],
			"cover_asset_id":    coverAssetID,
			"source_post_id":    sourcePostID,
			"participant_count": seed.ParticipantCount,
			"tags":              pq.StringArray(seed.Tags),
			"updated_at":        now,
		}).Error
	}

	topic := models.TrendingTopic{
		ID:               uuid.NewString(),
		Title:            seed.Title,
		CategoryID:       categoryIDs[seed.CategorySlug],
		CoverAssetID:     &coverAssetID,
		SourcePostID:     sourcePostID,
		ParticipantCount: seed.ParticipantCount,
		Tags:             pq.StringArray(seed.Tags),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return tx.Create(&topic).Error
}

func resolveTrendingTopicSourcePostID(tx *gorm.DB, seed frontendDemoTopic) (*string, error) {
	title := strings.TrimSpace(seed.SourcePostTitle)
	if title == "" {
		title = strings.TrimSpace(seed.Title)
	}
	if title == "" {
		return nil, nil
	}

	var list models.TierListPost
	err := tx.Where("title = ?", title).First(&list).Error
	if err == nil {
		return &list.PostID, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, err
}

func ensureAsset(tx *gorm.DB, url string, createdAt time.Time) (string, error) {
	var asset models.Asset
	err := tx.Where("url = ?", url).First(&asset).Error
	if err == nil {
		return asset.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", err
	}

	asset = models.Asset{ID: uuid.NewString(), URL: url, CreatedAt: createdAt}
	if err := tx.Create(&asset).Error; err != nil {
		return "", err
	}
	return asset.ID, nil
}

func valuesOf(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

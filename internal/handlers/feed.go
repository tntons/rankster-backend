package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
)

type FeedHandler struct {
	db *gorm.DB
}

func NewFeedHandler(db *gorm.DB) *FeedHandler {
	return &FeedHandler{db: db}
}

type feedCursor struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

func (h *FeedHandler) GetMainFeed(c *gin.Context) {
	limit := parseIntWithDefault(c.Query("limit"), 20)
	if limit < 1 {
		limit = 1
	}
	if limit > 50 {
		limit = 50
	}

	includeAds := true
	if raw := c.Query("includeAds"); raw != "" {
		includeAds = raw == "true" || raw == "1"
	}

	var cursor *feedCursor
	if raw := c.Query("cursor"); raw != "" {
		parsed, err := decodeCursor(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_CURSOR", "message": "cursor is invalid"})
			return
		}
		cursor = &parsed
	}

	authCtx := auth.FromAuthorization(c.GetHeader("Authorization"))

	now := time.Now()

	query := h.db.Model(&models.Post{}).
		Where("visibility = ?", "PUBLIC")

	if cursor != nil {
		createdAt, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
		if err == nil {
			query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", createdAt, createdAt, cursor.ID)
		}
	}

	var posts []models.Post
	if err := applyPostPreloads(query).
		Order("created_at desc, id desc").
		Limit(limit + 1).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DB_ERROR", "message": "failed to load feed"})
		return
	}

	hasMore := len(posts) > limit
	page := posts
	if hasMore {
		page = posts[:limit]
	}

	var nextCursor string
	if hasMore && len(page) > 0 {
		last := page[len(page)-1]
		nextCursor = encodeCursor(feedCursor{
			CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339Nano),
			ID:        last.ID,
		})
	}

	isAdFree := false
	if authCtx.Kind == "user" {
		var sub models.Subscription
		err := h.db.Where("user_id = ? AND status = ? AND plan IN ?", authCtx.UserID, "ACTIVE", []string{"PRO", "BUSINESS"}).
			Select("id").
			First(&sub).Error
		if err == nil {
			isAdFree = true
		}
	}

	items, err := buildFeedItems(h.db, page, authCtx, includeAds && !isAdFree, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FEED_BUILD_ERROR", "message": "failed to build feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"nextCursor": emptyToNull(nextCursor),
	})
}

func applyPostPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Creator.Profile").
		Preload("Category").
		Preload("Metrics").
		Preload("Rank.Image").
		Preload("Survey.SponsorOrg").
		Preload("Survey.Campaign").
		Preload("Survey.Questions", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("`order` asc")
		}).
		Preload("Survey.Questions.Options", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("`order` asc")
		})
}

func buildFeedItems(db *gorm.DB, posts []models.Post, authCtx auth.Context, includeAds bool, now time.Time) ([]gin.H, error) {
	items := make([]gin.H, 0, len(posts))
	injectionInterval := 7
	organicSinceAd := 0
	feedRequestID := uuid.NewString()
	seenSurveyPostIDs := map[string]struct{}{}

	for _, p := range posts {
		postView := toPostView(p)
		if p.Type == "SURVEY" {
			items = append(items, gin.H{"kind": "survey", "post": postView, "campaign": gin.H{"campaignId": campaignID(p)}})
		} else {
			items = append(items, gin.H{"kind": "rank", "post": postView})
		}

		organicSinceAd++
		if !includeAds || organicSinceAd < injectionInterval {
			continue
		}

		ad, err := pickEligibleSurveyAd(db, now, seenSurveyPostIDs)
		if err != nil {
			return nil, err
		}
		if ad == nil {
			organicSinceAd = 0
			continue
		}

		seenSurveyPostIDs[ad.SurveyPostID] = struct{}{}

		impression := models.SurveyImpression{
			ID:            uuid.NewString(),
			CampaignID:    ad.ID,
			FeedRequestID: &feedRequestID,
			CreatedAt:     time.Now(),
		}
		if authCtx.Kind == "user" {
			impression.UserID = &authCtx.UserID
		}
		if err := db.Create(&impression).Error; err != nil {
			return nil, err
		}

		injected := toPostView(ad.SurveyPost.Post)
		payload := gin.H{"kind": "survey", "post": injected, "campaign": gin.H{"campaignId": ad.ID}}
		if ad.SurveyPost.SponsorOrg != nil {
			payload["campaign"] = gin.H{
				"campaignId": ad.ID,
				"sponsoredBy": gin.H{"organizationId": ad.SurveyPost.SponsorOrg.ID, "name": ad.SurveyPost.SponsorOrg.Name},
			}
		}
		items = append(items, payload)
		organicSinceAd = 0
	}

	return items, nil
}

func pickEligibleSurveyAd(db *gorm.DB, now time.Time, exclude map[string]struct{}) (*models.SurveyCampaign, error) {
	var campaigns []models.SurveyCampaign
	if err := db.
		Where("start_at <= ?", now).
		Where("(end_at IS NULL OR end_at > ?)", now).
		Where("budget_cents > 0").
		Where("spent_cents < budget_cents").
		Order("updated_at desc").
		Limit(10).
		Preload("SurveyPost").
		Preload("SurveyPost.Post", func(tx *gorm.DB) *gorm.DB {
			return applyPostPreloads(tx)
		}).
		Preload("SurveyPost.SponsorOrg").
		Find(&campaigns).Error; err != nil {
		return nil, err
	}

	for i := range campaigns {
		campaign := &campaigns[i]
		if _, ok := exclude[campaign.SurveyPostID]; ok {
			continue
		}
		if campaign.SurveyPost.Post.Visibility != "PUBLIC" {
			continue
		}
		return campaign, nil
	}
	return nil, nil
}

func encodeCursor(cursor feedCursor) string {
	payload, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(raw string) (feedCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return feedCursor{}, err
	}
	var cursor feedCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return feedCursor{}, err
	}
	return cursor, nil
}

func parseIntWithDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func emptyToNull(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func campaignID(post models.Post) string {
	if post.Survey != nil && post.Survey.Campaign != nil {
		return post.Survey.Campaign.ID
	}
	return "unknown"
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) SearchOverview(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	response, err := h.search(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to search"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *FrontendHandler) GetTrendingTopics(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	items, err := h.trendingTopics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load topics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) GetCategories(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	items, err := h.categories(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to load categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *FrontendHandler) search(query string) (frontendSearchResponse, error) {
	categories, err := h.categories(query)
	if err != nil {
		return frontendSearchResponse{}, err
	}

	topics, err := h.trendingTopicsFiltered(query, 6)
	if err != nil {
		return frontendSearchResponse{}, err
	}

	var users []models.User
	userQuery := h.db.Preload("Profile").Preload("Stats").Joins("JOIN user_profiles ON user_profiles.user_id = users.id")
	if query != "" {
		like := "%" + query + "%"
		userQuery = userQuery.Where("LOWER(user_profiles.username) LIKE ? OR LOWER(COALESCE(user_profiles.display_name, '')) LIKE ? OR LOWER(COALESCE(user_profiles.bio, '')) LIKE ?", like, like, like)
	}
	if err := userQuery.Limit(5).Find(&users).Error; err != nil {
		return frontendSearchResponse{}, err
	}

	response := frontendSearchResponse{
		Users:      make([]frontendUserView, 0, len(users)),
		Topics:     topics,
		Categories: categories,
	}
	for _, user := range users {
		response.Users = append(response.Users, buildFrontendUser(user))
	}
	return response, nil
}

func (h *FrontendHandler) trendingTopics() ([]frontendTrendingTopicView, error) {
	return h.trendingTopicsFiltered("", 100)
}

func (h *FrontendHandler) trendingTopicsFiltered(query string, limit int) ([]frontendTrendingTopicView, error) {
	var topics []models.TrendingTopic
	db := h.db.Preload("Category").Preload("CoverAsset").Preload("SourcePost").Order("participant_count desc")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(title) LIKE ? OR EXISTS (SELECT 1 FROM unnest(tags) tag WHERE LOWER(tag) LIKE ?)", like, like)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&topics).Error; err != nil {
		return nil, err
	}

	items := make([]frontendTrendingTopicView, 0, len(topics))
	for _, topic := range topics {
		items = append(items, frontendTrendingTopicView{
			ID:               topic.ID,
			PostID:           topic.SourcePostID,
			Title:            topic.Title,
			Category:         topic.Category.Slug,
			CoverImage:       assetOrFallback(topic.CoverAsset, "ranks", slugify(topic.Title)),
			ParticipantCount: topic.ParticipantCount,
			Tags:             append([]string{}, topic.Tags...),
		})
	}
	return items, nil
}

func (h *FrontendHandler) categories(query string) ([]frontendCategoryView, error) {
	var categories []models.Category
	db := h.db.Order("name asc")
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ?", like, like)
	}
	if err := db.Find(&categories).Error; err != nil {
		return nil, err
	}

	items := make([]frontendCategoryView, 0, len(categories))
	for _, category := range categories {
		emoji := ""
		if category.Emoji != nil {
			emoji = *category.Emoji
		}
		color := ""
		if category.Color != nil {
			color = *category.Color
		}
		items = append(items, frontendCategoryView{
			ID:    category.Slug,
			Name:  category.Name,
			Emoji: emoji,
			Color: color,
		})
	}
	if len(items) > 6 && query != "" {
		items = items[:6]
	}
	return items, nil
}

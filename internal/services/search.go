package services

import (
	"sort"
	"strings"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type SearchService struct {
	search *repositories.SearchRepository
}

func NewSearchService(search *repositories.SearchRepository) *SearchService {
	return &SearchService{search: search}
}

func (s *SearchService) Search(query string) (views.SearchResponse, error) {
	normalizedQuery := normalizeSearchQuery(query)
	categories, err := s.Categories(normalizedQuery)
	if err != nil {
		return views.SearchResponse{}, err
	}

	topics, err := s.TrendingTopicsFiltered(normalizedQuery, 6)
	if err != nil {
		return views.SearchResponse{}, err
	}

	users, err := s.search.Users(normalizedQuery, 5)
	if err != nil {
		return views.SearchResponse{}, err
	}

	response := views.SearchResponse{
		Users:      make([]views.User, 0, len(users)),
		Topics:     topics,
		Categories: categories,
	}
	for _, user := range users {
		response.Users = append(response.Users, views.BuildUser(user))
	}
	return response, nil
}

func (s *SearchService) TrendingTopics() ([]views.TrendingTopic, error) {
	return s.TrendingTopicsFiltered("", 100)
}

func (s *SearchService) TrendingTopicsFiltered(query string, limit int) ([]views.TrendingTopic, error) {
	normalizedQuery := normalizeSearchQuery(query)
	rankTopics, err := s.search.RankTopics("", 0)
	if err != nil {
		return nil, err
	}

	items := buildAggregatedTopics(rankTopics, normalizedQuery)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func normalizeSearchQuery(query string) string {
	query = strings.TrimSpace(strings.ToLower(query))
	query = strings.TrimPrefix(query, "#")
	return query
}

type aggregatedTopic struct {
	topicID        string
	representative models.TierListPost
	count          int
	matchesQuery   bool
}

func buildAggregatedTopics(lists []models.TierListPost, query string) []views.TrendingTopic {
	groups := map[string]*aggregatedTopic{}
	for _, list := range lists {
		topicID := ""
		if list.TopicID != nil {
			topicID = strings.TrimSpace(*list.TopicID)
		}
		if topicID == "" {
			topicID = list.PostID
		}

		group, exists := groups[topicID]
		if !exists {
			copyList := list
			group = &aggregatedTopic{
				topicID:        topicID,
				representative: copyList,
			}
			groups[topicID] = group
		}
		group.count++
		if query == "" || rankTopicMatchesQuery(list, query) {
			group.matchesQuery = true
		}

		if list.PostID == topicID || (group.representative.CoverAssetID == nil && list.CoverAssetID != nil) {
			group.representative = list
		}
	}

	grouped := make([]*aggregatedTopic, 0, len(groups))
	for _, group := range groups {
		if !group.matchesQuery {
			continue
		}
		grouped = append(grouped, group)
	}

	sort.SliceStable(grouped, func(i, j int) bool {
		if grouped[i].count != grouped[j].count {
			return grouped[i].count > grouped[j].count
		}
		return grouped[i].representative.CreatedAt.After(grouped[j].representative.CreatedAt)
	})

	items := make([]views.TrendingTopic, 0, len(grouped))
	for _, group := range grouped {
		items = append(items, views.BuildRankPostTopicWithCount(group.representative, group.topicID, group.count))
	}
	return items
}

func rankTopicMatchesQuery(list models.TierListPost, query string) bool {
	values := []string{
		list.Title,
		list.Post.Category.Name,
		list.Post.Category.Slug,
	}
	values = append(values, list.Tags...)

	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func (s *SearchService) Categories(query string) ([]views.Category, error) {
	categories, err := s.search.Categories(query)
	if err != nil {
		return nil, err
	}

	items := make([]views.Category, 0, len(categories))
	for _, category := range categories {
		items = append(items, views.BuildCategory(category))
	}
	if len(items) > 6 && query != "" {
		items = items[:6]
	}
	return items, nil
}

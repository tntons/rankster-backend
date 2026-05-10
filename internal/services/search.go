package services

import (
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
	categories, err := s.Categories(query)
	if err != nil {
		return views.SearchResponse{}, err
	}

	topics, err := s.TrendingTopicsFiltered(query, 6)
	if err != nil {
		return views.SearchResponse{}, err
	}

	users, err := s.search.Users(query, 5)
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
	topics, err := s.search.TrendingTopics(query, limit)
	if err != nil {
		return nil, err
	}

	rankTopics, err := s.search.RankTopics(query, limit)
	if err != nil {
		return nil, err
	}

	items := make([]views.TrendingTopic, 0, len(topics)+len(rankTopics))
	seenPostIDs := map[string]bool{}
	seenTopicIDs := map[string]bool{}

	appendRankTopics := func() {
		for _, rankTopic := range rankTopics {
			if seenPostIDs[rankTopic.PostID] || seenTopicIDs[rankTopic.PostID] {
				continue
			}
			item := views.BuildRankPostTopic(rankTopic)
			items = append(items, item)
			seenPostIDs[rankTopic.PostID] = true
			seenTopicIDs[item.ID] = true
		}
	}
	appendTrendingTopics := func() {
		for _, topic := range topics {
			if topic.SourcePostID != nil && seenPostIDs[*topic.SourcePostID] {
				continue
			}
			item := views.BuildTrendingTopic(topic)
			if seenTopicIDs[item.ID] {
				continue
			}
			items = append(items, item)
			seenTopicIDs[item.ID] = true
			if topic.SourcePostID != nil {
				seenPostIDs[*topic.SourcePostID] = true
			}
		}
	}

	if query != "" {
		appendRankTopics()
		appendTrendingTopics()
	} else {
		appendTrendingTopics()
		appendRankTopics()
	}

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
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

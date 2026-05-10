package services

import (
	"encoding/base64"
	"fmt"
	"strings"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type FeedService struct {
	tierLists *repositories.TierListRepository
	rankPosts *RankPostService
}

func NewFeedService(tierLists *repositories.TierListRepository, rankPosts *RankPostService) *FeedService {
	return &FeedService{tierLists: tierLists, rankPosts: rankPosts}
}

func (s *FeedService) MainFeed(scope string, category string, offset, limit int, authUser *views.User) (views.FeedResponse, error) {
	if limit < 1 {
		limit = 20
	}
	category = strings.TrimSpace(strings.ToLower(category))

	if scope == "following" && authUser == nil {
		return views.FeedResponse{Items: []views.RankPost{}, NextCursor: nil}, nil
	}

	var (
		lists   []models.TierListPost
		hasMore bool
		err     error
	)

	switch scope {
	case "following":
		lists, hasMore, err = s.tierLists.FollowingFeed(authUser.ID, offset, limit, category)
	default:
		lists, hasMore, err = s.tierLists.Feed(offset, limit, category)
	}
	if err != nil {
		return views.FeedResponse{}, err
	}

	items, err := s.rankPosts.HydrateTierLists(lists, authUser)
	if err != nil {
		return views.FeedResponse{}, err
	}

	var nextCursor any
	if hasMore {
		nextCursor = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset+limit)))
	}

	return views.FeedResponse{Items: items, NextCursor: nextCursor}, nil
}

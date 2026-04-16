package services

import (
	"slices"
	"strconv"
	"strings"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type LeaderboardService struct {
	leaderboard *repositories.LeaderboardRepository
}

func NewLeaderboardService(leaderboard *repositories.LeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{leaderboard: leaderboard}
}

func (s *LeaderboardService) Leaderboard(timeframe string, category string) ([]views.LeaderboardEntry, error) {
	entries, err := s.leaderboard.Entries()
	if err != nil {
		return nil, err
	}

	items := make([]views.LeaderboardEntry, 0, len(entries))
	for _, entry := range entries {
		score := adjustedLeaderboardScore(entry, timeframe, category)
		change := adjustedLeaderboardChange(entry, timeframe, category)
		items = append(items, views.LeaderboardEntry{
			Rank:   entry.Rank,
			User:   views.BuildUser(entry.User),
			Score:  score,
			Change: change,
		})
	}

	slices.SortFunc(items, func(a, b views.LeaderboardEntry) int {
		if a.Score == b.Score {
			return strings.Compare(a.User.Username, b.User.Username)
		}
		return b.Score - a.Score
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return items, nil
}

func adjustedLeaderboardScore(entry models.LeaderboardEntry, timeframe string, category string) int {
	score := entry.Score

	switch timeframe {
	case "this-month":
		score += 4200 - (entry.Rank * 350)
	case "all-time":
		score += 9100 - (entry.Rank * 500)
	default:
		score += 1800 - (entry.Rank * 200)
	}

	if category != "" && category != "all" {
		score += stableValue(entry.UserID+":"+category) % 6000
	}

	return max(score, 1)
}

func adjustedLeaderboardChange(entry models.LeaderboardEntry, timeframe string, category string) string {
	base := stableValue(entry.UserID + ":" + timeframe + ":" + category)
	delta := (base % 7) - 3
	if delta > 0 {
		return "+" + strconv.Itoa(delta)
	}
	if delta < 0 {
		return strconv.Itoa(delta)
	}
	return "0"
}

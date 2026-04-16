package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"rankster-backend/internal/repositories"
	"rankster-backend/internal/views"
)

type ProfileService struct {
	db            *gorm.DB
	users         *repositories.UserRepository
	profiles      *repositories.ProfileRepository
	rankPosts     *RankPostService
	notifications *NotificationService
}

type FollowResult struct {
	IsFollowing             bool
	NotificationRecipientID string
	Notification            *views.Notification
}

func NewProfileService(
	db *gorm.DB,
	users *repositories.UserRepository,
	profiles *repositories.ProfileRepository,
	rankPosts *RankPostService,
	notifications *NotificationService,
) *ProfileService {
	return &ProfileService{
		db:            db,
		users:         users,
		profiles:      profiles,
		rankPosts:     rankPosts,
		notifications: notifications,
	}
}

func (s *ProfileService) BuildProfile(profileUserID string, authUser *views.User) (views.ProfileResponse, error) {
	userRecord, err := s.users.FindByID(profileUserID)
	if err != nil {
		return views.ProfileResponse{}, err
	}

	user := views.BuildUser(userRecord)
	rankings, err := s.rankPosts.RankingsForCreator(profileUserID, authUser)
	if err != nil {
		return views.ProfileResponse{}, err
	}

	likedPosts, err := s.rankPosts.LikedRankingsForUser(profileUserID, authUser)
	if err != nil {
		return views.ProfileResponse{}, err
	}

	stats, err := s.rankPosts.UserStats(profileUserID)
	if err != nil {
		return views.ProfileResponse{}, err
	}

	favoriteCategories, err := s.FavoriteCategoriesForUser(profileUserID)
	if err != nil {
		return views.ProfileResponse{}, err
	}

	pinnedPostID, err := s.profiles.PinnedPostID(profileUserID)
	if err != nil {
		return views.ProfileResponse{}, err
	}

	isFollowing := false
	if authUser != nil && authUser.ID != profileUserID {
		isFollowing, err = s.profiles.FollowState(authUser.ID, profileUserID)
		if err != nil {
			return views.ProfileResponse{}, err
		}
	}

	return views.ProfileResponse{
		User:         user,
		Rankings:     rankings,
		LikedPosts:   likedPosts,
		PinnedPostID: pinnedPostID,
		Stats: views.ProfileStats{
			TotalRankings: stats.RanksCreated,
			Followers:     stats.Followers,
			Following:     stats.Following,
			TotalLikes:    stats.LikesReceived,
		},
		FavoriteCategories: favoriteCategories,
		IsFollowing:        isFollowing,
	}, nil
}

func (s *ProfileService) FavoriteCategoriesForUser(userID string) ([]views.ProfileCategory, error) {
	rows, err := s.profiles.FavoriteCategories(userID, 4)
	if err != nil {
		return nil, err
	}

	total := 0
	for _, row := range rows {
		total += row.Count
	}

	out := make([]views.ProfileCategory, 0, len(rows))
	for _, row := range rows {
		pct := 0
		if total > 0 {
			pct = int(float64(row.Count) / float64(total) * 100)
		}
		out = append(out, views.ProfileCategory{
			ID:    row.ID,
			Name:  row.Name,
			Emoji: row.Emoji,
			Pct:   pct,
		})
	}
	return out, nil
}

func (s *ProfileService) PinnedPostIDForUser(userID string) (*string, error) {
	return s.profiles.PinnedPostID(userID)
}

func (s *ProfileService) FollowState(followerID, followingID string) (bool, error) {
	return s.profiles.FollowState(followerID, followingID)
}

func (s *ProfileService) UpdateCurrentProfile(userID string, displayName string, bio string, avatar string) error {
	return s.profiles.UpdateCurrentProfile(userID, displayName, bio, avatar)
}

func (s *ProfileService) SetFollowState(followerID, followingID string, shouldFollow bool) (bool, error) {
	return s.profiles.SetFollowState(followerID, followingID, shouldFollow, generateUUID)
}

func (s *ProfileService) Follow(authUser views.User, targetUserID string) (FollowResult, error) {
	changed, err := s.profiles.SetFollowState(authUser.ID, targetUserID, true, generateUUID)
	if err != nil {
		return FollowResult{}, err
	}

	result := FollowResult{IsFollowing: true}
	if changed && s.notifications != nil {
		result.NotificationRecipientID = targetUserID
		notification, err := s.notifications.Create(
			s.db,
			targetUserID,
			&authUser.ID,
			"follow",
			"New follower",
			fmt.Sprintf("%s started following you.", authUser.DisplayName),
			"/profile/"+authUser.Username,
			time.Now(),
		)
		if err != nil {
			return FollowResult{}, err
		}
		result.Notification = notification
	}

	return result, nil
}

func (s *ProfileService) Unfollow(authUserID string, targetUserID string) (FollowResult, error) {
	if _, err := s.profiles.SetFollowState(authUserID, targetUserID, false, generateUUID); err != nil {
		return FollowResult{}, err
	}
	return FollowResult{IsFollowing: false}, nil
}

func (s *ProfileService) SetPinnedPost(userID, postID string, shouldPin bool) error {
	return s.profiles.SetPinnedPost(userID, postID, shouldPin, generateUUID)
}

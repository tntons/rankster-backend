package handlers

import (
	"sync"

	"rankster-backend/internal/services"
	"rankster-backend/internal/views"
)

var errForbidden = services.ErrForbidden

type userView = views.User
type tierItemView = views.TierItem
type tierDataView = views.TierData
type commentView = views.Comment
type commentLikeResponse = views.CommentLikeResponse
type rankPostView = views.RankPost
type messageThreadView = views.MessageThread
type chatMessageView = views.ChatMessage
type chatSocketEvent = views.ChatSocketEvent
type messageInboxSocketEvent = views.MessageInboxSocketEvent

type chatClient struct {
	threadID string
	send     chan chatSocketEvent
}

type chatHub struct {
	mu      sync.RWMutex
	clients map[string]map[*chatClient]struct{}
}

type messageInboxClient struct {
	userID string
	send   chan messageInboxSocketEvent
}

type messageInboxHub struct {
	mu      sync.RWMutex
	clients map[string]map[*messageInboxClient]struct{}
}

type messageThreadDetailView = views.MessageThreadDetail
type notificationView = views.Notification
type notificationsResponse = views.NotificationsResponse
type notificationSocketEvent = views.NotificationSocketEvent

type notificationClient struct {
	userID string
	send   chan notificationSocketEvent
}

type notificationHub struct {
	mu      sync.RWMutex
	clients map[string]map[*notificationClient]struct{}
}

type createdMessageView = views.CreatedMessage
type trendingTopicView = views.TrendingTopic
type categoryView = views.Category
type profileResponse = views.ProfileResponse
type profileStatsView = views.ProfileStats
type profileCategoryView = views.ProfileCategory
type searchResponse = views.SearchResponse
type feedResponse = views.FeedResponse
type authResponse = views.AuthResponse
type googleAuthRequest = views.GoogleAuthRequest
type updateProfileRequest = views.UpdateProfileRequest
type leaderboardEntry = views.LeaderboardEntry
type createRankRequest = views.CreateRankRequest
type createCommentRequest = views.CreateCommentRequest

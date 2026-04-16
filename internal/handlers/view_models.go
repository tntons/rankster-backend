package handlers

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

var errForbidden = errors.New("forbidden")

type frontendUserView struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	DisplayName   string `json:"displayName"`
	Avatar        string `json:"avatar"`
	Bio           string `json:"bio"`
	Followers     int    `json:"followers"`
	Following     int    `json:"following"`
	TotalRankings int    `json:"totalRankings"`
	Verified      bool   `json:"verified"`
}

type frontendTierItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Emoji    *string `json:"emoji,omitempty"`
	ImageURL *string `json:"imageUrl,omitempty"`
}

type frontendTierData struct {
	S []frontendTierItem `json:"S"`
	A []frontendTierItem `json:"A"`
	B []frontendTierItem `json:"B"`
	C []frontendTierItem `json:"C"`
	D []frontendTierItem `json:"D"`
}

type frontendCommentView struct {
	ID        string           `json:"id"`
	User      frontendUserView `json:"user"`
	Text      string           `json:"text"`
	CreatedAt string           `json:"createdAt"`
	Likes     int              `json:"likes"`
	IsLiked   bool             `json:"isLiked"`
}

type frontendCommentLikeResponse struct {
	Likes   int  `json:"likes"`
	IsLiked bool `json:"isLiked"`
}

type frontendRankPostView struct {
	ID               string                `json:"id"`
	User             frontendUserView      `json:"user"`
	Title            string                `json:"title"`
	Category         string                `json:"category"`
	CoverImage       string                `json:"coverImage"`
	Tiers            frontendTierData      `json:"tiers"`
	AllItems         []frontendTierItem    `json:"allItems"`
	Description      string                `json:"description"`
	Tags             []string              `json:"tags"`
	Likes            int                   `json:"likes"`
	IsLiked          bool                  `json:"isLiked"`
	Comments         []frontendCommentView `json:"comments"`
	Shares           int                   `json:"shares"`
	CreatedAt        string                `json:"createdAt"`
	IsPublic         bool                  `json:"isPublic"`
	ParticipantCount int                   `json:"participantCount"`
	CanEdit          bool                  `json:"canEdit"`
}

type frontendMessageView struct {
	ID          string           `json:"id"`
	User        frontendUserView `json:"user"`
	LastMessage string           `json:"lastMessage"`
	Timestamp   string           `json:"timestamp"`
	Unread      int              `json:"unread"`
}

type frontendChatMessageView struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Mine      bool   `json:"mine"`
	Timestamp string `json:"timestamp"`
}

type frontendChatSocketEvent struct {
	Type      string                   `json:"type"`
	ThreadID  string                   `json:"threadId"`
	Message   *frontendChatMessageView `json:"message,omitempty"`
	Error     *string                  `json:"error,omitempty"`
	Timestamp string                   `json:"timestamp"`
}

type frontendMessageInboxSocketEvent struct {
	Type        string               `json:"type"`
	Thread      *frontendMessageView `json:"thread,omitempty"`
	UnreadCount int                  `json:"unreadCount"`
	Timestamp   string               `json:"timestamp"`
}

type frontendChatClient struct {
	threadID string
	send     chan frontendChatSocketEvent
}

type frontendChatHub struct {
	mu      sync.RWMutex
	clients map[string]map[*frontendChatClient]struct{}
}

type frontendMessageInboxClient struct {
	userID string
	send   chan frontendMessageInboxSocketEvent
}

type frontendMessageInboxHub struct {
	mu      sync.RWMutex
	clients map[string]map[*frontendMessageInboxClient]struct{}
}

type frontendMessageThreadDetailView struct {
	ID       string                    `json:"id"`
	User     frontendUserView          `json:"user"`
	Messages []frontendChatMessageView `json:"messages"`
}

type frontendNotificationView struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Actor     *frontendUserView `json:"actor,omitempty"`
	Href      string            `json:"href"`
	CreatedAt string            `json:"createdAt"`
	Read      bool              `json:"read"`
}

type frontendNotificationsResponse struct {
	Items       []frontendNotificationView `json:"items"`
	UnreadCount int                        `json:"unreadCount"`
}

type frontendNotificationSocketEvent struct {
	Type         string                    `json:"type"`
	Notification *frontendNotificationView `json:"notification,omitempty"`
	UnreadCount  int                       `json:"unreadCount"`
	Timestamp    string                    `json:"timestamp"`
}

type frontendNotificationClient struct {
	userID string
	send   chan frontendNotificationSocketEvent
}

type frontendNotificationHub struct {
	mu      sync.RWMutex
	clients map[string]map[*frontendNotificationClient]struct{}
}

type frontendCreatedMessage struct {
	Sender            frontendChatMessageView
	Recipient         *frontendChatMessageView
	RecipientThreadID *string
	RecipientUserID   *string
	RecipientThread   *frontendMessageView
}

type frontendTrendingTopicView struct {
	ID               string   `json:"id"`
	PostID           *string  `json:"postId,omitempty"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	CoverImage       string   `json:"coverImage"`
	ParticipantCount int      `json:"participantCount"`
	Tags             []string `json:"tags"`
}

type frontendCategoryView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Color string `json:"color"`
}

type frontendProfileResponse struct {
	User               frontendUserView              `json:"user"`
	Rankings           []frontendRankPostView        `json:"rankings"`
	LikedPosts         []frontendRankPostView        `json:"likedPosts"`
	PinnedPostID       *string                       `json:"pinnedPostId"`
	Stats              frontendProfileStatsView      `json:"stats"`
	FavoriteCategories []frontendProfileCategoryView `json:"favoriteCategories"`
	IsFollowing        bool                          `json:"isFollowing"`
}

type frontendProfileStatsView struct {
	TotalRankings int `json:"totalRankings"`
	Followers     int `json:"followers"`
	Following     int `json:"following"`
	TotalLikes    int `json:"totalLikes"`
}

type frontendProfileCategoryView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Pct   int    `json:"pct"`
}

type frontendSearchResponse struct {
	Users      []frontendUserView          `json:"users"`
	Topics     []frontendTrendingTopicView `json:"topics"`
	Categories []frontendCategoryView      `json:"categories"`
}

type frontendFeedResponse struct {
	Items      []frontendRankPostView `json:"items"`
	NextCursor any                    `json:"nextCursor"`
}

type frontendAuthResponse struct {
	AccessToken string           `json:"accessToken"`
	TokenType   string           `json:"tokenType"`
	User        frontendUserView `json:"user"`
}

type frontendGoogleAuthRequest struct {
	Credential string `json:"credential"`
}

type frontendUpdateProfileRequest struct {
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
}

type frontendLeaderboardEntry struct {
	Rank   int              `json:"rank"`
	User   frontendUserView `json:"user"`
	Score  int              `json:"score"`
	Change string           `json:"change"`
}

type frontendCreateRankRequest struct {
	Title        string             `json:"title"`
	Category     string             `json:"category"`
	Description  string             `json:"description"`
	Tags         []string           `json:"tags"`
	Tiers        frontendTierData   `json:"tiers"`
	AllItems     []frontendTierItem `json:"allItems"`
	IsPublic     *bool              `json:"isPublic"`
	SourcePostID string             `json:"sourcePostId"`
}

type frontendCreateCommentRequest struct {
	Text string `json:"text"`
}

type FrontendHandler struct {
	db              *gorm.DB
	publicBaseURL   string
	uploadDir       string
	googleClientID  string
	authTokenSecret string
	chatHub         *frontendChatHub
	messageInboxHub *frontendMessageInboxHub
	notificationHub *frontendNotificationHub
}

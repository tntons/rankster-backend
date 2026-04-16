package views

type User struct {
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

type TierItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Emoji    *string `json:"emoji,omitempty"`
	ImageURL *string `json:"imageUrl,omitempty"`
}

type TierData struct {
	S []TierItem `json:"S"`
	A []TierItem `json:"A"`
	B []TierItem `json:"B"`
	C []TierItem `json:"C"`
	D []TierItem `json:"D"`
}

type Comment struct {
	ID        string `json:"id"`
	User      User   `json:"user"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
	Likes     int    `json:"likes"`
	IsLiked   bool   `json:"isLiked"`
}

type CommentLikeResponse struct {
	Likes   int  `json:"likes"`
	IsLiked bool `json:"isLiked"`
}

type RankPost struct {
	ID               string     `json:"id"`
	User             User       `json:"user"`
	Title            string     `json:"title"`
	Category         string     `json:"category"`
	CoverImage       string     `json:"coverImage"`
	Tiers            TierData   `json:"tiers"`
	AllItems         []TierItem `json:"allItems"`
	Description      string     `json:"description"`
	Tags             []string   `json:"tags"`
	Likes            int        `json:"likes"`
	IsLiked          bool       `json:"isLiked"`
	Comments         []Comment  `json:"comments"`
	Shares           int        `json:"shares"`
	CreatedAt        string     `json:"createdAt"`
	IsPublic         bool       `json:"isPublic"`
	ParticipantCount int        `json:"participantCount"`
	CanEdit          bool       `json:"canEdit"`
}

type MessageThread struct {
	ID          string `json:"id"`
	User        User   `json:"user"`
	LastMessage string `json:"lastMessage"`
	Timestamp   string `json:"timestamp"`
	Unread      int    `json:"unread"`
}

type ChatMessage struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Mine      bool   `json:"mine"`
	Timestamp string `json:"timestamp"`
}

type ChatSocketEvent struct {
	Type      string       `json:"type"`
	ThreadID  string       `json:"threadId"`
	Message   *ChatMessage `json:"message,omitempty"`
	Error     *string      `json:"error,omitempty"`
	Timestamp string       `json:"timestamp"`
}

type MessageInboxSocketEvent struct {
	Type        string         `json:"type"`
	Thread      *MessageThread `json:"thread,omitempty"`
	UnreadCount int            `json:"unreadCount"`
	Timestamp   string         `json:"timestamp"`
}

type MessageThreadDetail struct {
	ID       string        `json:"id"`
	User     User          `json:"user"`
	Messages []ChatMessage `json:"messages"`
}

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Actor     *User  `json:"actor,omitempty"`
	Href      string `json:"href"`
	CreatedAt string `json:"createdAt"`
	Read      bool   `json:"read"`
}

type NotificationsResponse struct {
	Items       []Notification `json:"items"`
	UnreadCount int            `json:"unreadCount"`
}

type NotificationSocketEvent struct {
	Type         string        `json:"type"`
	Notification *Notification `json:"notification,omitempty"`
	UnreadCount  int           `json:"unreadCount"`
	Timestamp    string        `json:"timestamp"`
}

type CreatedMessage struct {
	Sender            ChatMessage
	Recipient         *ChatMessage
	RecipientThreadID *string
	RecipientUserID   *string
	RecipientThread   *MessageThread
}

type TrendingTopic struct {
	ID               string   `json:"id"`
	PostID           *string  `json:"postId,omitempty"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	CoverImage       string   `json:"coverImage"`
	ParticipantCount int      `json:"participantCount"`
	Tags             []string `json:"tags"`
}

type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Color string `json:"color"`
}

type ProfileResponse struct {
	User               User              `json:"user"`
	Rankings           []RankPost        `json:"rankings"`
	LikedPosts         []RankPost        `json:"likedPosts"`
	PinnedPostID       *string           `json:"pinnedPostId"`
	Stats              ProfileStats      `json:"stats"`
	FavoriteCategories []ProfileCategory `json:"favoriteCategories"`
	IsFollowing        bool              `json:"isFollowing"`
}

type ProfileStats struct {
	TotalRankings int `json:"totalRankings"`
	Followers     int `json:"followers"`
	Following     int `json:"following"`
	TotalLikes    int `json:"totalLikes"`
}

type ProfileCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
	Pct   int    `json:"pct"`
}

type SearchResponse struct {
	Users      []User          `json:"users"`
	Topics     []TrendingTopic `json:"topics"`
	Categories []Category      `json:"categories"`
}

type FeedResponse struct {
	Items      []RankPost `json:"items"`
	NextCursor any        `json:"nextCursor"`
}

type AuthResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	User        User   `json:"user"`
}

type GoogleAuthRequest struct {
	Credential string `json:"credential"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
}

type LeaderboardEntry struct {
	Rank   int    `json:"rank"`
	User   User   `json:"user"`
	Score  int    `json:"score"`
	Change string `json:"change"`
}

type CreateRankRequest struct {
	Title        string     `json:"title"`
	Category     string     `json:"category"`
	Description  string     `json:"description"`
	Tags         []string   `json:"tags"`
	Tiers        TierData   `json:"tiers"`
	AllItems     []TierItem `json:"allItems"`
	IsPublic     *bool      `json:"isPublic"`
	SourcePostID string     `json:"sourcePostId"`
}

type CreateCommentRequest struct {
	Text string `json:"text"`
}

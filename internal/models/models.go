package models

import (
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Auth    *UserAuth    `gorm:"foreignKey:UserID"`
	Profile *UserProfile `gorm:"foreignKey:UserID"`
	Stats   *UserStats   `gorm:"foreignKey:UserID"`

	Posts []Post `gorm:"foreignKey:CreatorID"`
}

type UserAuth struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	UserID       string `gorm:"type:uuid;uniqueIndex"`
	Provider     string
	Email        *string `gorm:"uniqueIndex"`
	PasswordHash *string
	ProviderSub  *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserProfile struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	UserID      string `gorm:"type:uuid;uniqueIndex"`
	Username    string `gorm:"uniqueIndex"`
	DisplayName *string
	Bio         *string
	AvatarURL   *string
	ThemeColor  *string
	Verified    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UserStats struct {
	ID                string `gorm:"type:uuid;primaryKey"`
	UserID            string `gorm:"type:uuid;uniqueIndex"`
	RanksCreatedCount int
	FollowersCount    int
	FollowingCount    int
	UpdatedAt         time.Time
}

type Subscription struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	UserID     string `gorm:"type:uuid;index"`
	Plan       string
	Status     string
	StartedAt  time.Time
	EndedAt    *time.Time
	Provider   *string
	ProviderID *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Follow struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	FollowerID  string `gorm:"type:uuid;index"`
	FollowingID string `gorm:"type:uuid;index"`
	CreatedAt   time.Time
}

type Category struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Slug        string         `gorm:"uniqueIndex" json:"slug"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Icon        *string        `json:"icon"`
	Emoji       *string        `json:"emoji"`
	Color       *string        `json:"color"`
	Tags        pq.StringArray `gorm:"type:text[]" json:"tags"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type TierListTemplate struct {
	ID          string  `gorm:"type:uuid;primaryKey"`
	CategoryID  string  `gorm:"type:uuid;index"`
	CreatorID   *string `gorm:"type:uuid;index"`
	IsMaster    bool
	Title       string
	Description *string
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TierDefinition struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	TemplateID string `gorm:"type:uuid;index"`
	Key        string `gorm:"index"`
	Label      string
	Order      int
	ColorHex   *string
}

type Asset struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	URL       string
	MimeType  *string
	Width     *int
	Height    *int
	SizeBytes *int
	Sha256    *string
	CreatedAt time.Time
}

type Post struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	Type       string    `json:"type"`
	Visibility string    `json:"visibility"`
	CreatorID  string    `gorm:"type:uuid;index"`
	CategoryID string    `gorm:"type:uuid;index"`
	Caption    *string   `json:"caption"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	Creator  User         `gorm:"foreignKey:CreatorID" json:"-"`
	Category Category     `gorm:"foreignKey:CategoryID" json:"-"`
	Metrics  *PostMetrics `gorm:"foreignKey:PostID" json:"-"`
	Rank     *RankPost    `gorm:"foreignKey:PostID" json:"-"`
	Survey   *SurveyPost  `gorm:"foreignKey:PostID" json:"-"`
}

type RankPost struct {
	PostID       string `gorm:"type:uuid;primaryKey"`
	TemplateID   string `gorm:"type:uuid;index"`
	TierKey      string
	ImageAssetID string `gorm:"type:uuid;index"`
	SubjectTitle *string
	SubjectURL   *string

	Image Asset `gorm:"foreignKey:ImageAssetID"`
}

type TierListPost struct {
	PostID           string `gorm:"type:uuid;primaryKey"`
	Title            string
	Description      *string
	CoverAssetID     *string        `gorm:"type:uuid;index"`
	Tags             pq.StringArray `gorm:"type:text[]"`
	ParticipantCount int
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Post       Post           `gorm:"foreignKey:PostID"`
	CoverAsset *Asset         `gorm:"foreignKey:CoverAssetID"`
	Items      []TierListItem `gorm:"foreignKey:TierListPostID;references:PostID"`
}

type TierListItem struct {
	ID             string `gorm:"type:uuid;primaryKey"`
	TierListPostID string `gorm:"type:uuid;index"`
	ExternalID     string
	Name           string
	Emoji          *string
	TierKey        string
	TierPosition   int
	ListPosition   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SurveyPost struct {
	PostID       string `gorm:"type:uuid;primaryKey"`
	SurveyType   string
	SponsorOrgID *string `gorm:"type:uuid;index"`
	Title        string
	Description  *string
	EndsAt       *time.Time

	Post       Post             `gorm:"foreignKey:PostID"`
	SponsorOrg *Organization    `gorm:"foreignKey:SponsorOrgID"`
	Campaign   *SurveyCampaign  `gorm:"foreignKey:SurveyPostID"`
	Questions  []SurveyQuestion `gorm:"foreignKey:SurveyPostID"`
}

type SurveyQuestion struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	SurveyPostID string `gorm:"type:uuid;index"`
	Order        int
	Type         string
	Prompt       string
	Required     bool

	Options []SurveyOption `gorm:"foreignKey:QuestionID"`
}

type SurveyOption struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	QuestionID string `gorm:"type:uuid;index"`
	Order      int
	Label      string
	Value      *string
}

type SurveyCampaign struct {
	ID                string `gorm:"type:uuid;primaryKey"`
	SurveyPostID      string `gorm:"type:uuid;uniqueIndex"`
	SponsorOrgID      string `gorm:"type:uuid;index"`
	StartAt           time.Time
	EndAt             *time.Time
	BudgetCents       int
	SpentCents        int
	TargetImpressions *int
	Targeting         *string `gorm:"type:jsonb"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	SurveyPost SurveyPost `gorm:"foreignKey:SurveyPostID"`
}

type SurveyImpression struct {
	ID            string  `gorm:"type:uuid;primaryKey"`
	CampaignID    string  `gorm:"type:uuid;index"`
	UserID        *string `gorm:"type:uuid;index"`
	FeedRequestID *string
	CreatedAt     time.Time
}

type PostMetrics struct {
	PostID       string `gorm:"type:uuid;primaryKey"`
	LikeCount    int
	CommentCount int
	ShareCount   int
	HotScore     float64
	UpdatedAt    time.Time
}

type Comment struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	PostID    string `gorm:"type:uuid;index"`
	AuthorID  string `gorm:"type:uuid;index"`
	Body      string
	LikeCount int
	CreatedAt time.Time
	UpdatedAt time.Time

	Author User `gorm:"foreignKey:AuthorID"`
}

type PostLike struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	PostID    string `gorm:"type:uuid;index"`
	UserID    string `gorm:"type:uuid;index"`
	CreatedAt time.Time
}

type PostShare struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	PostID    string `gorm:"type:uuid;index"`
	UserID    string `gorm:"type:uuid;index"`
	Channel   string
	Ref       *string
	CreatedAt time.Time
}

type PinnedPost struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	UserID    string `gorm:"type:uuid;index"`
	PostID    string `gorm:"type:uuid;index"`
	Order     *int
	CreatedAt time.Time
}

type MessageThread struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	OwnerUserID string `gorm:"type:uuid;index"`
	PeerUserID  string `gorm:"type:uuid;index"`
	LastMessage string
	UnreadCount int
	UpdatedAt   time.Time
	CreatedAt   time.Time

	OwnerUser User            `gorm:"foreignKey:OwnerUserID"`
	PeerUser  User            `gorm:"foreignKey:PeerUserID"`
	Messages  []DirectMessage `gorm:"foreignKey:ThreadID"`
}

type DirectMessage struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	ThreadID     string `gorm:"type:uuid;index"`
	SenderUserID string `gorm:"type:uuid;index"`
	Body         string
	CreatedAt    time.Time

	Thread     MessageThread `gorm:"foreignKey:ThreadID"`
	SenderUser User          `gorm:"foreignKey:SenderUserID"`
}

type Notification struct {
	ID          string  `gorm:"type:uuid;primaryKey"`
	UserID      string  `gorm:"type:uuid;index"`
	ActorUserID *string `gorm:"type:uuid;index"`
	Type        string  `gorm:"index"`
	Title       string
	Body        string
	ActionHref  string
	ReadAt      *time.Time
	CreatedAt   time.Time

	User      User  `gorm:"foreignKey:UserID"`
	ActorUser *User `gorm:"foreignKey:ActorUserID"`
}

type TrendingTopic struct {
	ID               string `gorm:"type:uuid;primaryKey"`
	Title            string
	CategoryID       string  `gorm:"type:uuid;index"`
	CoverAssetID     *string `gorm:"type:uuid;index"`
	SourcePostID     *string `gorm:"type:uuid;index"`
	ParticipantCount int
	Tags             pq.StringArray `gorm:"type:text[]"`
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Category   Category `gorm:"foreignKey:CategoryID"`
	CoverAsset *Asset   `gorm:"foreignKey:CoverAssetID"`
	SourcePost *Post    `gorm:"foreignKey:SourcePostID"`
}

type LeaderboardEntry struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	UserID    string `gorm:"type:uuid;index"`
	Rank      int
	Score     int
	Change    string
	CreatedAt time.Time
	UpdatedAt time.Time

	User User `gorm:"foreignKey:UserID"`
}

type Organization struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	Name      string
	Website   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

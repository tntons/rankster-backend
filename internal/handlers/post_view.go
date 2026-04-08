package handlers

import (
	"time"

	"rankster-backend/internal/models"
)

type PostView struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	Visibility string       `json:"visibility"`
	Caption    *string      `json:"caption"`
	CreatedAt  string       `json:"createdAt"`
	Creator    CreatorView  `json:"creator"`
	Category   CategoryView `json:"category"`
	Metrics    MetricsView  `json:"metrics"`
	Rank       *RankView    `json:"rank"`
	Survey     *SurveyView  `json:"survey"`
}

type CreatorView struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
}

type CategoryView struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Icon        *string  `json:"icon"`
	Tags        []string `json:"tags"`
}

type MetricsView struct {
	LikeCount    int     `json:"likeCount"`
	CommentCount int     `json:"commentCount"`
	ShareCount   int     `json:"shareCount"`
	HotScore     float64 `json:"hotScore"`
}

type RankView struct {
	TemplateID   string    `json:"templateId"`
	TierKey      string    `json:"tierKey"`
	Image        AssetView `json:"image"`
	SubjectTitle *string   `json:"subjectTitle"`
	SubjectURL   *string   `json:"subjectUrl"`
}

type AssetView struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	MimeType *string `json:"mimeType"`
	Width    *int    `json:"width"`
	Height   *int    `json:"height"`
}

type SurveyView struct {
	SurveyType  string               `json:"surveyType"`
	Sponsor     *SponsorView         `json:"sponsor"`
	Title       string               `json:"title"`
	Description *string              `json:"description"`
	EndsAt      *string              `json:"endsAt"`
	Questions   []SurveyQuestionView `json:"questions"`
}

type SponsorView struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
}

type SurveyQuestionView struct {
	ID       string             `json:"id"`
	Order    int                `json:"order"`
	Type     string             `json:"type"`
	Prompt   string             `json:"prompt"`
	Required bool               `json:"required"`
	Options  []SurveyOptionView `json:"options"`
}

type SurveyOptionView struct {
	ID    string  `json:"id"`
	Order int     `json:"order"`
	Label string  `json:"label"`
	Value *string `json:"value"`
}

func toPostView(post models.Post) PostView {
	creator := CreatorView{
		ID:          post.Creator.ID,
		Username:    "unknown",
		DisplayName: nil,
		AvatarURL:   nil,
	}
	if post.Creator.Profile != nil {
		creator.Username = post.Creator.Profile.Username
		creator.DisplayName = post.Creator.Profile.DisplayName
		creator.AvatarURL = post.Creator.Profile.AvatarURL
	}

	metrics := MetricsView{LikeCount: 0, CommentCount: 0, ShareCount: 0, HotScore: 0}
	if post.Metrics != nil {
		metrics = MetricsView{
			LikeCount:    post.Metrics.LikeCount,
			CommentCount: post.Metrics.CommentCount,
			ShareCount:   post.Metrics.ShareCount,
			HotScore:     post.Metrics.HotScore,
		}
	}

	var rank *RankView
	if post.Rank != nil {
		rank = &RankView{
			TemplateID:   post.Rank.TemplateID,
			TierKey:      post.Rank.TierKey,
			Image:        toAssetView(post.Rank.Image),
			SubjectTitle: post.Rank.SubjectTitle,
			SubjectURL:   post.Rank.SubjectURL,
		}
	}

	var survey *SurveyView
	if post.Survey != nil {
		survey = &SurveyView{
			SurveyType:  post.Survey.SurveyType,
			Title:       post.Survey.Title,
			Description: post.Survey.Description,
		}
		if post.Survey.EndsAt != nil {
			formatted := post.Survey.EndsAt.UTC().Format(time.RFC3339Nano)
			survey.EndsAt = &formatted
		}
		if post.Survey.SponsorOrg != nil {
			survey.Sponsor = &SponsorView{
				OrganizationID: post.Survey.SponsorOrg.ID,
				Name:           post.Survey.SponsorOrg.Name,
			}
		}
		for _, q := range post.Survey.Questions {
			qv := SurveyQuestionView{
				ID:       q.ID,
				Order:    q.Order,
				Type:     q.Type,
				Prompt:   q.Prompt,
				Required: q.Required,
				Options:  []SurveyOptionView{},
			}
			for _, o := range q.Options {
				qv.Options = append(qv.Options, SurveyOptionView{
					ID:    o.ID,
					Order: o.Order,
					Label: o.Label,
					Value: o.Value,
				})
			}
			survey.Questions = append(survey.Questions, qv)
		}
	}

	return PostView{
		ID:         post.ID,
		Type:       post.Type,
		Visibility: post.Visibility,
		Caption:    post.Caption,
		CreatedAt:  post.CreatedAt.UTC().Format(time.RFC3339Nano),
		Creator:    creator,
		Category: CategoryView{
			ID:          post.Category.ID,
			Slug:        post.Category.Slug,
			Name:        post.Category.Name,
			Description: post.Category.Description,
			Icon:        post.Category.Icon,
			Tags:        []string(post.Category.Tags),
		},
		Metrics: metrics,
		Rank:    rank,
		Survey:  survey,
	}
}

func toAssetView(asset models.Asset) AssetView {
	return AssetView{
		ID:       asset.ID,
		URL:      asset.URL,
		MimeType: asset.MimeType,
		Width:    asset.Width,
		Height:   asset.Height,
	}
}

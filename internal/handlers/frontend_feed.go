package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"rankster-backend/internal/auth"
	"rankster-backend/internal/models"
)

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
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Emoji *string `json:"emoji,omitempty"`
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
}

type frontendFeedResponse struct {
	Items      []frontendRankPostView `json:"items"`
	NextCursor any                    `json:"nextCursor"`
}

func buildFrontendFeedResponse(db *gorm.DB, posts []models.Post, authCtx auth.Context, nextCursor string) (frontendFeedResponse, error) {
	rankPosts := make([]models.Post, 0, len(posts))
	userIDs := map[string]struct{}{}
	postIDs := make([]string, 0, len(posts))

	for _, post := range posts {
		if post.Type != "RANK" || post.Rank == nil {
			continue
		}

		rankPosts = append(rankPosts, post)
		userIDs[post.CreatorID] = struct{}{}
		postIDs = append(postIDs, post.ID)
	}

	userMap, err := loadFrontendUsers(db, userIDs)
	if err != nil {
		return frontendFeedResponse{}, err
	}

	commentMap, err := loadFrontendComments(db, postIDs, userMap)
	if err != nil {
		return frontendFeedResponse{}, err
	}

	likedMap, err := loadUserLikes(db, postIDs, authCtx)
	if err != nil {
		return frontendFeedResponse{}, err
	}

	items := make([]frontendRankPostView, 0, len(rankPosts))
	for _, post := range rankPosts {
		items = append(items, toFrontendRankPostView(post, userMap, commentMap, likedMap))
	}

	return frontendFeedResponse{
		Items:      items,
		NextCursor: emptyToNull(nextCursor),
	}, nil
}

func loadFrontendUsers(db *gorm.DB, userIDs map[string]struct{}) (map[string]frontendUserView, error) {
	ids := make([]string, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return map[string]frontendUserView{}, nil
	}

	var users []models.User
	if err := db.
		Preload("Profile").
		Preload("Stats").
		Where("id IN ?", ids).
		Find(&users).Error; err != nil {
		return nil, err
	}

	var subscriptions []models.Subscription
	if err := db.
		Where("user_id IN ? AND status = ? AND plan IN ?", ids, "ACTIVE", []string{"PRO", "BUSINESS"}).
		Find(&subscriptions).Error; err != nil {
		return nil, err
	}

	verifiedMap := map[string]bool{}
	for _, subscription := range subscriptions {
		verifiedMap[subscription.UserID] = true
	}

	result := make(map[string]frontendUserView, len(users))
	for _, user := range users {
		result[user.ID] = toFrontendUserView(user, verifiedMap[user.ID])
	}

	return result, nil
}

func loadFrontendComments(db *gorm.DB, postIDs []string, userMap map[string]frontendUserView) (map[string][]frontendCommentView, error) {
	if len(postIDs) == 0 {
		return map[string][]frontendCommentView{}, nil
	}

	var comments []models.Comment
	if err := db.
		Where("post_id IN ?", postIDs).
		Order("created_at desc").
		Find(&comments).Error; err != nil {
		return nil, err
	}

	commenterIDs := map[string]struct{}{}
	for _, comment := range comments {
		if _, exists := userMap[comment.AuthorID]; !exists {
			commenterIDs[comment.AuthorID] = struct{}{}
		}
	}

	if len(commenterIDs) > 0 {
		commentUsers, err := loadFrontendUsers(db, commenterIDs)
		if err != nil {
			return nil, err
		}
		for id, user := range commentUsers {
			userMap[id] = user
		}
	}

	result := make(map[string][]frontendCommentView)
	for _, comment := range comments {
		user, ok := userMap[comment.AuthorID]
		if !ok {
			user = fallbackFrontendUser(comment.AuthorID)
		}

		result[comment.PostID] = append(result[comment.PostID], frontendCommentView{
			ID:        comment.ID,
			User:      user,
			Text:      comment.Body,
			CreatedAt: relativeTime(comment.CreatedAt),
			Likes:     0,
		})
	}

	return result, nil
}

func loadUserLikes(db *gorm.DB, postIDs []string, authCtx auth.Context) (map[string]bool, error) {
	result := map[string]bool{}
	if len(postIDs) == 0 || authCtx.Kind != "user" {
		return result, nil
	}

	var likes []models.PostLike
	if err := db.
		Where("user_id = ? AND post_id IN ?", authCtx.UserID, postIDs).
		Find(&likes).Error; err != nil {
		return nil, err
	}

	for _, like := range likes {
		result[like.PostID] = true
	}

	return result, nil
}

func toFrontendRankPostView(post models.Post, userMap map[string]frontendUserView, commentMap map[string][]frontendCommentView, likedMap map[string]bool) frontendRankPostView {
	user, ok := userMap[post.CreatorID]
	if !ok {
		user = fallbackFrontendUser(post.CreatorID)
	}

	tierData, allItems := frontendTierDataFromPost(post)
	description := ""
	if post.Caption != nil {
		description = *post.Caption
	}

	title := post.Category.Name
	if post.Rank != nil && post.Rank.SubjectTitle != nil && *post.Rank.SubjectTitle != "" {
		title = *post.Rank.SubjectTitle
	}
	if title == post.Category.Name && description != "" {
		title = description
	}

	comments := commentMap[post.ID]
	if comments == nil {
		comments = []frontendCommentView{}
	}

	likes := 0
	shares := 0
	commentCount := len(comments)
	if post.Metrics != nil {
		likes = post.Metrics.LikeCount
		shares = post.Metrics.ShareCount
		if post.Metrics.CommentCount > commentCount {
			commentCount = post.Metrics.CommentCount
		}
	}

	participantCount := likes + shares + commentCount
	if participantCount == 0 {
		participantCount = len(allItems)
	}

	return frontendRankPostView{
		ID:               post.ID,
		User:             user,
		Title:            title,
		Category:         post.Category.Slug,
		CoverImage:       rankCoverImage(post),
		Tiers:            tierData,
		AllItems:         allItems,
		Description:      description,
		Tags:             append([]string{}, []string(post.Category.Tags)...),
		Likes:            likes,
		IsLiked:          likedMap[post.ID],
		Comments:         comments,
		Shares:           shares,
		CreatedAt:        relativeTime(post.CreatedAt),
		IsPublic:         post.Visibility == "PUBLIC",
		ParticipantCount: participantCount,
	}
}

func toFrontendUserView(user models.User, verified bool) frontendUserView {
	profile := user.Profile
	stats := user.Stats

	displayName := "Unknown User"
	username := "unknown"
	avatar := ""
	bio := "Ranking everything, one take at a time."
	followers := 0
	following := 0
	totalRankings := 0

	if profile != nil {
		username = profile.Username
		if profile.DisplayName != nil && *profile.DisplayName != "" {
			displayName = *profile.DisplayName
		} else {
			displayName = profile.Username
		}
		if profile.AvatarURL != nil {
			avatar = *profile.AvatarURL
		}
		if profile.Bio != nil && *profile.Bio != "" {
			bio = *profile.Bio
		}
	}

	if stats != nil {
		followers = stats.FollowersCount
		following = stats.FollowingCount
		totalRankings = stats.RanksCreatedCount
	}

	return frontendUserView{
		ID:            user.ID,
		Username:      username,
		DisplayName:   displayName,
		Avatar:        avatar,
		Bio:           bio,
		Followers:     followers,
		Following:     following,
		TotalRankings: totalRankings,
		Verified:      verified,
	}
}

func fallbackFrontendUser(userID string) frontendUserView {
	return frontendUserView{
		ID:            userID,
		Username:      "unknown",
		DisplayName:   "Unknown User",
		Avatar:        "",
		Bio:           "Ranking everything, one take at a time.",
		Followers:     0,
		Following:     0,
		TotalRankings: 0,
		Verified:      false,
	}
}

func frontendTierDataFromPost(post models.Post) (frontendTierData, []frontendTierItem) {
	result := frontendTierData{
		S: []frontendTierItem{},
		A: []frontendTierItem{},
		B: []frontendTierItem{},
		C: []frontendTierItem{},
		D: []frontendTierItem{},
	}

	items := []frontendTierItem{}
	if post.Rank == nil {
		return result, items
	}

	itemName := post.Category.Name
	if post.Rank.SubjectTitle != nil && *post.Rank.SubjectTitle != "" {
		itemName = *post.Rank.SubjectTitle
	}

	item := frontendTierItem{
		ID:   post.Rank.Image.ID,
		Name: itemName,
	}
	items = append(items, item)

	switch strings.ToUpper(post.Rank.TierKey) {
	case "S":
		result.S = append(result.S, item)
	case "A":
		result.A = append(result.A, item)
	case "B":
		result.B = append(result.B, item)
	case "C":
		result.C = append(result.C, item)
	default:
		result.D = append(result.D, item)
	}

	return result, items
}

func rankCoverImage(post models.Post) string {
	if post.Rank != nil && post.Rank.Image.URL != "" {
		return post.Rank.Image.URL
	}
	return ""
}

func relativeTime(timestamp time.Time) string {
	diff := time.Since(timestamp)
	if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes < 1 {
			minutes = 1
		}
		return fmt.Sprintf("%dm ago", minutes)
	}

	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}

	days := int(diff.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%dd ago", days)
	}

	return timestamp.Format("Jan 2, 2006")
}

func sortFrontendComments(comments []frontendCommentView) {
	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].CreatedAt < comments[j].CreatedAt
	})
}

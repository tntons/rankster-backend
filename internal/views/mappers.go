package views

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"rankster-backend/internal/models"
)

func BuildUser(user models.User) User {
	profile := models.UserProfile{}
	if user.Profile != nil {
		profile = *user.Profile
	}
	stats := models.UserStats{}
	if user.Stats != nil {
		stats = *user.Stats
	}

	displayName := profile.Username
	if profile.DisplayName != nil && strings.TrimSpace(*profile.DisplayName) != "" {
		displayName = *profile.DisplayName
	}

	avatar := AssetURL("avatars", profile.Username)
	if profile.AvatarURL != nil && strings.TrimSpace(*profile.AvatarURL) != "" {
		avatar = AvatarURLOrFallback(profile.Username, *profile.AvatarURL)
	}

	return User{
		ID:            user.ID,
		Username:      profile.Username,
		DisplayName:   displayName,
		Avatar:        avatar,
		Bio:           derefString(profile.Bio),
		Followers:     stats.FollowersCount,
		Following:     stats.FollowingCount,
		TotalRankings: stats.RanksCreatedCount,
		Verified:      profile.Verified,
	}
}

func BuildNotification(notification models.Notification) Notification {
	var actor *User
	if notification.ActorUser != nil {
		view := BuildUser(*notification.ActorUser)
		if view.ID != "" {
			actor = &view
		}
	}

	return Notification{
		ID:        notification.ID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
		Actor:     actor,
		Href:      notification.ActionHref,
		CreatedAt: RelativeTime(notification.CreatedAt),
		Read:      notification.ReadAt != nil,
	}
}

func BuildRankPost(list models.TierListPost, comments []Comment, isLiked bool, canEdit bool) RankPost {
	if comments == nil {
		comments = []Comment{}
	}
	return RankPost{
		ID:               list.PostID,
		User:             BuildUser(list.Post.Creator),
		Title:            list.Title,
		Category:         list.Post.Category.Slug,
		CoverImage:       AssetOrFallback(list.CoverAsset, "ranks", slugify(list.Title)),
		Tiers:            BuildTierData(list.Items),
		TierRows:         BuildTierRows(list.TierConfig, list.Items),
		AllItems:         BuildAllItems(list.Items),
		Description:      derefString(list.Description),
		Tags:             append([]string{}, list.Tags...),
		Likes:            MetricLikeCount(list.Post.Metrics),
		IsLiked:          isLiked,
		Comments:         comments,
		Shares:           MetricShareCount(list.Post.Metrics),
		CreatedAt:        RelativeTime(list.CreatedAt),
		IsPublic:         list.Post.Visibility == "PUBLIC",
		ParticipantCount: list.ParticipantCount,
		CanEdit:          canEdit,
	}
}

func BuildTierRows(config string, items []models.TierListItem) []TierRow {
	rows := tierRowMetadata(config)
	itemsByTier := map[string][]TierItem{}
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		if a.TierKey == b.TierKey {
			return a.TierPosition - b.TierPosition
		}
		return a.ListPosition - b.ListPosition
	})

	for _, item := range sorted {
		view := TierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji, ImageURL: PublicURLPtr(item.ImageURL)}
		itemsByTier[item.TierKey] = append(itemsByTier[item.TierKey], view)
	}

	out := make([]TierRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, TierRow{
			ID:    row.ID,
			Label: row.Label,
			Items: append([]TierItem{}, itemsByTier[row.ID]...),
		})
	}
	return out
}

func tierRowMetadata(config string) []TierRow {
	var rows []TierRow
	if strings.TrimSpace(config) != "" && json.Unmarshal([]byte(config), &rows) == nil {
		return normalizeTierRows(rows)
	}
	return normalizeTierRows([]TierRow{
		{ID: "S", Label: "S"},
		{ID: "A", Label: "A"},
		{ID: "B", Label: "B"},
		{ID: "C", Label: "C"},
		{ID: "D", Label: "D"},
	})
}

func normalizeTierRows(rows []TierRow) []TierRow {
	out := make([]TierRow, 0, len(rows))
	for index, row := range rows {
		id := strings.TrimSpace(row.ID)
		if id == "" {
			id = fmt.Sprintf("tier-%d", index+1)
		}
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = id
		}
		if strings.TrimSpace(label) == "" {
			label = "Tier"
		}
		out = append(out, TierRow{ID: id, Label: label, Items: append([]TierItem{}, row.Items...)})
	}
	return out
}

func BuildTierData(items []models.TierListItem) TierData {
	data := TierData{S: []TierItem{}, A: []TierItem{}, B: []TierItem{}, C: []TierItem{}, D: []TierItem{}}
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		if a.TierKey == b.TierKey {
			return a.TierPosition - b.TierPosition
		}
		return a.ListPosition - b.ListPosition
	})

	for _, item := range sorted {
		view := TierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji, ImageURL: PublicURLPtr(item.ImageURL)}
		switch item.TierKey {
		case "S":
			data.S = append(data.S, view)
		case "A":
			data.A = append(data.A, view)
		case "B":
			data.B = append(data.B, view)
		case "C":
			data.C = append(data.C, view)
		case "D":
			data.D = append(data.D, view)
		}
	}
	return data
}

func BuildAllItems(items []models.TierListItem) []TierItem {
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		return a.ListPosition - b.ListPosition
	})

	out := make([]TierItem, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, TierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji, ImageURL: PublicURLPtr(item.ImageURL)})
	}
	return out
}

func BuildMessageThread(thread models.MessageThread) MessageThread {
	lastMessage := thread.LastMessage
	if strings.TrimSpace(lastMessage) == "" {
		lastMessage = "Say hi to start the conversation"
	}

	return MessageThread{
		ID:          thread.ID,
		User:        BuildUser(thread.PeerUser),
		LastMessage: lastMessage,
		Timestamp:   RelativeTime(thread.UpdatedAt),
		Unread:      thread.UnreadCount,
	}
}

func BuildChatMessage(message models.DirectMessage, ownerUserID string) ChatMessage {
	return ChatMessage{
		ID:        message.ID,
		Text:      message.Body,
		Mine:      message.SenderUserID == ownerUserID,
		Timestamp: ChatTimestamp(message.CreatedAt),
	}
}

func BuildMessageThreadDetail(thread models.MessageThread, ownerUserID string) MessageThreadDetail {
	messages := make([]ChatMessage, 0, len(thread.Messages))
	for _, message := range thread.Messages {
		messages = append(messages, BuildChatMessage(message, ownerUserID))
	}

	return MessageThreadDetail{
		ID:       thread.ID,
		User:     BuildUser(thread.PeerUser),
		Messages: messages,
	}
}

func BuildTrendingTopic(topic models.TrendingTopic) TrendingTopic {
	return TrendingTopic{
		ID:               topic.ID,
		PostID:           topic.SourcePostID,
		Title:            topic.Title,
		Category:         topic.Category.Slug,
		CoverImage:       AssetOrFallback(topic.CoverAsset, "ranks", slugify(topic.Title)),
		ParticipantCount: topic.ParticipantCount,
		Tags:             append([]string{}, topic.Tags...),
	}
}

func BuildCategory(category models.Category) Category {
	emoji := ""
	if category.Emoji != nil {
		emoji = *category.Emoji
	}
	color := ""
	if category.Color != nil {
		color = *category.Color
	}
	return Category{
		ID:    category.Slug,
		Name:  category.Name,
		Emoji: emoji,
		Color: color,
	}
}

func NewChatMessage(id string, text string, mine bool) ChatMessage {
	return ChatMessage{
		ID:        id,
		Text:      text,
		Mine:      mine,
		Timestamp: "Now",
	}
}

func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	diff := time.Since(t)
	if diff < time.Minute {
		return "Just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
}

func ChatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("3:04 PM")
}

func AssetOrFallback(asset *models.Asset, kind, slug string) string {
	if asset != nil && strings.TrimSpace(asset.URL) != "" {
		return PublicURL(asset.URL)
	}
	return AssetURL(kind, slug)
}

func AssetURL(kind string, slug string) string {
	return fmt.Sprintf("%s/assets/%s/%s.svg", publicBaseURL(), kind, safeSlug(slug))
}

func PublicURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return value
	}
	if parsed.Host != "localhost:8000" && parsed.Host != "127.0.0.1:8000" {
		return value
	}
	if !strings.HasPrefix(parsed.Path, "/assets/") && !strings.HasPrefix(parsed.Path, "/uploads/") {
		return value
	}
	return publicBaseURL() + parsed.EscapedPath()
}

func PublicURLPtr(value *string) *string {
	if value == nil {
		return nil
	}
	nextValue := PublicURL(*value)
	return &nextValue
}

func AvatarURLOrFallback(username string, value string) string {
	avatar := PublicURL(value)
	if isMissingLocalUploadURL(avatar) {
		return AssetURL("avatars", username)
	}
	return avatar
}

func isMissingLocalUploadURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || !strings.HasPrefix(parsed.Path, "/uploads/") {
		return false
	}

	publicBase, err := url.Parse(publicBaseURL())
	if err != nil {
		publicBase = nil
	}
	isLocalUploadHost := parsed.Host == "localhost:8000" || parsed.Host == "127.0.0.1:8000"
	if publicBase != nil && parsed.Host == publicBase.Host {
		isLocalUploadHost = true
	}
	if !isLocalUploadHost {
		return false
	}

	if _, err := os.Stat(strings.TrimPrefix(parsed.Path, "/")); err != nil {
		return true
	}
	return false
}

func publicBaseURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if baseURL == "" {
		return "http://localhost:8000"
	}
	return baseURL
}

func MetricLikeCount(metrics *models.PostMetrics) int {
	if metrics == nil {
		return 0
	}
	return metrics.LikeCount
}

func MetricShareCount(metrics *models.PostMetrics) int {
	if metrics == nil {
		return 0
	}
	return metrics.ShareCount
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "'", "")
	value = strings.ReplaceAll(value, "&", "and")

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "rankster"
	}
	return result
}

func safeSlug(raw string) string {
	value := strings.Trim(strings.ToLower(raw), "/ ")
	if value == "" {
		return "rankster"
	}

	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "rankster"
	}
	return builder.String()
}

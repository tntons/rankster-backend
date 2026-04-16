package handlers

import (
	"slices"
	"strings"

	"rankster-backend/internal/models"
)

func (h *FrontendHandler) hydrateTierLists(lists []models.TierListPost, authUser *frontendUserView) ([]frontendRankPostView, error) {
	postIDs := make([]string, 0, len(lists))
	for _, list := range lists {
		postIDs = append(postIDs, list.PostID)
	}

	commentsByPost, err := h.loadComments(postIDs, authUser)
	if err != nil {
		return nil, err
	}
	likedByPost, err := h.loadLikedPosts(postIDs, authUser)
	if err != nil {
		return nil, err
	}

	items := make([]frontendRankPostView, 0, len(lists))
	for _, list := range lists {
		canEdit := authUser != nil && list.Post.CreatorID == authUser.ID
		items = append(items, hydrateTierList(list, commentsByPost[list.PostID], likedByPost[list.PostID], canEdit))
	}
	return items, nil
}

func (h *FrontendHandler) loadComments(postIDs []string, authUser *frontendUserView) (map[string][]frontendCommentView, error) {
	out := map[string][]frontendCommentView{}
	if len(postIDs) == 0 {
		return out, nil
	}

	var comments []models.Comment
	err := h.db.
		Preload("Author.Profile").
		Preload("Author.Stats").
		Where("post_id IN ?", postIDs).
		Order("created_at desc").
		Find(&comments).Error
	if err != nil {
		return nil, err
	}

	likedByComment := map[string]bool{}
	if authUser != nil && len(comments) > 0 {
		commentIDs := make([]string, 0, len(comments))
		for _, comment := range comments {
			commentIDs = append(commentIDs, comment.ID)
		}

		var likes []models.CommentLike
		if err := h.db.Where("user_id = ? AND comment_id IN ?", authUser.ID, commentIDs).Find(&likes).Error; err != nil {
			return nil, err
		}
		for _, like := range likes {
			likedByComment[like.CommentID] = true
		}
	}

	for _, comment := range comments {
		out[comment.PostID] = append(out[comment.PostID], frontendCommentView{
			ID:        comment.ID,
			User:      buildFrontendUser(comment.Author),
			Text:      comment.Body,
			CreatedAt: relativeTime(comment.CreatedAt),
			Likes:     comment.LikeCount,
			IsLiked:   likedByComment[comment.ID],
		})
	}
	return out, nil
}

func (h *FrontendHandler) loadLikedPosts(postIDs []string, authUser *frontendUserView) (map[string]bool, error) {
	out := map[string]bool{}
	if authUser == nil || len(postIDs) == 0 {
		return out, nil
	}

	var likes []models.PostLike
	if err := h.db.Where("user_id = ? AND post_id IN ?", authUser.ID, postIDs).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, like := range likes {
		out[like.PostID] = true
	}
	return out, nil
}

func hydrateTierList(list models.TierListPost, comments []frontendCommentView, isLiked bool, canEdit bool) frontendRankPostView {
	if comments == nil {
		comments = []frontendCommentView{}
	}
	return frontendRankPostView{
		ID:               list.PostID,
		User:             buildFrontendUser(list.Post.Creator),
		Title:            list.Title,
		Category:         list.Post.Category.Slug,
		CoverImage:       assetOrFallback(list.CoverAsset, "ranks", slugify(list.Title)),
		Tiers:            buildTierData(list.Items),
		AllItems:         buildAllItems(list.Items),
		Description:      derefString(list.Description),
		Tags:             append([]string{}, list.Tags...),
		Likes:            metricLikeCount(list.Post.Metrics),
		IsLiked:          isLiked,
		Comments:         comments,
		Shares:           metricShareCount(list.Post.Metrics),
		CreatedAt:        relativeTime(list.CreatedAt),
		IsPublic:         list.Post.Visibility == "PUBLIC",
		ParticipantCount: list.ParticipantCount,
		CanEdit:          canEdit,
	}
}

func buildFrontendUser(user models.User) frontendUserView {
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

	avatar := assetURL("avatars", profile.Username)
	if profile.AvatarURL != nil && strings.TrimSpace(*profile.AvatarURL) != "" {
		avatar = *profile.AvatarURL
	}

	return frontendUserView{
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

func buildFrontendNotification(notification models.Notification) frontendNotificationView {
	var actor *frontendUserView
	if notification.ActorUser != nil {
		view := buildFrontendUser(*notification.ActorUser)
		if view.ID != "" {
			actor = &view
		}
	}

	return frontendNotificationView{
		ID:        notification.ID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
		Actor:     actor,
		Href:      notification.ActionHref,
		CreatedAt: relativeTime(notification.CreatedAt),
		Read:      notification.ReadAt != nil,
	}
}

func buildTierData(items []models.TierListItem) frontendTierData {
	data := frontendTierData{S: []frontendTierItem{}, A: []frontendTierItem{}, B: []frontendTierItem{}, C: []frontendTierItem{}, D: []frontendTierItem{}}
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		if a.TierKey == b.TierKey {
			return a.TierPosition - b.TierPosition
		}
		return a.ListPosition - b.ListPosition
	})

	for _, item := range sorted {
		view := frontendTierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji, ImageURL: item.ImageURL}
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

func buildAllItems(items []models.TierListItem) []frontendTierItem {
	sorted := append([]models.TierListItem{}, items...)
	slices.SortFunc(sorted, func(a, b models.TierListItem) int {
		return a.ListPosition - b.ListPosition
	})

	out := make([]frontendTierItem, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, frontendTierItem{ID: item.ExternalID, Name: item.Name, Emoji: item.Emoji, ImageURL: item.ImageURL})
	}
	return out
}

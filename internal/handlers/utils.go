package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"encoding/base64"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"rankster-backend/internal/models"
)

func ensureCategory(tx *gorm.DB, slug string, now time.Time) (models.Category, error) {
	slug = slugify(slug)
	var category models.Category
	err := tx.Where("slug = ?", slug).First(&category).Error
	if err == nil {
		return category, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.Category{}, err
	}

	name := titleizeSlug(slug)
	category = models.Category{
		ID:        generateUUID(),
		Slug:      slug,
		Name:      name,
		Tags:      pq.StringArray{slug},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return category, tx.Create(&category).Error
}

func titleizeSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func decodeCursor(raw string) int {
	if raw == "" {
		return 0
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0
	}
	return value
}

func relativeTime(t time.Time) string {
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

func chatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("3:04 PM")
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
		return fmt.Sprintf("+%d", delta)
	}
	if delta < 0 {
		return fmt.Sprintf("%d", delta)
	}
	return "0"
}

func assetOrFallback(asset *models.Asset, kind, slug string) string {
	if asset != nil && strings.TrimSpace(asset.URL) != "" {
		return asset.URL
	}
	return assetURL(kind, slug)
}

func assetURL(kind string, slug string) string {
	return fmt.Sprintf("http://localhost:8000/assets/%s/%s.svg", kind, safeSlug(slug))
}

func metricLikeCount(metrics *models.PostMetrics) int {
	if metrics == nil {
		return 0
	}
	return metrics.LikeCount
}

func metricShareCount(metrics *models.PostMetrics) int {
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

func generateUUID() string {
	return uuid.NewString()
}

func stableValue(input string) int {
	total := 0
	for _, char := range input {
		total += int(char)
	}
	return total
}

func intPtrValue(value int) *int {
	return &value
}

func coalesceEmoji(primary, secondary *string) *string {
	if primary != nil {
		return primary
	}
	return secondary
}

func coalesceImageURL(primary, secondary *string) *string {
	if value := optionalStringPtr(derefString(primary)); value != nil {
		return value
	}
	return optionalStringPtr(derefString(secondary))
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

func parseIntWithDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func stringPtr(value string) *string {
	return &value
}

func optionalStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

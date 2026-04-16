package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"rankster-backend/internal/models"
)

var ErrForbidden = errors.New("forbidden")
var ErrUnauthorized = errors.New("unauthorized")

type ComputedUserStats struct {
	RanksCreated     int
	LikesReceived    int
	CommentsReceived int
	Followers        int
	Following        int
}

func ensureCategory(tx *gorm.DB, slug string, now time.Time) (models.Category, error) {
	slug = slugify(slug)
	var category models.Category
	err := tx.Where("slug = ?", slug).First(&category).Error
	if err == nil {
		return category, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
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

func generateUUID() string {
	return uuid.NewString()
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

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

func stableValue(input string) int {
	total := 0
	for _, char := range input {
		total += int(char)
	}
	return total
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intFromString(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func rankCoverURL(title string) string {
	return fmt.Sprintf("http://localhost:8000/assets/ranks/%s.svg", slugify(title))
}

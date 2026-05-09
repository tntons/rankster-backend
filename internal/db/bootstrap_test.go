package db_test

import (
	"testing"

	appdb "rankster-backend/internal/db"
	"rankster-backend/internal/models"
	"rankster-backend/internal/testutil"
)

func TestEnsureDatabaseDoesNotResetExistingDemoProfile(t *testing.T) {
	database := testutil.NewTestDatabase(t)

	customAvatar := "https://example.com/custom-avatar.png"
	customName := "Custom Alex"
	customBio := "This profile should survive backend deploys."

	if err := database.Model(&models.UserProfile{}).
		Where("username = ?", "me").
		Updates(map[string]any{
			"avatar_url":   customAvatar,
			"display_name": customName,
			"bio":          customBio,
		}).Error; err != nil {
		t.Fatalf("customize demo profile: %v", err)
	}

	if err := appdb.EnsureDatabase(database, "http://localhost:8000"); err != nil {
		t.Fatalf("rerun database bootstrap: %v", err)
	}

	var profile models.UserProfile
	if err := database.Where("username = ?", "me").First(&profile).Error; err != nil {
		t.Fatalf("load demo profile: %v", err)
	}

	if profile.AvatarURL == nil || *profile.AvatarURL != customAvatar {
		t.Fatalf("avatar was reset by seed: got %v, want %q", profile.AvatarURL, customAvatar)
	}
	if profile.DisplayName == nil || *profile.DisplayName != customName {
		t.Fatalf("display name was reset by seed: got %v, want %q", profile.DisplayName, customName)
	}
	if profile.Bio == nil || *profile.Bio != customBio {
		t.Fatalf("bio was reset by seed: got %v, want %q", profile.Bio, customBio)
	}
}

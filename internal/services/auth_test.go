package services

import (
	"testing"

	"rankster-backend/internal/models"
	"rankster-backend/internal/repositories"
	"rankster-backend/internal/testutil"
)

func TestGoogleLoginDoesNotOverwriteEditedProfile(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repositories.NewUserRepository(database)
	authService := NewAuthService(database, users, "test-secret", "google-client-id")

	identity := GoogleIdentity{
		Subject:       "google-user-1",
		Email:         "rankster.user@example.com",
		Name:          "Google Name",
		PictureURL:    "https://lh3.googleusercontent.com/original-avatar",
		EmailVerified: true,
	}

	user, err := authService.FindOrCreateGoogleUser(identity)
	if err != nil {
		t.Fatalf("create google user: %v", err)
	}

	customName := "Custom Rankster Name"
	customBio := "Custom bio should not be replaced by auth refresh."
	customAvatar := "https://res.cloudinary.com/demo/image/upload/custom-avatar.png"
	if err := database.Model(&models.UserProfile{}).
		Where("user_id = ?", user.ID).
		Updates(map[string]any{
			"display_name": customName,
			"bio":          customBio,
			"avatar_url":   customAvatar,
		}).Error; err != nil {
		t.Fatalf("customize profile: %v", err)
	}

	identity.Name = "New Google Name"
	identity.PictureURL = "https://lh3.googleusercontent.com/new-google-avatar"
	identity.EmailVerified = false
	if _, err := authService.FindOrCreateGoogleUser(identity); err != nil {
		t.Fatalf("refresh google user: %v", err)
	}

	var profile models.UserProfile
	if err := database.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.DisplayName == nil || *profile.DisplayName != customName {
		t.Fatalf("display name was overwritten: got %v, want %q", profile.DisplayName, customName)
	}
	if profile.Bio == nil || *profile.Bio != customBio {
		t.Fatalf("bio was overwritten: got %v, want %q", profile.Bio, customBio)
	}
	if profile.AvatarURL == nil || *profile.AvatarURL != customAvatar {
		t.Fatalf("avatar was overwritten: got %v, want %q", profile.AvatarURL, customAvatar)
	}
	if profile.Verified {
		t.Fatalf("verified should still refresh from Google identity")
	}
}

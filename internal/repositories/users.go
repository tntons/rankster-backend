package repositories

import (
	"rankster-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(userID string) (models.User, error) {
	return FindUserByID(r.db, userID)
}

func (r *UserRepository) FindByUsername(username string) (models.User, error) {
	var profile models.UserProfile
	if err := r.db.Where("username = ?", username).First(&profile).Error; err != nil {
		return models.User{}, err
	}
	return r.FindByID(profile.UserID)
}

func (r *UserRepository) LookupGoogleAuth(tx *gorm.DB, subject string) (models.UserAuth, error) {
	var authRecord models.UserAuth
	err := tx.Where("provider = ? AND provider_sub = ?", "GOOGLE", subject).First(&authRecord).Error
	return authRecord, err
}

func (r *UserRepository) LookupAuthByEmail(tx *gorm.DB, email string) (models.UserAuth, error) {
	var authRecord models.UserAuth
	err := tx.Where("LOWER(email) = LOWER(?)", email).First(&authRecord).Error
	return authRecord, err
}

func (r *UserRepository) UsernameExists(tx *gorm.DB, username string) (bool, error) {
	var count int64
	if err := tx.Model(&models.UserProfile{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func FindUserByID(db *gorm.DB, userID string) (models.User, error) {
	var user models.User
	err := db.Preload("Profile").Preload("Stats").Where("id = ?", userID).First(&user).Error
	return user, err
}

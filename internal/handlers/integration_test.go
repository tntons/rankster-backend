package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rankster-backend/internal/models"
	"rankster-backend/internal/server"
	"rankster-backend/internal/testutil"
)

func TestSearchCategoriesReturnsSeededCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

	req := httptest.NewRequest(http.MethodGet, "/search/categories?q=coffee", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Items []struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) == 0 {
		t.Fatalf("expected at least one category")
	}
	if response.Items[0].Slug != "coffee" {
		t.Fatalf("expected coffee slug, got %q", response.Items[0].Slug)
	}
}

func TestFeedMainReturnsSeededItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

	req := httptest.NewRequest(http.MethodGet, "/feed/main", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) == 0 {
		t.Fatalf("expected feed items")
	}
}

func TestUserStatsRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

	req := httptest.NewRequest(http.MethodGet, "/user/stats", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestUserStatsReturnsSeededSubscriberData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	aliceUserID := seededUserID(t, database, "alice")
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

	req := httptest.NewRequest(http.MethodGet, "/user/stats", nil)
	req.Header.Set("Authorization", "Bearer "+aliceUserID)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		UserID string `json:"userId"`
		Totals struct {
			RanksCreated int `json:"ranksCreated"`
		} `json:"totals"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.UserID != aliceUserID {
		t.Fatalf("unexpected userId: %s", response.UserID)
	}
	if response.Totals.RanksCreated != 1 {
		t.Fatalf("expected 1 seeded rank, got %d", response.Totals.RanksCreated)
	}
}

func TestRankCreateCreatesPostAndUpdatesStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	aliceUserID := seededUserID(t, database, "alice")
	categoryID := seededCategoryID(t, database, "coffee")
	templateID := seededTemplateID(t, database, "Coffee Master Tier List")
	assetID := seededAssetID(t, database, "/assets/ranks/latte.svg")
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

	payload := map[string]any{
		"categoryId":   categoryID,
		"templateId":   templateID,
		"tierKey":      "B",
		"imageAssetId": assetID,
		"caption":      "Cold Brew belongs in B tier",
		"subjectTitle": "Cold Brew",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/rank/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+aliceUserID)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/user/stats", nil)
	statsReq.Header.Set("Authorization", "Bearer "+aliceUserID)
	statsRecorder := httptest.NewRecorder()

	router.ServeHTTP(statsRecorder, statsReq)

	if statsRecorder.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d; body=%s", statsRecorder.Code, http.StatusOK, statsRecorder.Body.String())
	}

	var statsResponse struct {
		Totals struct {
			RanksCreated int `json:"ranksCreated"`
		} `json:"totals"`
	}
	if err := json.Unmarshal(statsRecorder.Body.Bytes(), &statsResponse); err != nil {
		t.Fatalf("decode stats response: %v", err)
	}
	if statsResponse.Totals.RanksCreated != 2 {
		t.Fatalf("expected ranksCreated to increment to 2, got %d", statsResponse.Totals.RanksCreated)
	}
}

func seededUserID(t *testing.T, database *gorm.DB, username string) string {
	t.Helper()

	var profile models.UserProfile
	if err := database.Where("username = ?", username).First(&profile).Error; err != nil {
		t.Fatalf("load seeded user profile %q: %v", username, err)
	}
	return profile.UserID
}

func seededCategoryID(t *testing.T, database *gorm.DB, slug string) string {
	t.Helper()

	var category models.Category
	if err := database.Where("slug = ?", slug).First(&category).Error; err != nil {
		t.Fatalf("load seeded category %q: %v", slug, err)
	}
	return category.ID
}

func seededTemplateID(t *testing.T, database *gorm.DB, title string) string {
	t.Helper()

	var template models.TierListTemplate
	if err := database.Where("title = ?", title).First(&template).Error; err != nil {
		t.Fatalf("load seeded template %q: %v", title, err)
	}
	return template.ID
}

func seededAssetID(t *testing.T, database *gorm.DB, urlSuffix string) string {
	t.Helper()

	var asset models.Asset
	if err := database.Where("url LIKE ?", "%"+urlSuffix).First(&asset).Error; err != nil {
		t.Fatalf("load seeded asset %q: %v", urlSuffix, err)
	}
	return asset.ID
}

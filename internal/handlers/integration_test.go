package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"rankster-backend/internal/config"
	"rankster-backend/internal/models"
	"rankster-backend/internal/server"
	"rankster-backend/internal/testutil"
)

func TestMockLoginReturnsBearerSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	body := bytes.NewBufferString(`{"username":"me"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/mock-login", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		User        struct {
			Username    string `json:"username"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AccessToken == "" || response.TokenType != "Bearer" || response.User.Username != "me" || response.User.DisplayName != "Alex Rivera" {
		t.Fatalf("unexpected auth response: %+v", response)
	}
}

func TestFeedMainReturnsDatabaseRankPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	req := httptest.NewRequest(http.MethodGet, "/feed/main?limit=2", nil)
	req.Header.Set("Host", "localhost:8000")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Items []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			User  struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"items"`
		NextCursor any `json:"nextCursor"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 2 {
		t.Fatalf("expected 2 feed items, got %d", len(response.Items))
	}
	if response.Items[0].ID == "" || response.Items[0].User.Username == "" || response.Items[0].Title == "" {
		t.Fatalf("unexpected feed item payload: %+v", response.Items[0])
	}
	if response.NextCursor == nil {
		t.Fatalf("expected nextCursor for paged feed response")
	}
}

func TestProfileMeRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	req := httptest.NewRequest(http.MethodGet, "/profile/me", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProfileMeReturnsUserAndRankings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	req := httptest.NewRequest(http.MethodGet, "/profile/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Host", "localhost:8000")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
		Rankings []struct {
			ID string `json:"id"`
		} `json:"rankings"`
		LikedPosts []struct {
			ID string `json:"id"`
		} `json:"likedPosts"`
		Stats struct {
			TotalLikes int `json:"totalLikes"`
		} `json:"stats"`
		FavoriteCategories []struct {
			ID string `json:"id"`
		} `json:"favoriteCategories"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.User.Username != "me" {
		t.Fatalf("expected current db user, got %q", response.User.Username)
	}
	if len(response.LikedPosts) == 0 || response.Stats.TotalLikes == 0 || len(response.FavoriteCategories) == 0 {
		t.Fatalf("expected rich profile payload, got %+v", response)
	}
}

func TestUploadImageRequiresAuthAndStoresImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Chdir(t.TempDir())

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	unauthBody, unauthContentType := multipartImageBody(t)
	req := httptest.NewRequest(http.MethodPost, "/uploads/images", unauthBody)
	req.Header.Set("Content-Type", unauthContentType)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	token := mockLoginToken(t, router, "me")
	body, contentType := multipartImageBody(t)
	req = httptest.NewRequest(http.MethodPost, "/uploads/images", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	req.Host = "localhost:8000"
	recorder = httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		URL         string `json:"url"`
		Path        string `json:"path"`
		ContentType string `json:"contentType"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if !strings.HasPrefix(response.URL, "http://localhost:8000/uploads/images/") || response.ContentType != "image/png" {
		t.Fatalf("unexpected upload response: %+v", response)
	}
	if _, err := os.Stat(strings.TrimPrefix(response.Path, "/")); err != nil {
		t.Fatalf("expected uploaded file on disk: %v", err)
	}
}

func TestRankCreateCreatesNewDatabasePost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	payload := map[string]any{
		"title":       "Best Coffee Orders",
		"category":    "food",
		"description": "A database-backed ranking post",
		"tags":        []string{"coffee", "drinks"},
		"allItems": []map[string]any{
			{"id": "a", "name": "Latte", "imageUrl": "http://localhost:8000/uploads/images/test/latte.png"},
			{"id": "b", "name": "Americano"},
		},
		"tiers": map[string]any{
			"S": []map[string]any{{"id": "a", "name": "Latte", "imageUrl": "http://localhost:8000/uploads/images/test/latte.png"}},
			"A": []map[string]any{{"id": "b", "name": "Americano"}},
			"B": []map[string]any{},
			"C": []map[string]any{},
			"D": []map[string]any{},
		},
		"isPublic": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/rank/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "localhost:8000")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created struct {
		Title string `json:"title"`
		User  struct {
			Username string `json:"username"`
		} `json:"user"`
		AllItems []struct {
			ImageURL string `json:"imageUrl"`
		} `json:"allItems"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Title != "Best Coffee Orders" || created.User.Username != "me" {
		t.Fatalf("unexpected create response: %+v", created)
	}
	if len(created.AllItems) == 0 || created.AllItems[0].ImageURL == "" {
		t.Fatalf("expected imageUrl to persist in created rank, got %+v", created.AllItems)
	}
}

func TestRankTierRowsPersistCustomLabelsAndDeletedRows(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	payload := map[string]any{
		"title":       "Custom Tier Ranking",
		"category":    "food",
		"description": "A ranking with custom tiers",
		"tags":        []string{"custom"},
		"allItems": []map[string]any{
			{"id": "ramen", "name": "Ramen"},
			{"id": "pho", "name": "Pho"},
		},
		"tierRows": []map[string]any{
			{
				"id":    "legend",
				"label": "  Legends  ",
				"items": []map[string]any{{"id": "ramen", "name": "Ramen"}},
			},
			{
				"id":    "try-next",
				"label": "Worth Trying",
				"items": []map[string]any{{"id": "pho", "name": "Pho"}},
			},
		},
		"isPublic": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal create payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/rank/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created struct {
		ID       string `json:"id"`
		TierRows []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"tierRows"`
		Tiers struct {
			S []struct {
				ID string `json:"id"`
			} `json:"S"`
		} `json:"tiers"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created post id")
	}
	if len(created.TierRows) != 2 || created.TierRows[0].ID != "legend" || created.TierRows[0].Label != "Legends" || len(created.TierRows[0].Items) != 1 || created.TierRows[0].Items[0].ID != "ramen" {
		t.Fatalf("custom tierRows were not returned on create: %+v", created.TierRows)
	}
	if len(created.Tiers.S) != 0 {
		t.Fatalf("legacy tiers should remain populated only for matching default keys, got %+v", created.Tiers)
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/post/"+created.ID, nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("fetch status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var fetched struct {
		TierRows []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"tierRows"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode fetched post: %v", err)
	}
	if len(fetched.TierRows) != 2 || fetched.TierRows[1].ID != "try-next" || fetched.TierRows[1].Label != "Worth Trying" {
		t.Fatalf("custom tierRows were not persisted on fetch: %+v", fetched.TierRows)
	}

	updatePayload := map[string]any{
		"title":       "Custom Tier Ranking",
		"category":    "food",
		"description": "A ranking with one custom tier removed",
		"tags":        []string{"custom"},
		"allItems": []map[string]any{
			{"id": "ramen", "name": "Ramen"},
		},
		"tierRows": []map[string]any{
			{
				"id":    "legend",
				"label": "Hall of Fame",
				"items": []map[string]any{{"id": "ramen", "name": "Ramen"}},
			},
		},
		"isPublic": true,
	}
	body, err = json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("marshal update payload: %v", err)
	}

	req = httptest.NewRequest(http.MethodPatch, "/feed/post/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var updated struct {
		TierRows []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"tierRows"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated post: %v", err)
	}
	if len(updated.TierRows) != 1 || updated.TierRows[0].ID != "legend" || updated.TierRows[0].Label != "Hall of Fame" {
		t.Fatalf("deleted row or renamed label did not persist: %+v", updated.TierRows)
	}
}

func TestPostOwnerCanUpdateAndDeletePost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")
	otherToken := mockLoginToken(t, router, "tierqueen")

	payload := map[string]any{
		"title":       "Editable Coffee Ranking",
		"category":    "food",
		"description": "Original description",
		"tags":        []string{"coffee", "draft"},
		"allItems": []map[string]any{
			{"id": "latte", "name": "Latte"},
			{"id": "mocha", "name": "Mocha"},
		},
		"tiers": map[string]any{
			"S": []map[string]any{{"id": "latte", "name": "Latte"}},
			"A": []map[string]any{},
			"B": []map[string]any{{"id": "mocha", "name": "Mocha"}},
			"C": []map[string]any{},
			"D": []map[string]any{},
		},
		"isPublic": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal create payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/rank/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created struct {
		ID      string `json:"id"`
		CanEdit bool   `json:"canEdit"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	if created.ID == "" || !created.CanEdit {
		t.Fatalf("expected editable created post, got %+v", created)
	}

	updatePayload := map[string]any{
		"title":       "Updated Coffee Ranking",
		"category":    "food",
		"description": "Updated description",
		"tags":        []string{"coffee", "updated"},
		"allItems": []map[string]any{
			{"id": "latte", "name": "Latte"},
			{"id": "mocha", "name": "Mocha"},
		},
		"tiers": map[string]any{
			"S": []map[string]any{{"id": "mocha", "name": "Mocha"}},
			"A": []map[string]any{},
			"B": []map[string]any{},
			"C": []map[string]any{},
			"D": []map[string]any{{"id": "latte", "name": "Latte"}},
		},
		"isPublic": false,
	}
	body, err = json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("marshal update payload: %v", err)
	}

	req = httptest.NewRequest(http.MethodPatch, "/feed/post/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var updated struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		IsPublic    bool     `json:"isPublic"`
		CanEdit     bool     `json:"canEdit"`
		Tiers       struct {
			S []struct {
				Name string `json:"name"`
			} `json:"S"`
			D []struct {
				Name string `json:"name"`
			} `json:"D"`
		} `json:"tiers"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated post: %v", err)
	}
	if updated.Title != "Updated Coffee Ranking" || updated.Description != "Updated description" || updated.IsPublic || !updated.CanEdit || len(updated.Tags) != 2 || updated.Tags[1] != "updated" {
		t.Fatalf("unexpected updated post: %+v", updated)
	}
	if len(updated.Tiers.S) != 1 || updated.Tiers.S[0].Name != "Mocha" || len(updated.Tiers.D) != 1 || updated.Tiers.D[0].Name != "Latte" {
		t.Fatalf("tier list was not updated: %+v", updated.Tiers)
	}

	req = httptest.NewRequest(http.MethodDelete, "/feed/post/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("other user delete status = %d, want %d; body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/feed/post/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/post/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("deleted post status = %d, want %d; body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
}

func TestPostCommentAppendsCommentToPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	req := httptest.NewRequest(http.MethodGet, "/feed/main?limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("feed status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var feed struct {
		Items []struct {
			ID       string `json:"id"`
			Comments []struct {
				ID string `json:"id"`
			} `json:"comments"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &feed); err != nil {
		t.Fatalf("decode feed response: %v", err)
	}
	if len(feed.Items) == 0 || feed.Items[0].ID == "" {
		t.Fatalf("expected a feed post, got %+v", feed)
	}

	postID := feed.Items[0].ID
	initialCommentCount := len(feed.Items[0].Comments)
	body := bytes.NewBufferString(`{"text":"Testing comments from integration suite"}`)
	req = httptest.NewRequest(http.MethodPost, "/feed/post/"+postID+"/comments", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("comment status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created struct {
		ID   string `json:"id"`
		Text string `json:"text"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created comment: %v", err)
	}
	if created.ID == "" || created.Text != "Testing comments from integration suite" || created.User.Username != "me" {
		t.Fatalf("unexpected created comment: %+v", created)
	}

	req = httptest.NewRequest(http.MethodPost, "/feed/comments/"+created.ID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("like comment status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var likedComment struct {
		Likes   int  `json:"likes"`
		IsLiked bool `json:"isLiked"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &likedComment); err != nil {
		t.Fatalf("decode liked comment: %v", err)
	}
	if likedComment.Likes != 1 || !likedComment.IsLiked {
		t.Fatalf("unexpected liked comment response: %+v", likedComment)
	}

	req = httptest.NewRequest(http.MethodPost, "/feed/comments/"+created.ID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("second like comment status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &likedComment); err != nil {
		t.Fatalf("decode second liked comment: %v", err)
	}
	if likedComment.Likes != 1 || !likedComment.IsLiked {
		t.Fatalf("comment like should be idempotent, got %+v", likedComment)
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/post/"+postID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("post status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var post struct {
		Comments []struct {
			Text    string `json:"text"`
			Likes   int    `json:"likes"`
			IsLiked bool   `json:"isLiked"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode post response: %v", err)
	}
	if len(post.Comments) != initialCommentCount+1 || post.Comments[0].Text != "Testing comments from integration suite" {
		t.Fatalf("comment was not appended: %+v", post.Comments)
	}
	if post.Comments[0].Likes != 1 || !post.Comments[0].IsLiked {
		t.Fatalf("comment like was not hydrated: %+v", post.Comments[0])
	}

	req = httptest.NewRequest(http.MethodDelete, "/feed/comments/"+created.ID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unlike comment status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &likedComment); err != nil {
		t.Fatalf("decode unliked comment: %v", err)
	}
	if likedComment.Likes != 0 || likedComment.IsLiked {
		t.Fatalf("unexpected unliked comment response: %+v", likedComment)
	}
}

func TestPostLikePersistsHydratesIsIdempotentUnlikesAndNotifies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	ownerToken := mockLoginToken(t, router, "tierqueen")
	likerToken := mockLoginToken(t, router, "me")

	createBody := bytes.NewBufferString(`{
		"title":"Backend Like Test Ranking",
		"category":"anime",
		"description":"Created by integration test",
		"tags":["anime","backend"],
		"tierRows":[
			{"id":"S","label":"S","items":[{"id":"frieren","name":"Frieren"}]},
			{"id":"A","label":"A","items":[{"id":"dungeon","name":"Dungeon Meshi"}]}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/rank/create", createBody)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create rank status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("created post missing id: %+v", created)
	}

	var owner models.User
	if err := database.
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("user_profiles.username = ?", "tierqueen").
		First(&owner).Error; err != nil {
		t.Fatalf("load owner: %v", err)
	}

	var initialNotificationCount int64
	if err := database.Model(&models.Notification{}).Where("user_id = ?", owner.ID).Count(&initialNotificationCount).Error; err != nil {
		t.Fatalf("count initial notifications: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/feed/post/"+created.ID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+likerToken)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("like post status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var likedPost struct {
		Likes   int  `json:"likes"`
		IsLiked bool `json:"isLiked"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &likedPost); err != nil {
		t.Fatalf("decode liked post: %v", err)
	}
	if likedPost.Likes != 1 || !likedPost.IsLiked {
		t.Fatalf("unexpected liked post response: %+v", likedPost)
	}

	var likeRows int64
	if err := database.Model(&models.PostLike{}).Where("post_id = ?", created.ID).Count(&likeRows).Error; err != nil {
		t.Fatalf("count post likes: %v", err)
	}
	if likeRows != 1 {
		t.Fatalf("post_likes rows = %d, want 1", likeRows)
	}

	var metrics models.PostMetrics
	if err := database.Where("post_id = ?", created.ID).First(&metrics).Error; err != nil {
		t.Fatalf("load post metrics: %v", err)
	}
	if metrics.LikeCount != 1 || metrics.HotScore != 1 {
		t.Fatalf("metrics after like = %+v, want like_count=1 hot_score=1", metrics)
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/post/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+likerToken)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("refresh post status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var refreshed struct {
		Likes   int  `json:"likes"`
		IsLiked bool `json:"isLiked"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode refreshed post: %v", err)
	}
	if refreshed.Likes != 1 || !refreshed.IsLiked {
		t.Fatalf("post like was not hydrated after refresh: %+v", refreshed)
	}

	var notificationsAfterLike int64
	if err := database.Model(&models.Notification{}).Where("user_id = ?", owner.ID).Count(&notificationsAfterLike).Error; err != nil {
		t.Fatalf("count notifications after like: %v", err)
	}
	if notificationsAfterLike != initialNotificationCount+1 {
		t.Fatalf("notification count after like = %d, want %d", notificationsAfterLike, initialNotificationCount+1)
	}

	var notification models.Notification
	if err := database.
		Where("user_id = ? AND type = ? AND action_href = ?", owner.ID, "like", "/topic/"+created.ID).
		First(&notification).Error; err != nil {
		t.Fatalf("load like notification: %v", err)
	}
	if notification.ActorUserID == nil || *notification.ActorUserID == owner.ID {
		t.Fatalf("unexpected like notification actor: %+v", notification)
	}

	req = httptest.NewRequest(http.MethodPost, "/feed/post/"+created.ID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+likerToken)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("second like post status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &likedPost); err != nil {
		t.Fatalf("decode second liked post: %v", err)
	}
	if likedPost.Likes != 1 || !likedPost.IsLiked {
		t.Fatalf("post like should be idempotent, got %+v", likedPost)
	}

	if err := database.Model(&models.PostLike{}).Where("post_id = ?", created.ID).Count(&likeRows).Error; err != nil {
		t.Fatalf("count post likes after idempotent like: %v", err)
	}
	if likeRows != 1 {
		t.Fatalf("post_likes rows after idempotent like = %d, want 1", likeRows)
	}
	if err := database.Model(&models.Notification{}).Where("user_id = ?", owner.ID).Count(&notificationsAfterLike).Error; err != nil {
		t.Fatalf("count notifications after idempotent like: %v", err)
	}
	if notificationsAfterLike != initialNotificationCount+1 {
		t.Fatalf("notification count after idempotent like = %d, want %d", notificationsAfterLike, initialNotificationCount+1)
	}

	req = httptest.NewRequest(http.MethodDelete, "/feed/post/"+created.ID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+likerToken)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unlike post status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &likedPost); err != nil {
		t.Fatalf("decode unliked post: %v", err)
	}
	if likedPost.Likes != 0 || likedPost.IsLiked {
		t.Fatalf("unexpected unliked post response: %+v", likedPost)
	}

	if err := database.Where("post_id = ?", created.ID).First(&metrics).Error; err != nil {
		t.Fatalf("load post metrics after unlike: %v", err)
	}
	if metrics.LikeCount != 0 || metrics.HotScore != 0 {
		t.Fatalf("metrics after unlike = %+v, want like_count=0 hot_score=0", metrics)
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/post/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+likerToken)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("refresh unliked post status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode refreshed unliked post: %v", err)
	}
	if refreshed.Likes != 0 || refreshed.IsLiked {
		t.Fatalf("post unlike was not hydrated after refresh: %+v", refreshed)
	}
}

func TestMessagesThreadDetailReturnsConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")
	threadID := firstThreadID(t, router, token)

	req := httptest.NewRequest(http.MethodGet, "/messages/threads/"+threadID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		ID   string `json:"id"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
		Messages []struct {
			Text string `json:"text"`
			Mine bool   `json:"mine"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID == "" || response.User.Username == "" || len(response.Messages) == 0 {
		t.Fatalf("unexpected thread detail payload: %+v", response)
	}
}

func TestStartMessageThreadOpensConversationByUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	body := bytes.NewBufferString(`{"username":"animequeen"}`)
	req := httptest.NewRequest(http.MethodPost, "/messages/threads", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		ID   string `json:"id"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
		Messages []struct {
			Text string `json:"text"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID == "" || response.User.Username != "animequeen" || len(response.Messages) == 0 {
		t.Fatalf("unexpected started thread payload: %+v", response)
	}
}

func TestStartMessageThreadRejectsCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	body := bytes.NewBufferString(`{"username":"me"}`)
	req := httptest.NewRequest(http.MethodPost, "/messages/threads", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestPostMessageAppendsConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")
	threadID := firstThreadID(t, router, token)

	body := bytes.NewBufferString(`{"text":"Testing from integration suite"}`)
	req := httptest.NewRequest(http.MethodPost, "/messages/threads/"+threadID+"/messages", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/messages/threads/"+threadID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	var response struct {
		Messages []struct {
			Text string `json:"text"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Messages) == 0 || response.Messages[len(response.Messages)-1].Text != "Testing from integration suite" {
		t.Fatalf("message was not appended: %+v", response.Messages)
	}
}

func TestMessageThreadWebSocketSendsConversationEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")
	threadID := firstThreadID(t, router, token)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	values := url.Values{}
	values.Set("token", token)
	socketURL := strings.Replace(testServer.URL, "http://", "ws://", 1) + "/messages/threads/" + threadID + "/ws?" + values.Encode()

	socket, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer socket.Close()

	if err := socket.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	var ready struct {
		Type string `json:"type"`
	}
	if err := socket.ReadJSON(&ready); err != nil {
		t.Fatalf("read ready event: %v", err)
	}
	if ready.Type != "ready" {
		t.Fatalf("ready type = %q, want ready", ready.Type)
	}

	if err := socket.WriteJSON(map[string]string{"type": "message", "text": "Realtime test message"}); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	var event struct {
		Type    string `json:"type"`
		Message struct {
			Text string `json:"text"`
			Mine bool   `json:"mine"`
		} `json:"message"`
	}
	if err := socket.ReadJSON(&event); err != nil {
		t.Fatalf("read message event: %v", err)
	}
	if event.Type != "message" || event.Message.Text != "Realtime test message" || !event.Message.Mine {
		t.Fatalf("unexpected message event: %+v", event)
	}
}

func TestFollowAndUnfollowProfileUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	req := httptest.NewRequest(http.MethodPost, "/profile/tierqueen/follow", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("follow status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/profile/tierqueen/follow", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unfollow status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestNotificationsListAndMarkRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("notifications status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		UnreadCount int `json:"unreadCount"`
		Items       []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Read  bool   `json:"read"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode notifications response: %v", err)
	}
	if response.UnreadCount == 0 || len(response.Items) == 0 || response.Items[0].ID == "" || response.Items[0].Title == "" {
		t.Fatalf("unexpected notifications response: %+v", response)
	}

	req = httptest.NewRequest(http.MethodPost, "/notifications/"+response.Items[0].ID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("mark read status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("mark all read status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var updated struct {
		UnreadCount int `json:"unreadCount"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode mark all response: %v", err)
	}
	if updated.UnreadCount != 0 {
		t.Fatalf("unreadCount = %d, want 0", updated.UnreadCount)
	}
}

func TestActivityNotificationsExcludeDirectMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "rankmaster99")

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("notifications status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		UnreadCount int `json:"unreadCount"`
		Items       []struct {
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode notifications response: %v", err)
	}
	if response.UnreadCount != 0 || len(response.Items) != 0 {
		t.Fatalf("direct messages should stay out of activity notifications: %+v", response)
	}
}

func TestOpeningMessageThreadClearsUnreadCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	req := httptest.NewRequest(http.MethodGet, "/messages/threads", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("messages status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var threadsResponse struct {
		Items []struct {
			ID     string `json:"id"`
			Unread int    `json:"unread"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &threadsResponse); err != nil {
		t.Fatalf("decode messages response: %v", err)
	}

	var unreadThreadID string
	totalUnread := 0
	selectedUnread := 0
	for _, thread := range threadsResponse.Items {
		totalUnread += thread.Unread
		if unreadThreadID == "" && thread.Unread > 0 {
			unreadThreadID = thread.ID
			selectedUnread = thread.Unread
		}
	}
	if unreadThreadID == "" {
		t.Fatalf("expected at least one unread demo thread: %+v", threadsResponse)
	}

	req = httptest.NewRequest(http.MethodGet, "/messages/threads/"+unreadThreadID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("thread status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/messages/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unread count status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var countResponse struct {
		UnreadCount int `json:"unreadCount"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &countResponse); err != nil {
		t.Fatalf("decode unread count response: %v", err)
	}
	expectedUnread := totalUnread - selectedUnread
	if countResponse.UnreadCount != expectedUnread {
		t.Fatalf("unreadCount = %d, want %d after opening thread", countResponse.UnreadCount, expectedUnread)
	}
}

func TestPinAndUnpinProfilePost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database, testConfig())

	token := mockLoginToken(t, router, "me")

	req := httptest.NewRequest(http.MethodGet, "/profile/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	var profile struct {
		Rankings []struct {
			ID string `json:"id"`
		} `json:"rankings"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode profile response: %v", err)
	}
	if len(profile.Rankings) == 0 {
		t.Fatalf("expected rankings in profile response")
	}

	postID := profile.Rankings[0].ID
	req = httptest.NewRequest(http.MethodPost, "/profile/me/pinned/"+postID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("pin status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/profile/me/pinned/"+postID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unpin status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func mockLoginToken(t *testing.T, router http.Handler, username string) string {
	t.Helper()

	body := bytes.NewBufferString(`{"username":"` + username + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/mock-login", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return response.AccessToken
}

func multipartImageBody(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile("file", "pixel.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	// 1x1 transparent PNG.
	file.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89,
	})
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func firstThreadID(t *testing.T, router http.Handler, token string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/messages/threads", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("thread list status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode threads response: %v", err)
	}
	if len(response.Items) == 0 || response.Items[0].ID == "" {
		t.Fatalf("expected at least one message thread, got %+v", response.Items)
	}
	return response.Items[0].ID
}

func testConfig() config.Config {
	return config.Config{
		PublicBaseURL:   "http://localhost:8000",
		AuthTokenSecret: "test-auth-secret",
		EnableMockAuth:  true,
	}
}

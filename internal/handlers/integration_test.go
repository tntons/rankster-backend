package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"rankster-backend/internal/server"
	"rankster-backend/internal/testutil"
)

func TestMockLoginReturnsBearerSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

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
	RegisterRoutes(router, database)

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
	RegisterRoutes(router, database)

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
	RegisterRoutes(router, database)

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
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.User.Username != "me" {
		t.Fatalf("expected current db user, got %q", response.User.Username)
	}
}

func TestRankCreateCreatesNewDatabasePost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := testutil.NewTestDatabase(t)
	router := server.BuildRouter(database)
	RegisterRoutes(router, database)

	token := mockLoginToken(t, router, "me")

	payload := map[string]any{
		"title":       "Best Coffee Orders",
		"category":    "food",
		"description": "A database-backed ranking post",
		"tags":        []string{"coffee", "drinks"},
		"allItems": []map[string]any{
			{"id": "a", "name": "Latte"},
			{"id": "b", "name": "Americano"},
		},
		"tiers": map[string]any{
			"S": []map[string]any{{"id": "a", "name": "Latte"}},
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
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Title != "Best Coffee Orders" || created.User.Username != "me" {
		t.Fatalf("unexpected create response: %+v", created)
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

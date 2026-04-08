package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildRouterHealthz(t *testing.T) {
	t.Parallel()

	router := BuildRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != `{"ok":true}` {
		t.Fatalf("healthz body = %s", body)
	}
}

func TestBuildRouterServesAssets(t *testing.T) {
	t.Parallel()

	router := BuildRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/assets/ranks/latte.svg", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "image/svg+xml") {
		t.Fatalf("unexpected content-type: %s", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "<svg") {
		t.Fatalf("expected svg body, got %q", recorder.Body.String())
	}
}

func TestBuildRouterHandlesDevCORSPreflight(t *testing.T) {
	t.Parallel()

	router := BuildRouter(nil)
	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("allow-origin = %q, want localhost origin", got)
	}
}

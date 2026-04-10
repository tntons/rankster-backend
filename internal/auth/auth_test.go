package auth

import (
	"strings"
	"testing"
	"time"
)

func TestFromAuthorization(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	validToken, err := IssueUserToken("user-123", secret, time.Hour)
	if err != nil {
		t.Fatalf("IssueUserToken(): %v", err)
	}

	testCases := []struct {
		name string
		in   string
		want Context
	}{
		{name: "empty header", in: "", want: Context{Kind: "anonymous"}},
		{name: "wrong scheme", in: "Basic abc", want: Context{Kind: "anonymous"}},
		{name: "missing token", in: "Bearer   ", want: Context{Kind: "anonymous"}},
		{name: "valid bearer", in: "Bearer " + validToken, want: Context{Kind: "user", UserID: "user-123"}},
		{name: "case insensitive prefix", in: "bearer " + validToken, want: Context{Kind: "user", UserID: "user-123"}},
		{name: "invalid token", in: "Bearer invalid", want: Context{Kind: "anonymous"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := FromAuthorization(testCase.in, secret)
			if got != testCase.want {
				t.Fatalf("FromAuthorization(%q) = %#v, want %#v", testCase.in, got, testCase.want)
			}
		})
	}
}

func TestIssueUserTokenIncludesTwoSegments(t *testing.T) {
	t.Parallel()

	token, err := IssueUserToken("user-456", "test-secret", time.Hour)
	if err != nil {
		t.Fatalf("IssueUserToken(): %v", err)
	}

	if parts := strings.Split(token, "."); len(parts) != 2 {
		t.Fatalf("expected signed token with 2 segments, got %q", token)
	}
}

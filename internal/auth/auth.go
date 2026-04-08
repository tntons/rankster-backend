package auth

import (
	"strings"
)

type Context struct {
	Kind   string
	UserID string
}

func FromAuthorization(header string) Context {
	if header == "" {
		return Context{Kind: "anonymous"}
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return Context{Kind: "anonymous"}
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	if token == "" {
		return Context{Kind: "anonymous"}
	}
	return Context{Kind: "user", UserID: token}
}

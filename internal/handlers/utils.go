package handlers

import (
	"strconv"

	"encoding/base64"
)

func decodeCursor(raw string) int {
	if raw == "" {
		return 0
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0
	}
	return value
}

func parseIntWithDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

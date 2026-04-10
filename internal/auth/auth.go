package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Context struct {
	Kind   string
	UserID string
}

type userTokenClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

func FromAuthorization(header, secret string) Context {
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

	userID, ok := VerifyUserToken(token, secret)
	if !ok {
		return Context{Kind: "anonymous"}
	}

	return Context{Kind: "user", UserID: userID}
}

func IssueUserToken(userID, secret string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(secret) == "" {
		return "", ErrInvalidToken
	}

	claims := userTokenClaims{
		Sub: userID,
		Exp: time.Now().Add(ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(encodedPayload, secret)
	return encodedPayload + "." + signature, nil
}

func VerifyUserToken(token, secret string) (string, bool) {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(secret) == "" {
		return "", false
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", false
	}

	payloadPart := parts[0]
	signaturePart := parts[1]
	expectedSignature := sign(payloadPart, secret)
	if !hmac.Equal([]byte(signaturePart), []byte(expectedSignature)) {
		return "", false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return "", false
	}

	var claims userTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", false
	}

	if strings.TrimSpace(claims.Sub) == "" || claims.Exp <= time.Now().Unix() {
		return "", false
	}

	return claims.Sub, true
}

var ErrInvalidToken = errors.New("invalid token")

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

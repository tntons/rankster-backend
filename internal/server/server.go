package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func BuildRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), devCORS())
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			writeDevCORSHeaders(c)
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "route not found"})
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.GET("/assets/avatars/:slug.svg", serveAvatarSVG)
	router.GET("/assets/ranks/:slug.svg", serveRankSVG)
	return router
}

func devCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		writeDevCORSHeaders(c)

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}

func writeDevCORSHeaders(c *gin.Context) {
	origin := c.GetHeader("Origin")
	if origin == "" || !isAllowedDevOrigin(origin) {
		return
	}

	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Vary", "Origin")
	c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
}

func isAllowedDevOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
}

func serveAvatarSVG(c *gin.Context) {
	slug := sanitizeSlug(c.Param("slug"))
	label := strings.ToUpper(firstRune(slug))
	if label == "" {
		label = "R"
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 160 160" role="img" aria-label="%s avatar">
<defs>
<linearGradient id="avatarBg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
<stop offset="0%%" stop-color="#8b5cf6"/>
<stop offset="100%%" stop-color="#14b8a6"/>
</linearGradient>
</defs>
<rect width="160" height="160" rx="48" fill="url(#avatarBg)"/>
<circle cx="80" cy="62" r="28" fill="rgba(255,255,255,0.22)"/>
<path d="M36 136c8-25 26-38 44-38s36 13 44 38" fill="rgba(255,255,255,0.22)"/>
<text x="80" y="93" text-anchor="middle" font-size="54" font-family="Arial, Helvetica, sans-serif" font-weight="700" fill="#ffffff">%s</text>
</svg>`, html.EscapeString(slug), html.EscapeString(label))

	c.Data(http.StatusOK, "image/svg+xml; charset=utf-8", []byte(svg))
}

func serveRankSVG(c *gin.Context) {
	slug := sanitizeSlug(c.Param("slug"))
	label := prettyLabel(slug)
	if label == "" {
		label = "Rankster"
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 720" role="img" aria-label="%s cover">
<defs>
<linearGradient id="coverBg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
<stop offset="0%%" stop-color="#111827"/>
<stop offset="45%%" stop-color="#7c3aed"/>
<stop offset="100%%" stop-color="#f59e0b"/>
</linearGradient>
</defs>
<rect width="1200" height="720" rx="42" fill="url(#coverBg)"/>
<circle cx="1010" cy="154" r="96" fill="rgba(255,255,255,0.14)"/>
<circle cx="160" cy="620" r="120" fill="rgba(255,255,255,0.10)"/>
<rect x="84" y="88" width="132" height="132" rx="30" fill="rgba(255,255,255,0.18)"/>
<text x="150" y="170" text-anchor="middle" font-size="72" font-family="Arial, Helvetica, sans-serif" font-weight="700" fill="#ffffff">S</text>
<text x="84" y="346" font-size="78" font-family="Arial, Helvetica, sans-serif" font-weight="700" fill="#ffffff">%s</text>
<text x="84" y="420" font-size="34" font-family="Arial, Helvetica, sans-serif" fill="rgba(255,255,255,0.82)">Seeded Rankster demo asset</text>
</svg>`, html.EscapeString(label), html.EscapeString(label))

	c.Data(http.StatusOK, "image/svg+xml; charset=utf-8", []byte(svg))
}

func sanitizeSlug(raw string) string {
	value := strings.Trim(strings.ToLower(raw), "/ ")
	if value == "" {
		return "rankster"
	}

	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "rankster"
	}
	return builder.String()
}

func prettyLabel(slug string) string {
	parts := strings.Split(slug, "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func firstRune(value string) string {
	for _, r := range value {
		return string(r)
	}
	return ""
}

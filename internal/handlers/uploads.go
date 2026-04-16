package handlers

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"path/filepath"
)

func (h *FrontendHandler) UploadImage(c *gin.Context) {
	user, ok := h.requireUser(c)
	if !ok {
		return
	}

	const maxUploadBytes = 5 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "image file is required"})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "image must be 5MB or smaller"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "failed to read image"})
		return
	}
	defer file.Close()

	header := make([]byte, 512)
	n, _ := file.Read(header)
	contentType := http.DetectContentType(header[:n])
	ext, ok := imageExtensionForContentType(contentType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "only JPEG, PNG, WebP, and GIF images are supported"})
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "failed to prepare image"})
		return
	}

	uploadDir := filepath.Join(h.uploadDir, "images", user.ID)
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to prepare upload storage"})
		return
	}

	filename := uuid.NewString() + ext
	destination := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(fileHeader, destination); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to save image"})
		return
	}

	relativeURL := "/uploads/images/" + user.ID + "/" + filename
	c.JSON(http.StatusCreated, gin.H{
		"url":         h.publicURL(relativeURL),
		"path":        relativeURL,
		"contentType": contentType,
		"size":        fileHeader.Size,
	})
}

func imageExtensionForContentType(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}

func (h *FrontendHandler) publicURL(relativePath string) string {
	return h.publicBaseURL + relativePath
}

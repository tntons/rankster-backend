package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) UploadImage(c *gin.Context) {
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

	filename := uuid.NewString() + ext
	if h.cloudinaryConfigErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "image storage is not configured correctly"})
		return
	}
	if h.cloudinaryClient != nil {
		if _, err := file.Seek(0, 0); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "failed to prepare image"})
			return
		}

		result, err := h.cloudinaryClient.Upload.Upload(c.Request.Context(), file, uploader.UploadParams{
			Folder:       h.cloudinaryFolder + "/" + user.ID,
			PublicID:     strings.TrimSuffix(filename, ext),
			ResourceType: "image",
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to upload image"})
			return
		}

		imageURL := result.SecureURL
		if imageURL == "" {
			imageURL = result.URL
		}
		if imageURL == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "upload provider did not return an image URL"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"url":         imageURL,
			"path":        result.PublicID,
			"contentType": contentType,
			"size":        fileHeader.Size,
		})
		return
	}

	uploadDir := filepath.Join(h.uploadDir, "images", user.ID)
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "failed to prepare upload storage"})
		return
	}

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

func (h *Handler) publicURL(relativePath string) string {
	return h.publicBaseURL + relativePath
}

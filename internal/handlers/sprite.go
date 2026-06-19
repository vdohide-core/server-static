package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"server-static/internal/db/models"
	"server-static/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
)

// HandleSpriteVTT handles GET /{fileSlug}/sprite/sprite.vtt
func (h *Handler) HandleSpriteVTT(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	slug := strings.TrimSuffix(path, "/sprite/sprite.vtt")

	if slug == "" || strings.Contains(slug, "/") {
		HandleNotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	file, storageHostPort, err := h.resolveSpriteSource(ctx, slug)
	if err != nil {
		log.Printf("[Sprite] VTT resolve failed slug=%s: %v", slug, err)
		HandleNotFound(w, r)
		return
	}

	vttURL := fmt.Sprintf("http://%s/%s/sprite/sprite.vtt", storageHostPort, file.Slug)
	vttContent, err := utils.FetchURLContent(ctx, vttURL)
	if err != nil {
		log.Printf("[Sprite] Failed to fetch VTT from %s: %v", vttURL, err)
		HandleNotFound(w, r)
		return
	}

	responseBody := []byte(vttContent)
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(responseBody)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=63072000, immutable")
	w.Write(responseBody)
}

// HandleSpriteImage handles GET /{fileSlug}/sprite/sprite-{n}.jpg
func (h *Handler) HandleSpriteImage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/sprite/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		HandleNotFound(w, r)
		return
	}

	slug := parts[0]
	filename := parts[1]

	if !isValidSpriteFilename(filename) {
		HandleNotFound(w, r)
		return
	}

	if strings.Contains(slug, "/") {
		HandleNotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	file, storageHostPort, err := h.resolveSpriteSource(ctx, slug)
	if err != nil {
		HandleNotFound(w, r)
		return
	}

	sourceURL := spriteStorageURL(storageHostPort, file.Slug, filename)
	resp, err := fetchSpriteImage(ctx, sourceURL)
	if err != nil {
		log.Printf("[Sprite] Upstream request failed: %s → %v", sourceURL, err)
		HandleNotFound(w, r)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "image/jpeg")
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.Header().Set("Cache-Control", "public, max-age=63072000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	buf := make([]byte, 32*1024)
	io.CopyBuffer(w, resp.Body, buf)
}

func (h *Handler) resolveSpriteSource(ctx context.Context, slug string) (*models.File, string, error) {
	var file models.File
	if err := models.FileModel.Col().FindOne(ctx, bson.M{"slug": slug}).Decode(&file); err != nil {
		return nil, "", err
	}
	if file.IsTrashed() || file.IsDeleted() {
		return nil, "", fmt.Errorf("file unavailable")
	}

	var media models.Media
	err := models.MediaModel.Col().FindOne(ctx, bson.M{
		"fileId":    file.ID,
		"type":      models.MediaTypeThumbnail,
		"deletedAt": nil,
	}).Decode(&media)
	if err != nil {
		return nil, "", fmt.Errorf("thumbnail media: %w", err)
	}

	storageID := ""
	if media.StorageID != nil {
		storageID = *media.StorageID
	}

	var storage models.Storage
	if err := models.StorageModel.Col().FindOne(ctx, bson.M{"_id": storageID}).Decode(&storage); err != nil {
		return nil, "", fmt.Errorf("storage: %w", err)
	}

	storageHostPort := storage.GetHostPort()
	if storageHostPort == "" {
		return nil, "", fmt.Errorf("storage has no host")
	}

	return &file, storageHostPort, nil
}

func spriteStorageURL(hostPort, slug, filename string) string {
	return fmt.Sprintf("http://%s/%s/sprite/%s", hostPort, slug, filename)
}

func fetchSpriteImage(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("status %d from %s", resp.StatusCode, url)
	}
	return resp, nil
}

func isValidSpriteFilename(filename string) bool {
	if !strings.HasPrefix(filename, "sprite-") || !strings.HasSuffix(filename, ".jpg") {
		return false
	}
	return isDigitsOnly(strings.TrimSuffix(strings.TrimPrefix(filename, "sprite-"), ".jpg"))
}

func isDigitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

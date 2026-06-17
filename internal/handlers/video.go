package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"server-static/internal/db/models"
	"server-static/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
)

// HandleVideo handles GET /{mediaSlug}/video.m3u8
// Proxies the HLS segment playlist from storage and rewrites segment URLs
func (h *Handler) HandleVideo(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	slug := strings.TrimSuffix(path, "/video.m3u8")

	if slug == "" {
		HandleNotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// ─── Step 1: Find media by slug ──────────────────────────────────────
	var media models.Media
	err := models.MediaModel.Col().FindOne(ctx, bson.M{
		"slug":      slug,
		"deletedAt": bson.M{"$eq": nil},
	}).Decode(&media)
	if err != nil {
		log.Printf("[Video] Media not found: %s", slug)
		HandleNotFound(w, r)
		return
	}

	// ─── Step 2: Find storage ────────────────────────────────────────────
	storageID := ""
	if media.StorageID != nil {
		storageID = *media.StorageID
	}

	var storage models.Storage
	err = models.StorageModel.Col().FindOne(ctx, bson.M{"_id": storageID}).Decode(&storage)
	if err != nil {
		log.Printf("[Video] Storage not found for media=%s (storageId=%s)", slug, storageID)
		HandleNotFound(w, r)
		return
	}

	publicURL := ""
	if storage.PublicURL != nil {
		publicURL = *storage.PublicURL
	}
	if publicURL == "" {
		log.Printf("[Video] Storage has no publicUrl: storageId=%s", storage.ID)
		HandleNotFound(w, r)
		return
	}

	// ─── Step 3: Parse publicUrl domains (comma-separated) ──────────────
	parts := strings.Split(publicURL, ",")
	domains := make([]string, 0, len(parts))
	for _, d := range parts {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}

	// ─── Step 4: Fetch HLS playlist from storage server ─────────────────
	storageHost := storage.GetHost()
	if storageHost == "" {
		log.Printf("[Video] Storage has no host: storageId=%s", storage.ID)
		HandleNotFound(w, r)
		return
	}

	storageHLSURL := fmt.Sprintf("http://%s/%s/video.m3u8", storageHost, media.Slug)

	playlistContent, err := utils.FetchURLContent(ctx, storageHLSURL)
	if err != nil {
		log.Printf("[Video] Failed to fetch playlist from %s: %v", storageHLSURL, err)
		HandleNotFound(w, r)
		return
	}

	// ─── Step 5: Rewrite segment URLs to use publicUrl domains ──────────
	rewrittenPlaylist := utils.RewritePlaylist(playlistContent, domains, media.Slug)

	responseBody := []byte(rewrittenPlaylist)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(responseBody)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("CDN-Cache-Control", "max-age=31536000")

	w.Write(responseBody)
}

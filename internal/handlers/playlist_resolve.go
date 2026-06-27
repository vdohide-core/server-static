package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"server-static/internal/db/models"
	"server-static/internal/services"

	"go.mongodb.org/mongo-driver/bson"
)

// PlaylistContent holds resolved media URLs for a JW playlist feed.
type PlaylistContent struct {
	PosterURL    string
	PlaylistM3U8 string
	SpriteVttURL string
}

// PlaylistResolveResult is the output of playlist JSON resolution.
type PlaylistResolveResult struct {
	File    models.File
	Slug    string
	Content PlaylistContent
}

// PlaylistResolveError describes a failed playlist resolution.
type PlaylistResolveError struct {
	Status  int
	Message string
	State   string // queue | processing | error
}

func requestHost(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		return strings.TrimSpace(strings.Split(h, ",")[0])
	}
	return r.Host
}

func cdnHost(r *http.Request) string {
	host := requestHost(r)
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}

func isLocalRequest(r *http.Request) bool {
	host := strings.ToLower(cdnHost(r))
	return host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0"
}

func cfVisitorScheme(r *http.Request) string {
	raw := r.Header.Get("CF-Visitor")
	if raw == "" {
		return ""
	}
	var v struct {
		Scheme string `json:"scheme"`
	}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		raw = strings.ToLower(raw)
		if strings.Contains(raw, `"scheme":"https"`) || strings.Contains(raw, `"scheme": "https"`) {
			return "https"
		}
		if strings.Contains(raw, `"scheme":"http"`) || strings.Contains(raw, `"scheme": "http"`) {
			return "http"
		}
		return ""
	}
	return strings.ToLower(strings.TrimSpace(v.Scheme))
}

func forwardedProto(r *http.Request) string {
	p := r.Header.Get("X-Forwarded-Proto")
	if p == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(p, ",")[0]))
}

func requestProtocol(r *http.Request) string {
	if isLocalRequest(r) {
		return "http"
	}

	if scheme := cfVisitorScheme(r); scheme == "https" || scheme == "http" {
		return scheme
	}

	if p := forwardedProto(r); p == "https" || p == "http" {
		return p
	}

	if r.Header.Get("CF-Ray") != "" {
		return "https"
	}

	if r.Header.Get("X-Forwarded-Ssl") == "on" ||
		r.Header.Get("X-Forwarded-Scheme") == "https" ||
		r.Header.Get("X-Url-Scheme") == "https" {
		return "https"
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}

func (h *Handler) resolvePlaylist(r *http.Request, slug string) (*PlaylistResolveResult, *PlaylistResolveError) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var file models.File
	err := models.FileModel.Col().FindOne(ctx, bson.M{"slug": slug}).Decode(&file)
	if err != nil {
		return nil, &PlaylistResolveError{Status: http.StatusNotFound, Message: "not found"}
	}

	if file.IsTrashed() || file.IsDeleted() {
		return nil, &PlaylistResolveError{Status: http.StatusNotFound, Message: "not found"}
	}

	if file.SpaceID != nil && *file.SpaceID != "" {
		space := services.FindSpace(*file.SpaceID)
		if space != nil && space.Status == "error" {
			return nil, &PlaylistResolveError{Status: http.StatusNotFound, Message: "unavailable"}
		}
	}

	cursor, err := models.MediaModel.Col().Find(ctx, bson.M{
		"fileId":     file.ID,
		"type":       models.MediaTypeVideo,
		"resolution": bson.M{"$in": []string{"original", "1080", "720", "480", "360"}},
		"deletedAt":  bson.M{"$eq": nil},
	})
	if err != nil {
		return nil, &PlaylistResolveError{Status: http.StatusInternalServerError, Message: "error loading video"}
	}
	defer cursor.Close(ctx)

	medias := make(map[string]string)
	hasTranscoded := false
	for cursor.Next(ctx) {
		var media models.Media
		if err := cursor.Decode(&media); err != nil {
			continue
		}
		res := ""
		if media.Resolution != nil {
			res = *media.Resolution
		}
		if res != "" {
			medias[res] = media.Slug
			if res == "1080" || res == "720" || res == "480" || res == "360" {
				hasTranscoded = true
			}
		}
	}

	if hasTranscoded {
		delete(medias, "original")
	}

	if len(medias) == 0 {
		var vp models.VideoProcess
		vpErr := models.VideoProcessModel.Col().FindOne(ctx, bson.M{"fileId": file.ID}).Decode(&vp)

		state := "queue"
		if vpErr == nil && vp.Status != nil {
			switch *vp.Status {
			case "failed":
				state = "error"
			default:
				state = "processing"
			}
		}

		return nil, &PlaylistResolveError{
			Status:  http.StatusNotFound,
			Message: state,
			State:   state,
		}
	}

	reqProto := requestProtocol(r)
	host := cdnHost(r)
	if playlistHost := services.GetDomainPlaylist(requestHost(r)); playlistHost != "" {
		if i := strings.Index(playlistHost, ":"); i >= 0 {
			playlistHost = playlistHost[:i]
		}
		host = playlistHost
	}

	posterURL := ""
	var posterMedia models.Media
	err = models.MediaModel.Col().FindOne(ctx, bson.M{
		"fileId":     file.ID,
		"type":       models.MediaTypeImage,
		"resolution": "poster",
		"deletedAt":  bson.M{"$eq": nil},
	}).Decode(&posterMedia)
	if err == nil && posterMedia.StorageID != nil && *posterMedia.StorageID != "" {
		var storage models.Storage
		if sErr := models.StorageModel.Col().FindOne(ctx, bson.M{"_id": *posterMedia.StorageID}).Decode(&storage); sErr == nil {
			if storage.PublicURL != nil && *storage.PublicURL != "" {
				posterURL = strings.TrimRight(*storage.PublicURL, "/") + "/" + posterMedia.Slug + "/poster.jpg"
			}
		}
	}

	playlistM3U8 := reqProto + "://" + host + "/" + slug + "/playlist.m3u8"

	cdn := cdnHost(r)

	if posterURL == "" {
		thumbTime := 0
		if file.Metadata != nil && file.Metadata.Duration != nil {
			thumbTime = int(*file.Metadata.Duration / 2)
		}
		posterURL = reqProto + "://" + cdn + "/thumb/" + slug + "/" + fmt.Sprintf("%d", thumbTime) + ".jpg"
	}

	spriteVttURL := ""
	var spriteMedia models.Media
	if err := models.MediaModel.Col().FindOne(ctx, bson.M{
		"fileId":    file.ID,
		"type":      models.MediaTypeThumbnail,
		"fileName":  "sprite.vtt",
		"deletedAt": bson.M{"$eq": nil},
	}).Decode(&spriteMedia); err == nil {
		spriteVttURL = reqProto + "://" + cdn + "/" + slug + "/sprite/sprite.vtt"
	}

	return &PlaylistResolveResult{
		File: file,
		Slug: slug,
		Content: PlaylistContent{
			PosterURL:    posterURL,
			PlaylistM3U8: playlistM3U8,
			SpriteVttURL: spriteVttURL,
		},
	}, nil
}

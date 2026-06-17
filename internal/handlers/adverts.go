package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"server-static/internal/db/models"
	"server-static/internal/services"
)

// ImageAdvertsResponse is returned by GET /image/{slug}.json
type ImageAdvertsResponse struct {
	Enabled bool           `json:"enabled"`
	List    []ImageAdEntry `json:"list"`
}

// ImageAdEntry is a single image advert for the player overlay.
type ImageAdEntry struct {
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name,omitempty"`
	ImageURL   string   `json:"imageUrl"`
	WebsiteURL string   `json:"websiteUrl,omitempty"`
	ShowOn     []string `json:"showOn,omitempty"`
}

// ScriptAdvertsResponse is returned by GET /script/{slug}.json
type ScriptAdvertsResponse struct {
	Enabled bool            `json:"enabled"`
	List    []ScriptAdEntry `json:"list"`
}

// ScriptAdEntry is a single JavaScript advert.
type ScriptAdEntry struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Script string `json:"script"`
}

// ImageAdverts handles GET /image/{slug}.json
func (h *Handler) ImageAdverts(w http.ResponseWriter, r *http.Request) {
	slug := advertSlugFromPath(r.URL.Path, "/image/")
	if slug == "" {
		writeAdvertJSON(w, ImageAdvertsResponse{})
		return
	}

	entry := services.FindDomainBySlug(slug)
	if entry == nil {
		writeAdvertJSON(w, ImageAdvertsResponse{})
		return
	}

	writeAdvertJSON(w, buildImageAdverts(entry.Adverts))
}

// ScriptAdverts handles GET /script/{slug}.json
func (h *Handler) ScriptAdverts(w http.ResponseWriter, r *http.Request) {
	slug := advertSlugFromPath(r.URL.Path, "/script/")
	if slug == "" {
		writeAdvertJSON(w, ScriptAdvertsResponse{})
		return
	}

	entry := services.FindDomainBySlug(slug)
	if entry == nil {
		writeAdvertJSON(w, ScriptAdvertsResponse{})
		return
	}

	writeAdvertJSON(w, buildScriptAdverts(entry.Adverts))
}

func advertSlugFromPath(path, prefix string) string {
	slug := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(slug, ".json")
}

func buildImageAdverts(adverts *models.DomainAdverts) ImageAdvertsResponse {
	resp := ImageAdvertsResponse{List: []ImageAdEntry{}}
	if adverts == nil || !adverts.Image.Enabled {
		return resp
	}

	resp.Enabled = true
	for _, item := range adverts.Image.List {
		if !item.Enabled || item.ImageURL == nil || *item.ImageURL == "" {
			continue
		}
		entry := ImageAdEntry{
			ID:       item.ID,
			Name:     item.Name,
			ImageURL: *item.ImageURL,
			ShowOn:   item.ShowOn,
		}
		if item.WebsiteURL != nil {
			entry.WebsiteURL = *item.WebsiteURL
		}
		resp.List = append(resp.List, entry)
	}
	return resp
}

func buildScriptAdverts(adverts *models.DomainAdverts) ScriptAdvertsResponse {
	resp := ScriptAdvertsResponse{List: []ScriptAdEntry{}}
	if adverts == nil || !adverts.Script.Enabled {
		return resp
	}

	resp.Enabled = true
	for _, item := range adverts.Script.List {
		if !item.Enabled || item.Script == nil || *item.Script == "" {
			continue
		}
		resp.List = append(resp.List, ScriptAdEntry{
			ID:     item.ID,
			Name:   item.Name,
			Script: *item.Script,
		})
	}
	return resp
}

func writeAdvertJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = json.NewEncoder(w).Encode(v)
}

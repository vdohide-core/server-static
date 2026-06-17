package services

import (
	"log"
	"path/filepath"
	"sync"

	"server-static/internal/db/models"
)

// ─── File Paths ───────────────────────────────────────────────────────

// settingFilePath returns the absolute path to conf/setting.json
func settingFilePath() string {
	exe, err := executableDir()
	if err != nil {
		log.Printf("⚠️ Cannot get executable path: %v", err)
		return filepath.Join("conf", "setting.json")
	}
	return filepath.Join(exe, "conf", "setting.json")
}

// advertsFilePath returns the path to conf/adverts.json
func advertsFilePath() string {
	exe, err := executableDir()
	if err != nil {
		return filepath.Join("conf", "adverts.json")
	}
	return filepath.Join(exe, "conf", "adverts.json")
}

// spacesFilePath returns the path to conf/spaces.json
func spacesFilePath() string {
	exe, err := executableDir()
	if err != nil {
		return filepath.Join("conf", "spaces.json")
	}
	return filepath.Join(exe, "conf", "spaces.json")
}

// ─── Domain Cache ─────────────────────────────────────────────────────

var (
	domainCache   map[string]*models.DomainVastEntry // slug → entry
	domainCacheMu sync.RWMutex
)

// LoadDomains loads domain VAST entries into the in-memory cache.
func LoadDomains(domains []models.DomainVastEntry) {
	cache := make(map[string]*models.DomainVastEntry, len(domains))
	for i := range domains {
		if domains[i].Slug == "" {
			continue
		}
		cache[domains[i].Slug] = &domains[i]
	}

	domainCacheMu.Lock()
	domainCache = cache
	domainCacheMu.Unlock()

	log.Printf("📋 Loaded %d advert entries → conf/adverts.json", len(cache))
}

// FindDomainBySlug looks up a domain by slug from the in-memory cache.
func FindDomainBySlug(slug string) *models.DomainVastEntry {
	if slug == "" {
		return nil
	}

	domainCacheMu.RLock()
	defer domainCacheMu.RUnlock()

	return domainCache[slug]
}

// ─── Space Cache ──────────────────────────────────────────────────────

var (
	spaceCache   map[string]*models.Workspace // spaceId → Workspace
	spaceCacheMu sync.RWMutex
)

// FindSpace looks up a space (Workspace) by its ID from the in-memory cache.
// Returns nil if not found.
func FindSpace(spaceID string) *models.Workspace {
	if spaceID == "" {
		return nil
	}

	spaceCacheMu.RLock()
	defer spaceCacheMu.RUnlock()

	return spaceCache[spaceID]
}

// LoadSpaces loads Workspaces into the in-memory cache
func LoadSpaces(spaces []models.Workspace) {
	cache := make(map[string]*models.Workspace, len(spaces))
	for i := range spaces {
		cache[spaces[i].ID] = &spaces[i]
	}

	spaceCacheMu.Lock()
	spaceCache = cache
	spaceCacheMu.Unlock()

	log.Printf("📋 Loaded %d spaces → conf/spaces.json", len(cache))
}

// GetSpacePlan returns the plan for a space, nil if not found or no plan.
func GetSpacePlan(spaceID string) *models.WorkspacePlan {
	space := FindSpace(spaceID)
	if space == nil {
		return nil
	}
	return space.Plan
}

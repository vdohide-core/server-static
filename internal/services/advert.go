package services

import "server-static/internal/db/models"

// BuildAdvertFeed builds /advert/{adSlug}.json (script + image + video).
func BuildAdvertFeed(adSlug string) *models.DomainAdverts {
	if entry := FindDomainBySlug(adSlug); entry != nil && entry.Adverts != nil {
		return entry.Adverts
	}
	empty := models.DomainAdverts{}
	return &empty
}

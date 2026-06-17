package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"server-static/internal/db/models"
	"server-static/internal/services"
)

// Vast handles GET /vast/{domainSlug}.xml — generates VAST 3.0 XML from domain adverts.
func (h *Handler) Vast(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/vast/")
	slug := strings.TrimSuffix(path, ".xml")

	if slug == "" {
		writeEmptyVast(w)
		return
	}

	domain := services.FindDomainBySlug(slug)
	if domain == nil {
		writeEmptyVast(w)
		return
	}

	buildVideoVast(w, domain.Adverts)
}

func buildVideoVast(w http.ResponseWriter, adverts *models.DomainAdverts) {
	if adverts == nil || !adverts.Video.Enabled || len(adverts.Video.List) == 0 {
		writeEmptyVast(w)
		return
	}

	buildAdvertsVast(w, adverts.Video.List)
}

func buildAdvertsVast(w http.ResponseWriter, items []models.AdsContent) {
	var ads strings.Builder
	hasActive := false
	sequence := 0

	for _, item := range items {
		if !item.Enabled || item.Mp4URL == nil || *item.Mp4URL == "" {
			continue
		}
		hasActive = true
		sequence++

		adID := item.ID
		if adID == "" {
			adID = fmt.Sprintf("ad-%d", sequence)
		}
		skipSeconds := 5
		if item.SkipSeconds != nil {
			skipSeconds = *item.SkipSeconds
		}
		skipOffset := fmt.Sprintf("00:00:%02d", skipSeconds)

		websiteURL := ""
		if item.WebsiteURL != nil {
			websiteURL = *item.WebsiteURL
		}

		ads.WriteString(vastAdXML(sequence, adID, item.Name, skipOffset, websiteURL, *item.Mp4URL))
	}

	if !hasActive {
		writeEmptyVast(w)
		return
	}

	writeVast(w, ads.String())
}

func vastAdXML(sequence int, adID, name, skipOffset, websiteURL, mp4URL string) string {
	return fmt.Sprintf(`
    <Ad id="%s" sequence="%d">
      <InLine>
        <AdSystem version="2.0">JW Player</AdSystem>
        <AdTitle>%s</AdTitle>
        <Creatives>
          <Creative sequence="1">
            <Linear skipoffset="%s">
              <VideoClicks>
                <ClickThrough>%s</ClickThrough>
              </VideoClicks>
              <MediaFiles>
                <MediaFile
                  id="%s"
                  delivery="progressive"
                  type="video/mp4"
                  bitrate="400"
                  width="640"
                  height="360"
                >%s</MediaFile>
              </MediaFiles>
            </Linear>
          </Creative>
          <Creative> </Creative>
        </Creatives>
      </InLine>
    </Ad>`, html.EscapeString(adID), sequence, html.EscapeString(name), skipOffset, html.EscapeString(websiteURL), html.EscapeString(adID), html.EscapeString(mp4URL))
}

func writeVast(w http.ResponseWriter, adsXML string) {
	vast := fmt.Sprintf(`<?xml version="1.0"?>
<VAST
  version="3.0"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:noNamespaceSchemaLocation="vast3_draft.xsd"
>%s
</VAST>`, adsXML)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(vast))
}

func writeEmptyVast(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(`<?xml version="1.0"?>
<VAST version="3.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="vast3_draft.xsd">
</VAST>`))
}

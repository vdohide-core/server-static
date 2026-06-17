package services

import "encoding/json"
// PlayerSettings holds the global default player configuration.
// These are hardcoded defaults — no longer stored in the database.
// Per-domain overrides come from CustomDomain.Player.
type PlayerSettings struct {
	LogoImageURL    string  `json:"logoImageUrl"`
	LogoWebsiteURL  string  `json:"logoWebsiteUrl"`
	LogoPosition    string  `json:"logoPosition"`
	PosterURL       string  `json:"posterUrl"`
	BaseColor       string  `json:"baseColor"`
	DisplayTitle    bool    `json:"displayTitle"`
	AutoPlay        bool    `json:"autoPlay"`
	MuteSound       bool    `json:"muteSound"`
	RepeatVideo     bool    `json:"repeatVideo"`
	ContinuePlay    bool    `json:"continuePlay"`
	ContinuePlayArk bool    `json:"continuePlayArk"`
	Sharing         bool    `json:"sharing"`
	Captions        bool    `json:"captions"`
	PlaybackRate    bool    `json:"playbackRate"`
	Keyboard        bool    `json:"keyboard"`
	Download        bool    `json:"download"`
	PIP             bool    `json:"pip"`
	ShowPreviewTime bool    `json:"showPreviewTime"`
	FastForward     bool    `json:"fastForward"`
	Rewind          bool    `json:"rewind"`
	SeekStep        int     `json:"seekStep"`
}

// AdvertImageConfig holds image ad overlay config passed to the player JS
type AdvertImageConfig struct {
	ImageUrl   string   `json:"imageUrl"`
	WebsiteUrl string   `json:"websiteUrl"`
	ShowOn     []string `json:"showOn"`
}

// PlayerConfig holds all config passed to the player template (rendered as JS)
type PlayerConfig struct {
	Title           string             `json:"title"`
	Poster          string             `json:"poster"`
	PlaylistURL     string             `json:"playlistUrl"`
	Medias          map[string]string  `json:"medias"`
	SpriteVttUrl    string             `json:"spriteVttUrl,omitempty"`
	BaseColor       string             `json:"baseColor"`
	DisplayTitle    bool               `json:"displayTitle"`
	Autostart       bool               `json:"autostart"`
	Mute            bool               `json:"mute"`
	Repeat          bool               `json:"repeat"`
	ContinuePlayback    bool           `json:"continuePlayBack"`
	ContinuePlaybackArk bool           `json:"continuePlayBackArk"`
	Sharing         bool               `json:"sharing"`
	Captions        bool               `json:"captions"`
	PlaybackRate    bool               `json:"playbackRate"`
	Keyboard        bool               `json:"keyboard"`
	Download        bool               `json:"download"`
	Pip             bool               `json:"pip"`
	ShowPreviewTime bool               `json:"showPreviewTime"`
	FastForward     bool               `json:"fastForward"`
	Rewind          bool               `json:"rewind"`
	SeekStep        int                `json:"seekStep"`
	WatermarkEnabled  bool             `json:"watermarkEnabled"`
	WatermarkUrl      string           `json:"watermarkUrl"`
	WatermarkWebUrl   string           `json:"watermarkWebUrl"`
	WatermarkPosition string           `json:"watermarkPosition"`
	WatermarkOpacity  float64          `json:"watermarkOpacity"`
	VastURL         string             `json:"vastUrl,omitempty"`
	AdvertImages    []AdvertImageConfig `json:"advertImages,omitempty"`
}

// GetPlayerSettings returns the hardcoded global default player settings.
// Per-domain overrides are applied separately via CustomDomain.Player.
func GetPlayerSettings() PlayerSettings {
	return PlayerSettings{
		LogoImageURL:    "",
		LogoWebsiteURL:  "",
		LogoPosition:    "",
		PosterURL:       "",
		BaseColor:       "#ff8800",
		DisplayTitle:    false,
		AutoPlay:        false,
		MuteSound:       false,
		RepeatVideo:     false,
		ContinuePlay:    true,
		ContinuePlayArk: false,
		Sharing:         false,
		Captions:        false,
		PlaybackRate:    false,
		Keyboard:        false,
		Download:        true,
		PIP:             true,
		ShowPreviewTime: true,
		FastForward:     true,
		Rewind:          true,
		SeekStep:        10,
	}
}

// IsMaintenanceMode reads player_maintenance from setting.json.
// Returns false by default if the key is missing or unreadable.
func IsMaintenanceMode() bool {
	settings, err := ReadSettingFile()
	if err != nil {
		return false
	}
	raw, exists := settings["player_maintenance"]
	if !exists {
		return false
	}
	var val bool
	if err := json.Unmarshal(raw, &val); err != nil {
		return false
	}
	return val
}

// BuildPlayerConfig creates a PlayerConfig from global default settings.
func BuildPlayerConfig(
	title string,
	posterURL string,
	playlistURL string,
	medias map[string]string,
	settings PlayerSettings,
) PlayerConfig {
	return PlayerConfig{
		Title:               title,
		Poster:              posterURL,
		PlaylistURL:         playlistURL,
		Medias:              medias,
		BaseColor:           settings.BaseColor,
		DisplayTitle:        settings.DisplayTitle,
		Autostart:           settings.AutoPlay,
		Mute:                settings.MuteSound,
		Repeat:              settings.RepeatVideo,
		ContinuePlayback:    settings.ContinuePlay,
		ContinuePlaybackArk: settings.ContinuePlayArk,
		Sharing:             settings.Sharing,
		Captions:            settings.Captions,
		PlaybackRate:        settings.PlaybackRate,
		Keyboard:            settings.Keyboard,
		Download:            settings.Download,
		Pip:                 settings.PIP,
		ShowPreviewTime:     settings.ShowPreviewTime,
		FastForward:         settings.FastForward,
		Rewind:              settings.Rewind,
		SeekStep:            settings.SeekStep,
	}
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ConnectResources are administrator-managed tutorial videos and installer
// downloads for the connection center. Files live in external storage; only
// their HTTPS addresses are stored here. Signed-in users receive the enabled
// entries through an authenticated endpoint, never through public settings.
type ConnectResources struct {
	Videos    []ConnectVideo    `json:"videos"`
	Downloads []ConnectDownload `json:"downloads"`
}

type ConnectVideo struct {
	ID        string `json:"id"`
	OS        string `json:"os"`
	Title     string `json:"title"`
	TitleEN   string `json:"title_en"`
	URL       string `json:"url"`
	PosterURL string `json:"poster_url"`
	Enabled   bool   `json:"enabled"`
}

type ConnectDownload struct {
	ID      string `json:"id"`
	OS      string `json:"os"`
	Name    string `json:"name"`
	NameEN  string `json:"name_en"`
	Version string `json:"version"`
	Note    string `json:"note"`
	NoteEN  string `json:"note_en"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Enabled bool   `json:"enabled"`
}

const (
	maxConnectVideos    = 12
	maxConnectDownloads = 30
)

var (
	// "all" entries (one video or installer for every system) appear under each system tab.
	connectResourceOS = map[string]bool{"macos": true, "windows": true, "linux": true, "all": true}
	sha256Hex         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func ParseConnectResources(raw string) ConnectResources {
	var resources ConnectResources
	_ = json.Unmarshal([]byte(raw), &resources)
	if resources.Videos == nil {
		resources.Videos = []ConnectVideo{}
	}
	if resources.Downloads == nil {
		resources.Downloads = []ConnectDownload{}
	}
	return resources
}

func ValidateConnectResources(resources ConnectResources) error {
	if len(resources.Videos) > maxConnectVideos {
		return fmt.Errorf("at most %d tutorial videos are supported", maxConnectVideos)
	}
	if len(resources.Downloads) > maxConnectDownloads {
		return fmt.Errorf("at most %d downloads are supported", maxConnectDownloads)
	}
	ids := map[string]bool{}
	for _, video := range resources.Videos {
		if err := validateConnectID(video.ID, ids); err != nil {
			return err
		}
		if !connectResourceOS[video.OS] {
			return fmt.Errorf("invalid tutorial video system")
		}
		if utf8.RuneCountInString(video.Title) > 80 || utf8.RuneCountInString(video.TitleEN) > 80 {
			return fmt.Errorf("tutorial video titles are too long")
		}
		if video.Enabled && strings.TrimSpace(video.URL) == "" {
			return fmt.Errorf("enabled tutorial videos require a URL")
		}
		if err := validateConnectURL(video.URL); err != nil {
			return err
		}
		if err := validateConnectURL(video.PosterURL); err != nil {
			return err
		}
	}
	for _, download := range resources.Downloads {
		if err := validateConnectID(download.ID, ids); err != nil {
			return err
		}
		if !connectResourceOS[download.OS] {
			return fmt.Errorf("invalid download system")
		}
		if utf8.RuneCountInString(download.Name) > 80 || utf8.RuneCountInString(download.NameEN) > 80 || utf8.RuneCountInString(download.Version) > 40 ||
			utf8.RuneCountInString(download.Note) > 200 || utf8.RuneCountInString(download.NoteEN) > 200 {
			return fmt.Errorf("download text is too long")
		}
		if download.Enabled && (strings.TrimSpace(download.Name) == "" || strings.TrimSpace(download.URL) == "") {
			return fmt.Errorf("enabled downloads require a name and URL")
		}
		if err := validateConnectURL(download.URL); err != nil {
			return err
		}
		if download.SHA256 != "" && !sha256Hex.MatchString(download.SHA256) {
			return fmt.Errorf("SHA-256 must be 64 lowercase hexadecimal characters")
		}
	}
	return nil
}

func validateConnectID(id string, seen map[string]bool) error {
	if id == "" || len(id) > 80 || seen[id] {
		return fmt.Errorf("connect resource IDs must be present and unique")
	}
	seen[id] = true
	return nil
}

// HTTPS only: the connection center is served over HTTPS and mixed media is blocked.
func validateConnectURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("connect resource links must be absolute HTTPS URLs without credentials")
	}
	return nil
}

// UserConnectResources drops disabled and invalid drafts before they reach users.
func UserConnectResources(raw string) ConnectResources {
	parsed := ParseConnectResources(raw)
	result := ConnectResources{Videos: []ConnectVideo{}, Downloads: []ConnectDownload{}}
	for _, video := range parsed.Videos {
		if video.Enabled && ValidateConnectResources(ConnectResources{Videos: []ConnectVideo{video}}) == nil {
			result.Videos = append(result.Videos, video)
		}
	}
	for _, download := range parsed.Downloads {
		if download.Enabled && ValidateConnectResources(ConnectResources{Downloads: []ConnectDownload{download}}) == nil {
			result.Downloads = append(result.Downloads, download)
		}
	}
	return result
}

func connectResourcesAvailable(raw string) bool {
	resources := UserConnectResources(raw)
	return len(resources.Videos) > 0 || len(resources.Downloads) > 0
}

func (s *SettingService) GetUserConnectResources(ctx context.Context) (ConnectResources, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyConnectResources)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return UserConnectResources(""), nil
		}
		return ConnectResources{}, fmt.Errorf("get connect resources: %w", err)
	}
	return UserConnectResources(raw), nil
}

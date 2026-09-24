//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type connectResourcesRepoStub struct {
	SettingRepository
	value string
	err   error
}

func (s *connectResourcesRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key != SettingKeyConnectResources {
		panic("unexpected key " + key)
	}
	return s.value, s.err
}

func validConnectResources() ConnectResources {
	return ConnectResources{
		Videos: []ConnectVideo{{ID: "mac", OS: "macos", Title: "Mac 教程", URL: "https://media.example.com/mac.mp4", Enabled: true}},
		Downloads: []ConnectDownload{{
			ID: "ccs-mac", OS: "macos", Name: "CC Switch", Version: "3.20.2",
			URL: "https://media.example.com/ccs.dmg", SHA256: strings.Repeat("a", 64), Enabled: true,
		}},
	}
}

func TestConnectResourcesValidation(t *testing.T) {
	require.NoError(t, ValidateConnectResources(validConnectResources()))
	require.NoError(t, ValidateConnectResources(ParseConnectResources("")))

	cases := map[string]func(*ConnectResources){
		"http video":          func(r *ConnectResources) { r.Videos[0].URL = "http://media.example.com/mac.mp4" },
		"javascript poster":   func(r *ConnectResources) { r.Videos[0].PosterURL = "javascript:alert(1)" },
		"relative download":   func(r *ConnectResources) { r.Downloads[0].URL = "/files/ccs.dmg" },
		"credentials":         func(r *ConnectResources) { r.Downloads[0].URL = "https://u:p@media.example.com/a.dmg" },
		"unknown video os":    func(r *ConnectResources) { r.Videos[0].OS = "android" },
		"unknown download os": func(r *ConnectResources) { r.Downloads[0].OS = "ios" },
		"uppercase sha":       func(r *ConnectResources) { r.Downloads[0].SHA256 = strings.Repeat("A", 64) },
		"short sha":           func(r *ConnectResources) { r.Downloads[0].SHA256 = "abc" },
		"shared id":           func(r *ConnectResources) { r.Downloads[0].ID = "mac" },
		"enabled no url":      func(r *ConnectResources) { r.Videos[0].URL = "" },
		"enabled no name":     func(r *ConnectResources) { r.Downloads[0].Name = " " },
		"long title":          func(r *ConnectResources) { r.Videos[0].Title = strings.Repeat("长", 81) },
		"too many videos": func(r *ConnectResources) {
			for i := 0; i < maxConnectVideos; i++ {
				r.Videos = append(r.Videos, ConnectVideo{ID: "v" + strings.Repeat("x", i+1), OS: "windows"})
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			resources := validConnectResources()
			mutate(&resources)
			require.Error(t, ValidateConnectResources(resources))
		})
	}

	shared := validConnectResources()
	shared.Videos[0].OS = "all"
	require.NoError(t, ValidateConnectResources(shared), "one video may serve every system")

	draft := validConnectResources()
	draft.Videos[0].Enabled = false
	draft.Videos[0].URL = ""
	require.NoError(t, ValidateConnectResources(draft), "disabled drafts may be incomplete")
}

func TestUserConnectResourcesDropsDraftsAndInvalidEntries(t *testing.T) {
	resources := validConnectResources()
	resources.Videos = append(resources.Videos,
		ConnectVideo{ID: "win-draft", OS: "windows", URL: "https://media.example.com/win.mp4"},
		ConnectVideo{ID: "win-bad", OS: "windows", URL: "http://media.example.com/win.mp4", Enabled: true},
	)
	resources.Downloads = append(resources.Downloads, ConnectDownload{ID: "off", OS: "all", Name: "Off", URL: "https://media.example.com/off.zip"})
	raw, err := json.Marshal(resources)
	require.NoError(t, err)

	visible := UserConnectResources(string(raw))
	require.Len(t, visible.Videos, 1)
	require.Equal(t, "mac", visible.Videos[0].ID)
	require.Len(t, visible.Downloads, 1)
	require.Equal(t, "ccs-mac", visible.Downloads[0].ID)

	empty := UserConnectResources("not json")
	require.NotNil(t, empty.Videos)
	require.NotNil(t, empty.Downloads)
}

func TestGetUserConnectResources(t *testing.T) {
	ctx := context.Background()
	raw, err := json.Marshal(validConnectResources())
	require.NoError(t, err)

	svc := NewSettingService(&connectResourcesRepoStub{value: string(raw)}, &config.Config{})
	got, err := svc.GetUserConnectResources(ctx)
	require.NoError(t, err)
	require.Len(t, got.Videos, 1)

	svc = NewSettingService(&connectResourcesRepoStub{err: ErrSettingNotFound}, &config.Config{})
	got, err = svc.GetUserConnectResources(ctx)
	require.NoError(t, err)
	require.Empty(t, got.Videos)
	require.Empty(t, got.Downloads)

	svc = NewSettingService(&connectResourcesRepoStub{err: errors.New("db down")}, &config.Config{})
	_, err = svc.GetUserConnectResources(ctx)
	require.Error(t, err)
}

func TestConnectResourcesSettingsRoundTrip(t *testing.T) {
	raw, err := json.Marshal(validConnectResources())
	require.NoError(t, err)
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	require.NoError(t, svc.UpdateSettings(context.Background(), &SystemSettings{ConnectResources: string(raw)}))
	require.Equal(t, string(raw), repo.updates[SettingKeyConnectResources])
}

// Anonymous visitors only learn whether resources exist, never their addresses.
func TestPublicSettingsExposeOnlyConnectResourceAvailability(t *testing.T) {
	ctx := context.Background()
	raw, err := json.Marshal(validConnectResources())
	require.NoError(t, err)

	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{SettingKeyConnectResources: string(raw)}}, &config.Config{})
	settings, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, settings.ConnectResourcesAvailable)
	injected, err := svc.GetPublicSettingsForInjection(ctx)
	require.NoError(t, err)
	for _, value := range []any{settings, injected} {
		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "media.example.com")
	}
	require.Contains(t, connectJSON(t, injected), `"connect_resources_available":true`)

	drafts := validConnectResources()
	drafts.Videos[0].Enabled = false
	drafts.Downloads[0].Enabled = false
	raw, err = json.Marshal(drafts)
	require.NoError(t, err)
	svc = NewSettingService(&settingPublicRepoStub{values: map[string]string{SettingKeyConnectResources: string(raw)}}, &config.Config{})
	settings, err = svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.False(t, settings.ConnectResourcesAvailable)
}

func connectJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

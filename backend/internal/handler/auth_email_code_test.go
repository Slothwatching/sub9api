package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerEmailLoginCache struct {
	service.EmailCache
	valid bool
}

func (c *handlerEmailLoginCache) ReserveEmailLoginCode(context.Context, string, string) (bool, error) {
	return true, nil
}
func (c *handlerEmailLoginCache) ConsumeEmailLoginCode(context.Context, string, string) (bool, error) {
	v := c.valid
	c.valid = false
	return v, nil
}

func TestEmailLoginRequiresTOTPAndPreventsReplay(t *testing.T) {
	cache := &handlerEmailLoginCache{valid: true}
	totpCache := &oauthPendingFlowTotpCacheStub{}
	h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{
		emailCache: cache, settingValues: map[string]string{service.SettingKeyEmailCodeLoginEnabled: "true", service.SettingKeyTotpEnabled: "true"}, totpCache: totpCache,
	})
	user, err := client.User.Create().SetEmail("user@example.com").SetUsername("user").SetPasswordHash("hash").SetRole(service.RoleUser).SetStatus(service.StatusActive).SetTotpEnabled(true).Save(context.Background())
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-code/login", strings.NewReader(`{"email":"user@example.com","code":"123456"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.LoginWithEmailCode(c)
		if i == 0 {
			require.Equal(t, http.StatusOK, w.Code)
			data := decodeJSONResponseData(t, w)
			require.Equal(t, true, data["requires_2fa"])
			require.NotContains(t, data, "access_token")
			require.NotContains(t, data, "refresh_token")
			token, ok := data["temp_token"].(string)
			require.True(t, ok)
			session, err := totpCache.GetLoginSession(context.Background(), token)
			require.NoError(t, err)
			require.Equal(t, user.ID, session.UserID)
		} else {
			require.NotEqual(t, http.StatusOK, w.Code)
			require.NotContains(t, w.Body.String(), "access_token")
		}
	}
}

func TestEmailLoginDisabledAndInvalidInput(t *testing.T) {
	h, _ := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{})
	for _, body := range []string{`{"email":"user@example.com","code":"123456"}`, `{"email":"user@example.com","code":"abc123"}`, `{"email":"user@example.com","code":"123"}`} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-code/login", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.LoginWithEmailCode(c)
		require.NotEqual(t, http.StatusOK, w.Code)
		require.NotContains(t, w.Body.String(), "access_token")
	}
}

func TestEmailLoginAccountAndBackendGates(t *testing.T) {
	for _, kind := range []string{"active", "disabled", "backend-user"} {
		t.Run(kind, func(t *testing.T) {
			cache := &handlerEmailLoginCache{valid: true}
			h, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{emailCache: cache,
				settingValues: map[string]string{service.SettingKeyEmailCodeLoginEnabled: "true", service.SettingKeyBackendModeEnabled: boolSettingValue(kind == "backend-user")}})
			status := service.StatusActive
			if kind == "disabled" {
				status = "disabled"
			}
			_, err := client.User.Create().SetEmail("gate@example.com").SetUsername("gate").SetPasswordHash("hash").SetRole(service.RoleUser).SetStatus(status).Save(context.Background())
			require.NoError(t, err)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-code/login", strings.NewReader(`{"email":"gate@example.com","code":"123456"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.LoginWithEmailCode(c)
			if kind == "active" {
				require.Equal(t, http.StatusOK, w.Code)
				data := decodeJSONResponseData(t, w)
				require.NotEmpty(t, data["access_token"])
				require.NotEmpty(t, data["refresh_token"])
			} else {
				require.NotEqual(t, http.StatusOK, w.Code)
				require.NotContains(t, w.Body.String(), "access_token")
			}
		})
	}
}

func TestEmailLoginCaptchaFailureDoesNotConsumeCode(t *testing.T) {
	cache := &handlerEmailLoginCache{valid: true}
	h, _ := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{emailCache: cache,
		settingValues: map[string]string{service.SettingKeyEmailCodeLoginEnabled: "true", service.SettingKeyTurnstileEnabled: "true"}})
	for _, send := range []bool{false, true} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/email-code/login", strings.NewReader(`{"email":"user@example.com","code":"123456"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		if send {
			h.SendEmailLoginCode(c)
		} else {
			h.LoginWithEmailCode(c)
		}
		require.NotEqual(t, http.StatusOK, w.Code)
		require.True(t, cache.valid)
	}
}

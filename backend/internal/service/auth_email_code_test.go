//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type emailLoginUserRepo struct {
	UserRepository
	user *User
	err  error
}

func (r *emailLoginUserRepo) GetByEmail(context.Context, string) (*User, error) { return r.user, r.err }

type emailLoginCacheStub struct {
	EmailCache
	digest string
	err    error
}

func (c *emailLoginCacheStub) ReserveEmailLoginCode(_ context.Context, _, digest string) (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	if c.digest != "" {
		return false, nil
	}
	c.digest = digest
	return true, nil
}
func (c *emailLoginCacheStub) ConsumeEmailLoginCode(_ context.Context, _, digest string) (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	ok := c.digest != "" && c.digest == digest
	if ok {
		c.digest = ""
	}
	return ok, nil
}
func emailLoginFixture(t *testing.T) (*AuthService, *emailLoginUserRepo, *emailLoginCacheStub, *settingRepoStub) {
	backendModeCache.Store((*cachedBackendMode)(nil))
	t.Cleanup(func() { backendModeCache.Store((*cachedBackendMode)(nil)) })
	settings := &settingRepoStub{values: map[string]string{SettingKeyEmailCodeLoginEnabled: "true", SettingKeySMTPHost: "smtp.example.com"}}
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret"}}
	cache := &emailLoginCacheStub{}
	repo := &emailLoginUserRepo{user: &User{ID: 1, Email: "user@example.com", Status: StatusActive, PasswordHash: "original"}}
	s := &AuthService{cfg: cfg, userRepo: repo, settingService: NewSettingService(settings, cfg), emailService: NewEmailService(settings, cache), emailQueueService: &EmailQueueService{taskChan: make(chan EmailTask, 10)}}
	return s, repo, cache, settings
}
func TestEmailLoginUnknownAndInactiveUseGenericSend(t *testing.T) {
	for _, kind := range []string{"active", "missing", "inactive", "backend-user", "reserved"} {
		t.Run(kind, func(t *testing.T) {
			s, repo, cache, settings := emailLoginFixture(t)
			email := "user@example.com"
			switch kind {
			case "missing":
				repo.user = nil
				repo.err = ErrUserNotFound
			case "inactive":
				repo.user.Status = "disabled"
			case "backend-user":
				settings.values[SettingKeyBackendModeEnabled] = "true"
			case "reserved":
				email = "account@dingtalk-connect.invalid"
			}
			require.NoError(t, s.SendEmailLoginCode(context.Background(), email, "zh"))
			task := <-s.emailQueueService.taskChan
			require.Equal(t, kind != "active", task.SkipDelivery)
			require.Len(t, task.LoginCode, 6)
			require.Len(t, cache.digest, 64)
		})
	}
}
func TestEmailLoginUsesCurrentPasswordAndSingleUse(t *testing.T) {
	s, repo, _, _ := emailLoginFixture(t)
	ctx := context.Background()
	require.NoError(t, s.SendEmailLoginCode(ctx, "user@example.com", "en"))
	task := <-s.emailQueueService.taskChan
	_, err := s.LoginWithEmailCode(ctx, "user@example.com", "0000000")
	require.ErrorIs(t, err, ErrInvalidVerifyCode)
	user, err := s.LoginWithEmailCode(ctx, "User@Example.com", task.LoginCode)
	require.NoError(t, err)
	require.Equal(t, repo.user, user)
	_, err = s.LoginWithEmailCode(ctx, "user@example.com", task.LoginCode)
	require.ErrorIs(t, err, ErrInvalidVerifyCode)
	require.NoError(t, s.SendEmailLoginCode(ctx, "user@example.com", "en"))
	task = <-s.emailQueueService.taskChan
	repo.user.PasswordHash = "changed"
	_, err = s.LoginWithEmailCode(ctx, "user@example.com", task.LoginCode)
	require.ErrorIs(t, err, ErrInvalidVerifyCode)
}
func TestEmailLoginDisabledAndDependenciesFailClosed(t *testing.T) {
	s, _, cache, settings := emailLoginFixture(t)
	ctx := context.Background()
	settings.values[SettingKeyEmailCodeLoginEnabled] = "false"
	require.ErrorIs(t, s.SendEmailLoginCode(ctx, "u@example.com", ""), ErrEmailCodeLoginDisabled)
	_, err := s.LoginWithEmailCode(ctx, "u@example.com", "123456")
	require.ErrorIs(t, err, ErrEmailCodeLoginDisabled)
	settings.values[SettingKeyEmailCodeLoginEnabled] = "true"
	cache.err = errors.New("redis down")
	require.ErrorIs(t, s.SendEmailLoginCode(ctx, "u@example.com", ""), ErrServiceUnavailable)
	_, err = s.LoginWithEmailCode(ctx, "u@example.com", "123456")
	require.ErrorIs(t, err, ErrServiceUnavailable)
	cache.err = nil
	s.emailQueueService = nil
	require.ErrorIs(t, s.SendEmailLoginCode(ctx, "u@example.com", ""), ErrServiceUnavailable)
}

package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Fork-local D85: a separate cache protocol prevents registration/recovery codes
// from becoming login credentials. Implementations must reserve and consume atomically.
type EmailLoginCodeCache interface {
	ReserveEmailLoginCode(ctx context.Context, email, digest string) (bool, error)
	ConsumeEmailLoginCode(ctx context.Context, email, digest string) (bool, error)
}

var ErrEmailCodeLoginDisabled = infraerrors.Forbidden("EMAIL_CODE_LOGIN_DISABLED", "email code login is disabled")

func (s *AuthService) emailLoginCache(ctx context.Context) (EmailLoginCodeCache, error) {
	if s.settingService == nil || !s.settingService.IsEmailCodeLoginEnabled(ctx) {
		return nil, ErrEmailCodeLoginDisabled
	}
	if s.emailService == nil || s.cfg == nil || s.cfg.JWT.Secret == "" {
		return nil, ErrServiceUnavailable
	}
	cache, ok := s.emailService.cache.(EmailLoginCodeCache)
	if !ok {
		return nil, ErrServiceUnavailable
	}
	return cache, nil
}

func (s *AuthService) emailLoginDigest(email, code string, user *User) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.JWT.Secret))
	var id, version int64
	if user != nil {
		id, version = user.ID, resolvedTokenVersion(user)
	}
	// A password change, account replacement or email change invalidates pending codes.
	_, _ = fmt.Fprintf(mac, "email-login\x00%s\x00%s\x00%d\x00%d", email, code, id, version)
	return hex.EncodeToString(mac.Sum(nil))
}

// SendEmailLoginCode returns the same result for unknown, inactive and eligible
// accounts. Both branches reserve the same cooldown and enter the bounded queue;
// SMTP runs outside the request so delivery timing does not disclose membership.
func (s *AuthService) SendEmailLoginCode(ctx context.Context, email, locale string) error {
	cache, err := s.emailLoginCache(ctx)
	if err != nil {
		return err
	}
	if s.emailQueueService == nil {
		return ErrServiceUnavailable
	}
	if _, err := s.emailService.GetSMTPConfig(ctx); err != nil {
		return ErrEmailNotConfigured
	}
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return ErrServiceUnavailable
	}
	if err != nil {
		user = nil
	}
	code, err := s.emailService.GenerateVerifyCode()
	if err != nil {
		return ErrServiceUnavailable
	}
	reserved, err := cache.ReserveEmailLoginCode(ctx, email, s.emailLoginDigest(email, code, user))
	if err != nil {
		return ErrServiceUnavailable
	}
	if !reserved {
		return nil
	} // generic success, including cooldown/hourly quota
	eligible := user != nil && user.IsActive() && !isReservedEmail(email)
	if eligible && s.settingService.IsBackendModeEnabled(ctx) && !user.IsAdmin() {
		eligible = false
	}
	task := EmailTask{Email: email, SiteName: s.settingService.GetSiteName(ctx), TaskType: TaskTypeEmailLogin, Locale: locale, LoginCode: code, SkipDelivery: !eligible}
	if err := s.emailQueueService.enqueueEmailLogin(task); err != nil {
		return ErrServiceUnavailable
	}
	return nil
}

func (s *AuthService) LoginWithEmailCode(ctx context.Context, email, code string) (*User, error) {
	cache, err := s.emailLoginCache(ctx)
	if err != nil {
		return nil, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, ErrServiceUnavailable
	}
	if err != nil {
		user = nil
	}
	valid, err := cache.ConsumeEmailLoginCode(ctx, email, s.emailLoginDigest(email, code, user))
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	if !valid || user == nil || !user.IsActive() || isReservedEmail(email) {
		return nil, ErrInvalidVerifyCode
	}
	return user, nil
}

func (s *EmailService) sendEmailLoginCode(ctx context.Context, task EmailTask) error {
	if task.SkipDelivery {
		return nil
	}
	subject := fmt.Sprintf("[%s] Sign-in verification code", task.SiteName)
	body := fmt.Sprintf("<p>Your sign-in code for %s is:</p><p style=\"font-size:32px;letter-spacing:6px\">%s</p><p>Expires in 5 minutes. Never share this code. If you did not request it, ignore this email.</p>", html.EscapeString(task.SiteName), html.EscapeString(task.LoginCode))
	if strings.HasPrefix(strings.ToLower(task.Locale), "zh") {
		subject = fmt.Sprintf("[%s] 登录验证码", task.SiteName)
		body = fmt.Sprintf("<p>您在 %s 的登录验证码为：</p><p style=\"font-size:32px;letter-spacing:6px\">%s</p><p>5 分钟内有效，请勿向任何人提供。如非本人操作，请忽略此邮件。</p>", html.EscapeString(task.SiteName), html.EscapeString(task.LoginCode))
	}
	return s.SendEmail(ctx, task.Email, subject, body)
}

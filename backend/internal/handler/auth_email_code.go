package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// Fork-local D85: dedicated endpoints, never reuse registration codes.
type EmailCodeLoginRequest struct {
	Email                 string `json:"email" binding:"required,email,max=254"`
	Code                  string `json:"code" binding:"required,len=6,numeric"`
	TurnstileToken        string `json:"turnstile_token"`
	TencentCaptchaTicket  string `json:"tencent_captcha_ticket"`
	TencentCaptchaRandstr string `json:"tencent_captcha_randstr"`
}

func (h *AuthHandler) SendEmailLoginCode(c *gin.Context) {
	var req SendVerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Email) > 254 {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.authService.VerifyCaptcha(c.Request.Context(), captchaProof(req.TurnstileToken, req.TencentCaptchaTicket, req.TencentCaptchaRandstr), ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.authService.SendEmailLoginCode(c.Request.Context(), req.Email, c.GetHeader("Accept-Language")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, SendVerifyCodeResponse{Message: "If the account is eligible, a sign-in code will be sent", Countdown: 60})
}

func (h *AuthHandler) LoginWithEmailCode(c *gin.Context) {
	var req EmailCodeLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.authService.VerifyCaptcha(c.Request.Context(), captchaProof(req.TurnstileToken, req.TencentCaptchaTicket, req.TencentCaptchaRandstr), ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	user, err := h.authService.LoginWithEmailCode(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.completePrimaryLogin(c, user)
}

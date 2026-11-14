//go:build unit

package handler

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestClassifyOpsUnsupportedModel(t *testing.T) {
	message := "The 'gpt-5.4-mini' model is not supported when using Codex with a ChatGPT account."
	for _, tc := range []struct {
		name                      string
		status                    int
		upstream                  int
		phase, typ, body, message string
		want                      bool
	}{
		{name: "upstream codex", status: 400, upstream: 400, phase: "upstream", message: message, want: true},
		{name: "structured code", status: 400, upstream: 400, body: `{"error":{"type":"invalid_request_error","code":"model_not_found","message":"unavailable"}}`, want: true},
		{name: "structured type", status: 400, body: `{"error":{"type":"model_not_found"}}`, want: true},
		{name: "local gateway", status: 400, typ: "model_not_found", want: true},
		{name: "detail body", status: 400, upstream: 400, body: `{"detail":"The model is not supported when using Codex with a ChatGPT account."}`, want: true},
		{name: "other invalid request", status: 400, message: "invalid request body"},
		{name: "server failure", status: 502, upstream: 502, message: message},
		{name: "rate limit", status: 429, upstream: 429, typ: "model_not_found", message: message},
		{name: "auth failure", status: 401, upstream: 401, message: message},
		{name: "forbidden", status: 403, upstream: 403, message: message},
		{name: "timeout", status: 504, upstream: 504, message: message},
		{name: "mismatched upstream auth", status: 400, upstream: 401, message: message},
		{name: "account auth stage", status: 400, upstream: 400, phase: "account_auth", message: message},
		{name: "recovered attempt", status: 200, upstream: 400, message: message},
		{name: "unrelated unsupported feature", status: 400, message: "OpenAI WSv1 is temporarily unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := &service.OpsInsertErrorLogInput{StatusCode: tc.status, ErrorPhase: tc.phase, ErrorType: tc.typ, ErrorBody: tc.body, ErrorMessage: tc.message, ErrorOwner: "provider", ErrorSource: "upstream_http"}
			if tc.upstream > 0 {
				entry.UpstreamStatusCode = &tc.upstream
			}
			classifyOpsUnsupportedModel(entry)
			require.Equal(t, tc.want, entry.IsBusinessLimited)
			require.Equal(t, "provider", entry.ErrorOwner)
			require.Equal(t, "upstream_http", entry.ErrorSource)
		})
	}
	classifyOpsUnsupportedModel(nil)
	entry := &service.OpsInsertErrorLogInput{StatusCode: http.StatusNotFound, IsBusinessLimited: true}
	classifyOpsUnsupportedModel(entry)
	require.True(t, entry.IsBusinessLimited, "preserve existing local model-configuration exclusion")
	upstreamMessage := message
	entry = &service.OpsInsertErrorLogInput{StatusCode: 400, UpstreamErrorMessage: &upstreamMessage}
	classifyOpsUnsupportedModel(entry)
	require.True(t, entry.IsBusinessLimited, "sanitized client envelope retains upstream reason")
}

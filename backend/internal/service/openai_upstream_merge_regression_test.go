package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMergeCodexModelsRetirementPreservesIndependentDisplaySelection(t *testing.T) {
	for _, accountType := range []string{AccountTypeOAuth, AccountTypeAPIKey} {
		t.Run(accountType, func(t *testing.T) {
			group := &Group{
				ID: 1, Platform: PlatformOpenAI,
				ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.5", "gpt-6.1-sol"}},
			}
			svc := &OpenAIGatewayService{accountRepo: codexModelsVisibilityAccountRepo{
				byGroup: map[int64][]Account{1: {{ID: 1, Platform: PlatformOpenAI, Type: accountType}}},
			}}
			body := []byte(`{"models":[{"slug":"gpt-5.5"},{"slug":"gpt-6.1-sol","unknown":true},{"slug":"gpt-5.6-sol"}]}`)
			oldETag := CodexModelsManifestETag(body)
			manifest := &OpenAIModelsResponse{Body: body, ETag: oldETag}

			require.NoError(t, svc.MergeGroupConfiguredCodexModels(context.Background(), group, manifest, oldETag))

			want := []string{"gpt-6.1-sol"}
			if accountType == AccountTypeAPIKey {
				want = []string{"gpt-5.5", "gpt-6.1-sol"}
			}
			require.Equal(t, want, codexManifestModelSlugs(t, manifest.Body), "display selection must apply alongside account-specific retirement")
			require.True(t, gjson.GetBytes(manifest.Body, `models.#(slug=="gpt-6.1-sol").unknown`).Bool())
			require.False(t, manifest.NotModified, "the stale upstream ETag must not hide the filtered catalog")
			require.Equal(t, CodexModelsManifestETag(manifest.Body), manifest.ETag)
			require.NotEqual(t, oldETag, manifest.ETag)
			require.False(t, group.ModelAllowlistEnabled())
			require.True(t, group.ModelAllowlist.Allows("gpt-5.6-sol"), "hiding a model must not enable request restrictions")
		})
	}
}

func TestStreamingCreatedAtStampingPreservesPublicModelRewrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mapped := range []string{"provider-model", "public"} {
		t.Run(mapped, func(t *testing.T) {
			body := `data: {"type":"response.created","response":{"id":"resp_1","object":"response","model":"provider-alias","status":"in_progress","output":[]}}` + "\n\n" +
				`data: {"type":"response.completed","response":{"id":"resp_1","object":"response","model":"provider-alias","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"provider-alias"}]}],"usage":{"input_tokens":1,"output_tokens":1}}}` + "\n\n"
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
			svc := &OpenAIGatewayService{}

			_, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "public", mapped)

			require.NoError(t, err)
			payloads := sseDataPayloads(recorder.Body.String())
			require.Len(t, payloads, 2)
			createdAt := gjson.Get(payloads[0], "response.created_at")
			require.Positive(t, createdAt.Int())
			for _, payload := range payloads {
				require.Equal(t, createdAt.Int(), gjson.Get(payload, "response.created_at").Int(), "clients require a stable timestamp across lifecycle events")
				require.Equal(t, "public", gjson.Get(payload, "response.model").String(), "clients must see their requested name even without a channel mapping")
			}
			require.Equal(t, "provider-alias", gjson.Get(payloads[1], "response.output.0.content.0.text").String())
			require.Equal(t, "provider-alias", observedUpstreamResponseModel(c))
		})
	}
}

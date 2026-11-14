//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// D144: existing group display configuration can narrow both mapped and default
// catalogs without changing account capability or request authorization.
func TestGatewayModelsD144DisplayConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, typ := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		for _, mapped := range []bool{false, true} {
			account := service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: typ}
			if mapped {
				account.Credentials = map[string]any{"model_mapping": map[string]any{"gpt-6.1-sol": "gpt-6.1-sol", "gpt-5.4-mini": "gpt-5.4-mini"}}
			}
			group := &service.Group{ID: 144, Platform: service.PlatformOpenAI, ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-6.1-sol"}}}
			h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{byGroup: map[int64][]service.Account{144: {account}}})
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{Group: group})
			h.Models(c)
			require.Equal(t, http.StatusOK, rec.Code)
			var got gatewayModelsResponseForTest
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Equal(t, []string{"gpt-6.1-sol"}, modelIDsForTest(got.Data))
			require.True(t, group.ModelAllowlist.Allows("gpt-5.4-mini"), "display selection must not enable request restrictions")
			require.True(t, account.IsModelSupported("gpt-5.4-mini"), "display selection must not change routing capabilities")
		}
	}
}

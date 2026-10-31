//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type checkoutSettingRepo struct {
	service.SettingRepository
	values             map[string]string
	getMultipleCalls   int
	failPlanConfigRead bool
}

func (r *checkoutSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}

func (r *checkoutSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.getMultipleCalls++
	if r.failPlanConfigRead && r.getMultipleCalls == 2 {
		return nil, errors.New("settings database unavailable")
	}
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = r.values[key]
	}
	return values, nil
}

func TestGetCheckoutInfoKeepsBalanceRechargeAvailableWithBrokenRenewalPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, test := range []struct {
		name               string
		policy             func(legacyID, replacementID int64) *service.SubscriptionRenewalPolicy
		wantError          bool
		failPlanConfigRead bool
	}{
		{
			name: "missing legacy plan",
			policy: func(_, replacementID int64) *service.SubscriptionRenewalPolicy {
				return &service.SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{999}, ReplacementPlanIDs: []int64{replacementID}}
			},
			wantError: true,
		},
		{
			name: "missing replacement plan",
			policy: func(legacyID, _ int64) *service.SubscriptionRenewalPolicy {
				return &service.SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{legacyID}, ReplacementPlanIDs: []int64{999}}
			},
			wantError: true,
		},
		{
			name: "invalid enabled policy",
			policy: func(_, _ int64) *service.SubscriptionRenewalPolicy {
				return &service.SubscriptionRenewalPolicy{Enabled: true}
			},
			wantError: true,
		},
		{
			name: "plan configuration infrastructure failure",
			policy: func(legacyID, replacementID int64) *service.SubscriptionRenewalPolicy {
				return &service.SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{legacyID}, ReplacementPlanIDs: []int64{replacementID}}
			},
			failPlanConfigRead: true,
		},
		{
			name: "valid policy",
			policy: func(legacyID, replacementID int64) *service.SubscriptionRenewalPolicy {
				return &service.SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{legacyID}, ReplacementPlanIDs: []int64{replacementID}}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "checkout.db"))
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			_, err = db.Exec("PRAGMA foreign_keys = ON")
			require.NoError(t, err)
			client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
			t.Cleanup(func() { _ = client.Close() })

			user := client.User.Create().SetEmail("checkout@example.com").SetPasswordHash("hash").SaveX(ctx)
			legacyGroup := client.Group.Create().SetName("legacy").SetPlatform("openai").SetSubscriptionType("subscription").SaveX(ctx)
			currentGroup := client.Group.Create().SetName("current").SetPlatform("openai").SetSubscriptionType("subscription").SaveX(ctx)
			legacy := client.SubscriptionPlan.Create().SetGroupID(legacyGroup.ID).SetName("legacy").SetPrice(10).SetForSale(false).SaveX(ctx)
			current := client.SubscriptionPlan.Create().SetGroupID(currentGroup.ID).SetName("current").SetPrice(20).SetForSale(true).SaveX(ctx)
			client.PaymentProviderInstance.Create().SetProviderKey(payment.TypeStripe).
				SetConfig(`{"currency":"USD"}`).SetSupportedTypes("card,link").
				SetLimits(`{"stripe":{"singleMin":5,"singleMax":500}}`).SaveX(ctx)

			rawPolicy, err := json.Marshal(test.policy(legacy.ID, current.ID))
			require.NoError(t, err)
			repo := &checkoutSettingRepo{values: map[string]string{
				service.SettingPaymentEnabled:            "true",
				service.SettingBalanceRechargeMult:       "1.5",
				service.SettingRechargeFeeRate:           "0.03",
				service.SettingHelpText:                  "Balance recharge help",
				service.SettingSubscriptionRenewalPolicy: string(rawPolicy),
			}}
			repo.failPlanConfigRead = test.failPlanConfigRead
			configSvc := service.NewPaymentConfigService(client, repo, nil)
			paymentSvc := service.NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, configSvc, nil, nil, nil)
			h := NewPaymentHandler(paymentSvc, configSvc)
			router := gin.New()
			router.GET("/api/v1/payment/checkout-info", func(c *gin.Context) {
				c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})
				h.GetCheckoutInfo(c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil))

			if test.failPlanConfigRead {
				// Initial payment configuration succeeds; failure inside ListPlansForSale must stay fatal.
				require.Equal(t, 2, repo.getMultipleCalls)
				require.Equal(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
				var failure struct {
					Code int             `json:"code"`
					Data json.RawMessage `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &failure))
				require.Equal(t, http.StatusInternalServerError, failure.Code)
				require.Empty(t, failure.Data)
				return
			}

			// A subscription configuration fault must not disable the independent balance checkout.
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			var resp struct {
				Code int                        `json:"code"`
				Data map[string]json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
			require.Zero(t, resp.Code)
			var checkout checkoutInfoResponse
			data, err := json.Marshal(resp.Data)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, &checkout))
			require.False(t, checkout.BalanceDisabled)
			require.Equal(t, 1.5, checkout.BalanceRechargeMultiplier)
			require.Equal(t, 0.03, checkout.RechargeFeeRate)
			require.Equal(t, "Balance recharge help", checkout.HelpText)
			require.Contains(t, checkout.Methods, payment.TypeStripe)
			require.Equal(t, payment.TypeStripe, checkout.Methods[payment.TypeStripe].PaymentType)
			require.Equal(t, 5.0, checkout.GlobalMin)
			require.Equal(t, 500.0, checkout.GlobalMax)
			if test.wantError {
				require.Equal(t, "INVALID_RENEWAL_POLICY", checkout.SubscriptionRenewalError)
				require.NotContains(t, resp.Data, "subscription_renewal")
			} else {
				require.NotContains(t, resp.Data, "subscription_renewal_error")
				require.NotNil(t, checkout.SubscriptionRenewal)
				require.Equal(t, []int64{legacyGroup.ID}, checkout.SubscriptionRenewal.LegacyGroupIDs)
				require.Len(t, checkout.Plans, 1)
				require.Equal(t, current.ID, checkout.Plans[0].ID)
			}
		})
	}
}

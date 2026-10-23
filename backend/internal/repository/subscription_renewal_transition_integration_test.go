//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestGrandfatheredRenewalDefersThenActivatesExactlyOnce(t *testing.T) {
	ctx := context.Background()
	client := integrationEntClient
	stamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	u := client.User.Create().SetEmail("renewal-" + stamp + "@example.test").SetPasswordHash("test-only").SaveX(ctx)
	old := client.Group.Create().SetName("legacy-" + stamp).SetPlatform("openai").SetSubscriptionType("subscription").SetDailyLimitUsd(22.5).SetWeeklyLimitUsd(75).SetMonthlyLimitUsd(150).SaveX(ctx)
	current := client.Group.Create().SetName("replacement-" + stamp).SetPlatform("openai").SetSubscriptionType("subscription").SetDailyLimitUsd(5).SaveX(ctx)
	oldExpiry := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	original := client.UserSubscription.Create().SetUserID(u.ID).SetGroupID(old.ID).SetStartsAt(time.Now().Add(-24 * time.Hour)).SetExpiresAt(oldExpiry).SetDailyUsageUsd(3).SetMonthlyUsageUsd(99).SaveX(ctx)
	key := client.APIKey.Create().SetUserID(u.ID).SetGroupID(old.ID).SetKey("test-only-renewal-" + stamp).SetName("original key").SetQuota(100).SaveX(ctx)
	unrelatedUser := client.User.Create().SetEmail("other-" + stamp + "@example.test").SetPasswordHash("test-only").SaveX(ctx)
	unrelatedKey := client.APIKey.Create().SetUserID(unrelatedUser.ID).SetGroupID(old.ID).SetKey("test-only-other-" + stamp).SetName("other owner").SaveX(ctx)
	order := client.PaymentOrder.Create().SetUserID(u.ID).SetUserEmail(u.Email).SetUserName("test").SetClientIP("127.0.0.1").SetSrcHost("test.local").SetAmount(63).SetPayAmount(63).SetRechargeCode("").SetPaymentType("alipay").SetPaymentTradeNo("test-trade").SetOrderType("subscription").SetSubscriptionGroupID(current.ID).SetSubscriptionDays(30).SetStatus(service.OrderStatusPaid).SetPaidAt(time.Now()).SetExpiresAt(time.Now().Add(-time.Minute)).SetProviderSnapshot(map[string]any{"subscription_renewal": map[string]any{"activation_unix_ms": oldExpiry.UnixMilli() + 1, "legacy_group_ids": []int64{old.ID}}}).SaveX(ctx)
	t.Cleanup(func() {
		cleanCtx := mixins.SkipSoftDelete(context.Background())
		_, _ = client.PaymentAuditLog.Delete().Where(paymentauditlog.OrderIDEQ(fmt.Sprint(order.ID))).Exec(cleanCtx)
		_, _ = client.PaymentOrder.Delete().Where(paymentorder.UserIDIn(u.ID, unrelatedUser.ID)).Exec(cleanCtx)
		_, _ = client.APIKey.Delete().Where(apikey.UserIDIn(u.ID, unrelatedUser.ID)).Exec(cleanCtx)
		_, _ = client.UserSubscription.Delete().Where(usersubscription.UserIDIn(u.ID, unrelatedUser.ID)).Exec(cleanCtx)
		_ = client.Group.DeleteOneID(old.ID).Exec(cleanCtx)
		_ = client.Group.DeleteOneID(current.ID).Exec(cleanCtx)
		_ = client.User.DeleteOneID(u.ID).Exec(cleanCtx)
		_ = client.User.DeleteOneID(unrelatedUser.ID).Exec(cleanCtx)
	})
	groups := NewGroupRepository(client, integrationDB)
	subs := NewUserSubscriptionRepository(client)
	subscriptionService := service.NewSubscriptionService(groups, subs, nil, client, nil)
	t.Cleanup(subscriptionService.Stop)
	svc := service.NewPaymentService(client, nil, nil, nil, subscriptionService, nil, nil, groups, nil)
	require.NoError(t, svc.ExecuteSubscriptionFulfillment(ctx, order.ID))
	require.Equal(t, service.OrderStatusScheduled, client.PaymentOrder.GetX(ctx, order.ID).Status)
	require.False(t, client.UserSubscription.Query().Where(usersubscription.UserIDEQ(u.ID), usersubscription.GroupIDEQ(current.ID)).ExistX(ctx))
	require.Equal(t, old.ID, *client.APIKey.GetX(ctx, key.ID).GroupID)
	preserved := client.UserSubscription.GetX(ctx, original.ID)
	require.True(t, oldExpiry.Equal(preserved.ExpiresAt))
	require.Equal(t, 99.0, preserved.MonthlyUsageUsd)
	// Boundary crossed: two webhook/job workers race on the same paid order.
	past := time.Now().Add(-time.Second)
	client.UserSubscription.UpdateOneID(original.ID).SetExpiresAt(past).SaveX(ctx)
	client.PaymentOrder.UpdateOneID(order.ID).SetProviderSnapshot(map[string]any{"subscription_renewal": map[string]any{"activation_unix_ms": past.UnixMilli(), "legacy_group_ids": []int64{old.ID}}}).SaveX(ctx)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- svc.ExecuteSubscriptionFulfillment(ctx, order.ID) }()
	}
	wg.Wait()
	close(errs)
	// A fresh lease conflict is an expected loser; retry must converge idempotently.
	for err := range errs {
		if err != nil {
			require.Equal(t, "CONFLICT", infraerrors.Reason(err))
		}
	}
	require.NoError(t, svc.ExecuteSubscriptionFulfillment(ctx, order.ID))
	require.Equal(t, service.OrderStatusCompleted, client.PaymentOrder.GetX(ctx, order.ID).Status)
	granted := client.UserSubscription.Query().Where(usersubscription.UserIDEQ(u.ID), usersubscription.GroupIDEQ(current.ID)).OnlyX(ctx)
	require.WithinDuration(t, time.Now().AddDate(0, 0, 30), granted.ExpiresAt, 5*time.Second)
	require.Equal(t, current.ID, *client.APIKey.GetX(ctx, key.ID).GroupID)
	require.Equal(t, key.Key, client.APIKey.GetX(ctx, key.ID).Key)
	require.Equal(t, 100.0, client.APIKey.GetX(ctx, key.ID).Quota)
	require.Equal(t, old.ID, *client.APIKey.GetX(ctx, unrelatedKey.ID).GroupID)
	require.Equal(t, 1, client.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(fmt.Sprint(order.ID)), paymentauditlog.ActionEQ("SUBSCRIPTION_ASSIGNED")).CountX(ctx))
	require.Equal(t, 99.0, client.UserSubscription.GetX(ctx, original.ID).MonthlyUsageUsd)
	// A crash after assignment but before final status must be recoverable by
	// the periodic job without adding another month or changing a key secret.
	client.PaymentOrder.UpdateOneID(order.ID).SetStatus(service.OrderStatusRecharging).SetUpdatedAt(time.Now().Add(-time.Hour)).SaveX(ctx)
	_, err := svc.ActivateDueSubscriptionRenewals(ctx)
	require.NoError(t, err)
	require.Equal(t, service.OrderStatusCompleted, client.PaymentOrder.GetX(ctx, order.ID).Status)
	require.True(t, granted.ExpiresAt.Equal(client.UserSubscription.GetX(ctx, granted.ID).ExpiresAt))
	// Keep synthetic fixtures within this disposable integration database.
}

type renewalSettings struct {
	service.SettingRepository
	values map[string]string
}

func (s *renewalSettings) GetMultiple(context.Context, []string) (map[string]string, error) {
	return s.values, nil
}
func (s *renewalSettings) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s *renewalSettings) SetMultiple(_ context.Context, values map[string]string) error {
	for k, v := range values {
		s.values[k] = v
	}
	return nil
}

func TestReplacementCheckoutCapturesServerPriceAndLegacyEnd(t *testing.T) {
	ctx := context.Background()
	client := integrationEntClient
	stamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	u := client.User.Create().SetEmail("checkout-renewal-" + stamp + "@example.test").SetPasswordHash("test-only").SaveX(ctx)
	legacy := client.Group.Create().SetName("old-checkout-" + stamp).SetPlatform("openai").SetSubscriptionType("subscription").SaveX(ctx)
	current := client.Group.Create().SetName("new-checkout-" + stamp).SetPlatform("openai").SetSubscriptionType("subscription").SaveX(ctx)
	oldPlan := client.SubscriptionPlan.Create().SetName("legacy plan").SetGroupID(legacy.ID).SetPrice(60).SaveX(ctx)
	newPlan := client.SubscriptionPlan.Create().SetName("current plan").SetGroupID(current.ID).SetPrice(63).SaveX(ctx)
	end := time.Now().Add(10 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	client.UserSubscription.Create().SetUserID(u.ID).SetGroupID(legacy.ID).SetStartsAt(time.Now().Add(-time.Hour)).SetExpiresAt(end).SaveX(ctx)
	// EasyPay popup mode only builds a redirect URL. It performs no network calls.
	providerConfig, _ := json.Marshal(map[string]string{"pid": "local-test", "pkey": "local-test-only", "apiBase": "https://payments.example.test", "notifyUrl": "https://site.example.test/notify", "returnUrl": "https://site.example.test/return", "paymentMode": "popup"})
	instance := client.PaymentProviderInstance.Create().SetProviderKey(payment.TypeEasyPay).SetConfig(string(providerConfig)).SetSupportedTypes("alipay").SetPaymentMode("popup").SaveX(ctx)
	t.Cleanup(func() {
		clean := mixins.SkipSoftDelete(context.Background())
		_, _ = client.PaymentOrder.Delete().Where(paymentorder.UserIDEQ(u.ID)).Exec(clean)
		_, _ = client.UserSubscription.Delete().Where(usersubscription.UserIDEQ(u.ID)).Exec(clean)
		_ = client.SubscriptionPlan.DeleteOneID(oldPlan.ID).Exec(clean)
		_ = client.SubscriptionPlan.DeleteOneID(newPlan.ID).Exec(clean)
		_ = client.PaymentProviderInstance.DeleteOneID(instance.ID).Exec(clean)
		_ = client.Group.DeleteOneID(legacy.ID).Exec(clean)
		_ = client.Group.DeleteOneID(current.ID).Exec(clean)
		_ = client.User.DeleteOneID(u.ID).Exec(clean)
	})
	settings := &renewalSettings{values: map[string]string{service.SettingPaymentEnabled: "true"}}
	cfgService := service.NewPaymentConfigService(client, settings, nil)
	require.NoError(t, cfgService.UpdatePaymentConfig(ctx, service.UpdatePaymentConfigRequest{SubscriptionRenewalPolicy: &service.SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{oldPlan.ID}, ReplacementPlanIDs: []int64{newPlan.ID}}}))
	offered, err := cfgService.ListPlansForSale(ctx)
	require.NoError(t, err)
	require.Len(t, offered, 1)
	require.Equal(t, newPlan.ID, offered[0].ID)
	groups := NewGroupRepository(client, integrationDB)
	svc := service.NewPaymentService(client, payment.NewRegistry(), payment.NewDefaultLoadBalancer(client, nil), nil, nil, cfgService, NewUserRepository(client, integrationDB), groups, nil)
	response, err := svc.CreateOrder(ctx, service.CreateOrderRequest{UserID: u.ID, PlanID: newPlan.ID, Amount: 0.01, OrderType: payment.OrderTypeSubscription, PaymentType: "alipay", ClientIP: "127.0.0.1", SrcHost: "https://site.example.test"})
	require.NoError(t, err)
	stored := client.PaymentOrder.GetX(ctx, response.OrderID)
	require.Equal(t, 63.0, stored.Amount)
	require.Equal(t, 30, *stored.SubscriptionDays)
	require.WithinDuration(t, end, *service.PaymentOrderScheduledActivation(stored), time.Millisecond)
	require.Contains(t, response.PayURL, "payments.example.test/submit.php")
	// Once the old entitlement has expired, a fresh purchase is immediate.
	_, err = client.UserSubscription.Update().Where(usersubscription.UserIDEQ(u.ID), usersubscription.GroupIDEQ(legacy.ID)).SetExpiresAt(time.Now().Add(-time.Hour)).Save(ctx)
	require.NoError(t, err)
	immediate, err := svc.CreateOrder(ctx, service.CreateOrderRequest{UserID: u.ID, PlanID: newPlan.ID, OrderType: payment.OrderTypeSubscription, PaymentType: "alipay", ClientIP: "127.0.0.1", SrcHost: "https://site.example.test"})
	require.NoError(t, err)
	require.Nil(t, service.PaymentOrderScheduledActivation(client.PaymentOrder.GetX(ctx, immediate.OrderID)))

	_, err = svc.CreateOrder(ctx, service.CreateOrderRequest{UserID: u.ID, PlanID: oldPlan.ID, OrderType: payment.OrderTypeSubscription, PaymentType: "alipay"})
	require.Equal(t, "PLAN_RETIRED", infraerrors.Reason(err))
}

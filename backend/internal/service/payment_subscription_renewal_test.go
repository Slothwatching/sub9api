//go:build unit

package service

import (
	"context"
	"encoding/json"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSubscriptionRenewalPolicyFailsClosed(t *testing.T) {
	require.Nil(t, parseSubscriptionRenewalPolicy(""))
	require.Error(t, validateSubscriptionRenewalPolicy(parseSubscriptionRenewalPolicy("corrupt")))
	require.Error(t, validateSubscriptionRenewalPolicy(&SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{1}, ReplacementPlanIDs: []int64{1}}))
	require.Error(t, validateSubscriptionRenewalPolicy(&SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{1, 1}, ReplacementPlanIDs: []int64{9}}))
	require.NoError(t, validateSubscriptionRenewalPolicy(&SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{1}, ReplacementPlanIDs: []int64{9}}))
}
func renewalSnapshot(at time.Time, gids ...int64) map[string]any {
	return map[string]any{renewalSnapshotKey: subscriptionRenewalSnapshot{ActivationMillis: at.UnixMilli() + 1, LegacyGroupIDs: gids}}
}
func TestScheduledRenewalPaymentKeepsLegacyEntitlementUntouched(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, OrderStatusPaid, time.Now())
	legacy, err := client.Group.Create().SetName("legacy").SetSubscriptionType("subscription").Save(ctx)
	require.NoError(t, err)
	expires := time.Now().Add(48 * time.Hour).Truncate(time.Microsecond)
	sub, err := client.UserSubscription.Create().SetUserID(order.UserID).SetGroupID(legacy.ID).SetStartsAt(time.Now().Add(-time.Hour)).SetExpiresAt(expires).SetDailyUsageUsd(7.25).SetMonthlyUsageUsd(42).Save(ctx)
	require.NoError(t, err)
	order, err = client.PaymentOrder.UpdateOneID(order.ID).SetProviderSnapshot(renewalSnapshot(expires, legacy.ID)).SetStatus(OrderStatusPending).ClearPaidAt().Save(ctx)
	require.NoError(t, err)
	svc := &PaymentService{entClient: client}
	for range 2 {
		require.NoError(t, svc.toPaid(ctx, order, "test-trade", order.PayAmount, "alipay"))
	}
	unchanged, err := client.UserSubscription.Get(ctx, sub.ID)
	require.NoError(t, err)
	require.True(t, expires.Equal(unchanged.ExpiresAt))
	require.Equal(t, 7.25, unchanged.DailyUsageUsd)
	require.Equal(t, 42.0, unchanged.MonthlyUsageUsd)
	require.Equal(t, 1, client.UserSubscription.Query().CountX(ctx))
	require.Equal(t, OrderStatusScheduled, client.PaymentOrder.GetX(ctx, order.ID).Status)
	// An administrative extension cannot make the replacement activate early.
	extended := expires.Add(24 * time.Hour)
	client.UserSubscription.UpdateOneID(sub.ID).SetExpiresAt(extended).SaveX(ctx)
	require.NoError(t, svc.ExecuteSubscriptionFulfillment(ctx, order.ID))
	require.WithinDuration(t, extended, *PaymentOrderScheduledActivation(client.PaymentOrder.GetX(ctx, order.ID)), time.Millisecond)
}
func TestLegacyPlanRejectedBeforeProviderOrUserAccess(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	policy := &SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{1}, ReplacementPlanIDs: []int64{9}}
	data, err := json.Marshal(policy)
	require.NoError(t, err)
	settings := &paymentFulfillmentSettingRepoStub{values: map[string]string{SettingPaymentEnabled: "true", SettingSubscriptionRenewalPolicy: string(data)}}
	svc := &PaymentService{entClient: client, configService: NewPaymentConfigService(client, settings, nil)}
	_, err = svc.CreateOrder(ctx, CreateOrderRequest{UserID: 1, OrderType: payment.OrderTypeSubscription, PlanID: 1, PaymentType: "alipay"})
	require.Equal(t, "PLAN_RETIRED", infraerrors.Reason(err))
	require.Zero(t, client.PaymentOrder.Query().CountX(ctx))
}
func TestScheduledRenewalSnapshotSurvivesJSONAndRejectsMalformedData(t *testing.T) {
	now := time.Now()
	raw, err := json.Marshal(renewalSnapshot(now, 7))
	require.NoError(t, err)
	var stored map[string]any
	require.NoError(t, json.Unmarshal(raw, &stored))
	order := &dbent.PaymentOrder{Status: OrderStatusPaid, ProviderSnapshot: stored}
	require.WithinDuration(t, now, *PaymentOrderScheduledActivation(order), time.Millisecond)
	order.ProviderSnapshot[renewalSnapshotKey] = map[string]any{"activation_unix_ms": "not-a-date"}
	_, err = readSubscriptionRenewal(order)
	require.Error(t, err)
}

func TestScheduledRenewalFullRefundDoesNotDeductExistingEntitlements(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createCompletedRefundableOrderForTest(t, ctx, client, "scheduled-renewal-refund")
	// A paid order without a provider trade reference is the existing local-only
	// refund fixture; no real gateway, account or money is involved.
	order = client.PaymentOrder.UpdateOneID(order.ID).SetStatus(OrderStatusScheduled).SetOrderType(payment.OrderTypeSubscription).SetSubscriptionGroupID(7).SetSubscriptionDays(30).SetPaymentTradeNo("").SetProviderSnapshot(renewalSnapshot(time.Now().Add(time.Hour), 1)).SaveX(ctx)
	svc := &PaymentService{entClient: client}
	_, _, err := svc.PrepareRefund(ctx, order.ID, 50, "partial", false, true)
	require.Equal(t, "INVALID_AMOUNT", infraerrors.Reason(err))
	plan, early, err := svc.PrepareRefund(ctx, order.ID, 0, "cancel scheduled term", false, true)
	require.NoError(t, err)
	require.Nil(t, early)
	require.True(t, plan.ScheduledRenewal)
	require.Equal(t, payment.DeductionTypeNone, plan.DeductionType)
	result, err := svc.ExecuteRefund(ctx, plan)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, OrderStatusRefunded, client.PaymentOrder.GetX(ctx, order.ID).Status)
	count, err := svc.ActivateDueSubscriptionRenewals(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestUnpaidReplacementCannotActivate(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, OrderStatusFailed, time.Now())
	order = client.PaymentOrder.UpdateOneID(order.ID).ClearPaidAt().SetProviderSnapshot(renewalSnapshot(time.Now().Add(-time.Hour), 1)).SaveX(ctx)
	svc := &PaymentService{entClient: client}
	require.Equal(t, "INVALID_STATUS", infraerrors.Reason(svc.ExecuteSubscriptionFulfillment(ctx, order.ID)))
	require.Zero(t, client.UserSubscription.Query().CountX(ctx))
}

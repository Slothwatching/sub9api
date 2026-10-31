//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestDeletePlanPreservesRenewalReferences(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		for _, legacy := range []bool{true, false} {
			t.Run(fmtPolicyCase(enabled, legacy), func(t *testing.T) {
				ctx := context.Background()
				client := newPaymentConfigServiceTestClient(t)
				group := client.Group.Create().SetName("referenced").SetSubscriptionType("subscription").SaveX(ctx)
				plan := client.SubscriptionPlan.Create().SetGroupID(group.ID).SetName("referenced").SetPrice(10).SaveX(ctx)
				policy := &SubscriptionRenewalPolicy{Enabled: enabled, LegacyPlanIDs: []int64{plan.ID}, ReplacementPlanIDs: []int64{999}}
				if !legacy {
					policy.LegacyPlanIDs, policy.ReplacementPlanIDs = policy.ReplacementPlanIDs, policy.LegacyPlanIDs
				}
				raw, err := json.Marshal(policy)
				require.NoError(t, err)
				repo := &paymentConfigSettingRepoStub{values: map[string]string{SettingSubscriptionRenewalPolicy: string(raw)}}
				svc := NewPaymentConfigService(client, repo, nil)
				require.Equal(t, "PLAN_IN_RENEWAL_POLICY", infraerrors.Reason(svc.DeletePlan(ctx, plan.ID)))
				require.NotNil(t, client.SubscriptionPlan.GetX(ctx, plan.ID))
				// Off-sale products still identify grandfathered groups and remain editable.
				offSale := false
				_, err = svc.UpdatePlan(ctx, plan.ID, UpdatePlanRequest{ForSale: &offSale})
				require.NoError(t, err)
				require.False(t, client.SubscriptionPlan.GetX(ctx, plan.ID).ForSale)
				// Explicitly removing the policy reference makes an unused product deletable.
				repo.values[SettingSubscriptionRenewalPolicy] = ""
				require.NoError(t, svc.DeletePlan(ctx, plan.ID))
				require.False(t, client.SubscriptionPlan.Query().ExistX(ctx))
			})
		}
	}
}

func fmtPolicyCase(enabled, legacy bool) string {
	if enabled {
		if legacy {
			return "enabled legacy"
		}
		return "enabled replacement"
	}
	if legacy {
		return "disabled legacy"
	}
	return "disabled replacement"
}

type renewalPolicyReadFailure struct{ SettingRepository }

func (renewalPolicyReadFailure) GetValue(context.Context, string) (string, error) {
	return "", errors.New("settings unavailable")
}

func TestDeletePlanDoesNotIgnoreUnreadableRenewalPolicy(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	group := client.Group.Create().SetName("protected").SaveX(ctx)
	plan := client.SubscriptionPlan.Create().SetGroupID(group.ID).SetName("protected").SetPrice(10).SaveX(ctx)
	svc := NewPaymentConfigService(client, renewalPolicyReadFailure{}, nil)
	require.ErrorContains(t, svc.DeletePlan(ctx, plan.ID), "settings unavailable")
	require.NotNil(t, client.SubscriptionPlan.GetX(ctx, plan.ID))
}

func TestSubscriptionPurchaseFailsClosedWhenRenewalPlanIsMissing(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	group := client.Group.Create().SetName("current").SetSubscriptionType("subscription").SaveX(ctx)
	plan := client.SubscriptionPlan.Create().SetGroupID(group.ID).SetName("current").SetPrice(10).SaveX(ctx)
	cfg := &PaymentConfig{SubscriptionRenewalPolicy: &SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{999}, ReplacementPlanIDs: []int64{plan.ID}}}
	svc := &PaymentService{entClient: client, configService: NewPaymentConfigService(client, nil, nil)}
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()
	// Even a user with no legacy subscription must not buy while its activation rules are unknown.
	_, err = svc.attachSubscriptionRenewal(ctx, tx, CreateOrderRequest{UserID: 1, OrderType: payment.OrderTypeSubscription, PlanID: plan.ID}, cfg, nil)
	require.Equal(t, "INVALID_RENEWAL_POLICY", infraerrors.Reason(err))
	snapshot, err := svc.attachSubscriptionRenewal(ctx, tx, CreateOrderRequest{UserID: 1, OrderType: payment.OrderTypeBalance}, cfg, map[string]any{"currency": "USD"})
	require.NoError(t, err)
	require.Equal(t, "USD", snapshot["currency"])
	require.Zero(t, tx.PaymentOrder.Query().CountX(ctx))
}

func TestRenewalPolicyRepairKeepsGrandfatheredEntitlements(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	oldGroup := client.Group.Create().SetName("legacy").SetSubscriptionType("subscription").SaveX(ctx)
	newGroup := client.Group.Create().SetName("current").SetSubscriptionType("subscription").SaveX(ctx)
	oldPlan := client.SubscriptionPlan.Create().SetGroupID(oldGroup.ID).SetName("legacy").SetPrice(10).SetForSale(false).SaveX(ctx)
	newPlan := client.SubscriptionPlan.Create().SetGroupID(newGroup.ID).SetName("current").SetPrice(20).SaveX(ctx)
	user := client.User.Create().SetEmail("renewal-repair@example.com").SetPasswordHash("hash").SaveX(ctx)
	end := time.Now().Add(48 * time.Hour).Truncate(time.Microsecond)
	sub := client.UserSubscription.Create().SetUserID(user.ID).SetGroupID(oldGroup.ID).SetStartsAt(time.Now().Add(-time.Hour)).SetExpiresAt(end).SetMonthlyUsageUsd(42).SaveX(ctx)
	broken := &SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{oldPlan.ID, 999}, ReplacementPlanIDs: []int64{newPlan.ID}}
	raw, err := json.Marshal(broken)
	require.NoError(t, err)
	repo := &paymentConfigSettingRepoStub{values: map[string]string{SettingSubscriptionRenewalPolicy: string(raw)}}
	cfgSvc := NewPaymentConfigService(client, repo, nil)
	require.Equal(t, "INVALID_RENEWAL_POLICY", infraerrors.Reason(cfgSvc.UpdatePaymentConfig(ctx, UpdatePaymentConfigRequest{SubscriptionRenewalPolicy: broken})))
	repaired := &SubscriptionRenewalPolicy{Enabled: true, LegacyPlanIDs: []int64{oldPlan.ID}, ReplacementPlanIDs: []int64{newPlan.ID}}
	require.NoError(t, cfgSvc.UpdatePaymentConfig(ctx, UpdatePaymentConfigRequest{SubscriptionRenewalPolicy: repaired}))
	cfg, err := cfgSvc.GetPaymentConfig(ctx)
	require.NoError(t, err)
	svc := &PaymentService{entClient: client, configService: cfgSvc}
	info, err := svc.SubscriptionRenewalInfo(ctx, user.ID, cfg)
	require.NoError(t, err)
	require.WithinDuration(t, end, *info.ActivationAt, time.Millisecond)
	require.Equal(t, []int64{oldGroup.ID}, info.LegacyGroupIDs)
	unchanged := client.UserSubscription.GetX(ctx, sub.ID)
	require.True(t, unchanged.ExpiresAt.Equal(end))
	require.Equal(t, 42.0, unchanged.MonthlyUsageUsd)
	require.Equal(t, oldGroup.ID, unchanged.GroupID)
}

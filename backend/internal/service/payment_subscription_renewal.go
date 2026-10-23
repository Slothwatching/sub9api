package service

// D93: replacement orders wait durably in their existing order snapshot. The
// grandfathered subscription and quota group are never edited by the renewal.
import (
	"context"
	"encoding/json"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"errors"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"sort"
	"time"
)

const SettingSubscriptionRenewalPolicy = "SUBSCRIPTION_RENEWAL_POLICY"
const renewalSnapshotKey = "subscription_renewal"

type SubscriptionRenewalPolicy struct {
	Enabled            bool    `json:"enabled"`
	LegacyPlanIDs      []int64 `json:"legacy_plan_ids"`
	ReplacementPlanIDs []int64 `json:"replacement_plan_ids"`
}
type subscriptionRenewalSnapshot struct {
	ActivationMillis int64   `json:"activation_unix_ms"`
	LegacyGroupIDs   []int64 `json:"legacy_group_ids"`
}
type ScheduledSubscriptionRenewal struct {
	OrderID      int64     `json:"order_id"`
	PlanID       int64     `json:"plan_id"`
	ActivationAt time.Time `json:"activation_at"`
}
type SubscriptionRenewalInfo struct {
	ActivationAt   *time.Time                     `json:"activation_at,omitempty"`
	LegacyGroupIDs []int64                        `json:"legacy_group_ids"`
	Scheduled      []ScheduledSubscriptionRenewal `json:"scheduled"`
}

func parseSubscriptionRenewalPolicy(raw string) *SubscriptionRenewalPolicy {
	if raw == "" {
		return nil
	}
	var p SubscriptionRenewalPolicy
	if json.Unmarshal([]byte(raw), &p) != nil {
		return &SubscriptionRenewalPolicy{Enabled: true}
	}
	return &p
}
func validateSubscriptionRenewalPolicy(p *SubscriptionRenewalPolicy) error {
	if p == nil || !p.Enabled {
		return nil
	}
	if len(p.LegacyPlanIDs) == 0 || len(p.ReplacementPlanIDs) == 0 || len(p.LegacyPlanIDs) > 100 || len(p.ReplacementPlanIDs) > 100 {
		return infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "select legacy and replacement subscription plans")
	}
	seen := map[int64]bool{}
	for _, ids := range [][]int64{p.LegacyPlanIDs, p.ReplacementPlanIDs} {
		for _, id := range ids {
			if id <= 0 || seen[id] {
				return infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "plan IDs must be positive, unique and disjoint")
			}
			seen[id] = true
		}
	}
	return nil
}
func containsPlan(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
func (s *PaymentConfigService) renewalGroups(ctx context.Context, p *SubscriptionRenewalPolicy) ([]int64, error) {
	if err := validateSubscriptionRenewalPolicy(p); err != nil {
		return nil, err
	}
	if p == nil || !p.Enabled {
		return nil, nil
	}
	ids := append(append([]int64{}, p.LegacyPlanIDs...), p.ReplacementPlanIDs...)
	plans, err := s.entClient.SubscriptionPlan.Query().Where(subscriptionplan.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(plans) != len(ids) {
		return nil, infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "configured renewal plan is missing")
	}
	legacy := map[int64]bool{}
	current := map[int64]bool{}
	for _, p1 := range plans {
		if containsPlan(p.LegacyPlanIDs, p1.ID) {
			legacy[p1.GroupID] = true
		} else {
			current[p1.GroupID] = true
		}
	}
	groups := make([]int64, 0, len(legacy))
	for id := range legacy {
		if current[id] {
			return nil, infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "legacy and replacement plans must use separate quota groups")
		}
		groups = append(groups, id)
	}
	groupIDs := make([]int64, 0, len(plans))
	for _, plan := range plans {
		groupIDs = append(groupIDs, plan.GroupID)
	}
	quotaGroups, err := s.entClient.Group.Query().Where(group.IDIn(groupIDs...)).All(ctx)
	if err != nil {
		return nil, err
	}
	found := map[int64]bool{}
	platform := ""
	for _, g := range quotaGroups {
		found[g.ID] = true
		if g.SubscriptionType != "subscription" {
			return nil, infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "renewal products must use subscription groups")
		}
		if platform != "" && platform != g.Platform {
			return nil, infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "renewal products must use the same platform to preserve API keys")
		}
		platform = g.Platform
	}
	for _, id := range groupIDs {
		if !found[id] {
			return nil, infraerrors.BadRequest("INVALID_RENEWAL_POLICY", "configured subscription group is missing")
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i] < groups[j] })
	return groups, nil
}
func legacySubscriptionEnd(ctx context.Context, client *dbent.Client, uid int64, groups []int64, now time.Time) (time.Time, error) {
	if len(groups) == 0 {
		return now, nil
	}
	subs, err := client.UserSubscription.Query().Where(usersubscription.UserIDEQ(uid), usersubscription.GroupIDIn(groups...), usersubscription.StatusEQ(SubscriptionStatusActive), usersubscription.ExpiresAtGT(now)).All(ctx)
	if err != nil {
		return time.Time{}, err
	}
	end := now
	for _, sub := range subs {
		if sub.ExpiresAt.After(end) {
			end = sub.ExpiresAt
		}
	}
	return end, nil
}
func readSubscriptionRenewal(o *dbent.PaymentOrder) (*subscriptionRenewalSnapshot, error) {
	if o == nil || o.ProviderSnapshot == nil {
		return nil, nil
	}
	raw, ok := o.ProviderSnapshot[renewalSnapshotKey]
	if !ok {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var snap subscriptionRenewalSnapshot
	if err = json.Unmarshal(data, &snap); err != nil || snap.ActivationMillis <= 0 || len(snap.LegacyGroupIDs) == 0 {
		return nil, fmt.Errorf("invalid subscription renewal snapshot")
	}
	for _, id := range snap.LegacyGroupIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid legacy renewal group")
		}
	}
	return &snap, nil
}
func PaymentOrderScheduledActivation(o *dbent.PaymentOrder) *time.Time {
	snap, err := readSubscriptionRenewal(o)
	if err != nil || snap == nil || o.Status == OrderStatusCompleted {
		return nil
	}
	at := time.UnixMilli(snap.ActivationMillis).UTC()
	return &at
}
func (s *PaymentService) attachSubscriptionRenewal(ctx context.Context, tx *dbent.Tx, req CreateOrderRequest, cfg *PaymentConfig, snapshot map[string]any) (map[string]any, error) {
	p := cfg.SubscriptionRenewalPolicy
	if req.OrderType != "subscription" || p == nil || !p.Enabled {
		return snapshot, nil
	}
	if err := validateSubscriptionRenewalPolicy(p); err != nil {
		return nil, err
	}
	if !containsPlan(p.ReplacementPlanIDs, req.PlanID) {
		return nil, infraerrors.BadRequest("PLAN_RETIRED", "select a current subscription plan for renewal")
	}
	groups, err := s.configService.renewalGroups(ctx, p)
	if err != nil {
		return nil, err
	}
	if _, err = tx.User.Query().Where(user.IDEQ(req.UserID)).ForUpdate().Only(ctx); err != nil {
		return nil, err
	}
	end, err := legacySubscriptionEnd(ctx, tx.Client(), req.UserID, groups, time.Now())
	if err != nil {
		return nil, err
	}
	if !end.After(time.Now()) {
		return snapshot, nil
	}
	if snapshot == nil {
		snapshot = map[string]any{}
	}
	snapshot[renewalSnapshotKey] = subscriptionRenewalSnapshot{ActivationMillis: (end.UnixMilli() + 1), LegacyGroupIDs: groups}
	return snapshot, nil
}
func (s *PaymentService) SubscriptionRenewalInfo(ctx context.Context, uid int64, cfg *PaymentConfig) (*SubscriptionRenewalInfo, error) {
	p := cfg.SubscriptionRenewalPolicy
	info := &SubscriptionRenewalInfo{LegacyGroupIDs: []int64{}, Scheduled: []ScheduledSubscriptionRenewal{}}
	if p != nil && p.Enabled {
		groups, err := s.configService.renewalGroups(ctx, p)
		if err != nil {
			return nil, err
		}
		info.LegacyGroupIDs = groups
		now := time.Now()
		end, err := legacySubscriptionEnd(ctx, s.entClient, uid, groups, now)
		if err != nil {
			return nil, err
		}
		if end.After(now) {
			info.ActivationAt = &end
		}
	}
	orders, err := s.entClient.PaymentOrder.Query().Where(paymentorder.UserIDEQ(uid), paymentorder.StatusIn(OrderStatusScheduled, OrderStatusPaid), paymentorder.PaidAtNotNil(), paymentorder.OrderTypeEQ("subscription"), func(sel *sql.Selector) {
		sel.Where(sqljson.HasKey(paymentorder.FieldProviderSnapshot, sqljson.Path(renewalSnapshotKey)))
	}).Order(dbent.Asc(paymentorder.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		at := PaymentOrderScheduledActivation(o)
		if at != nil && o.PlanID != nil {
			info.Scheduled = append(info.Scheduled, ScheduledSubscriptionRenewal{OrderID: o.ID, PlanID: *o.PlanID, ActivationAt: *at})
		}
	}
	return info, nil
}

// Snapshot errors fail closed. A later administrative extension of the original
// entitlement postpones activation as well; it must not silently discard time.
func (s *PaymentService) deferSubscriptionRenewal(ctx context.Context, o *dbent.PaymentOrder) (bool, error) {
	snap, err := readSubscriptionRenewal(o)
	if err != nil || snap == nil {
		return false, err
	}
	if o.PaidAt == nil {
		return false, infraerrors.BadRequest("INVALID_STATUS", "scheduled subscription order is not paid")
	}
	assigned, err := hasPaymentSubscriptionAssignmentAudit(ctx, s.entClient, o.ID)
	if err != nil {
		return false, err
	}
	if assigned {
		return false, nil
	}
	now := time.Now()
	end, err := legacySubscriptionEnd(ctx, s.entClient, o.UserID, snap.LegacyGroupIDs, now)
	if err != nil {
		return false, err
	}
	if end.After(now) && end.UnixMilli()+1 > snap.ActivationMillis {
		snap.ActivationMillis = end.UnixMilli() + 1
	}
	if snap.ActivationMillis <= now.UnixMilli() {
		return false, nil
	}
	updated := make(map[string]any, len(o.ProviderSnapshot))
	for k, v := range o.ProviderSnapshot {
		updated[k] = v
	}
	updated[renewalSnapshotKey] = snap
	_, err = s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(o.ID), paymentorder.StatusIn(OrderStatusPaid, OrderStatusScheduled, OrderStatusFailed)).SetStatus(OrderStatusScheduled).SetProviderSnapshot(updated).Save(ctx)
	return true, err
}
func (s *PaymentService) ActivateDueSubscriptionRenewals(ctx context.Context) (int, error) {
	now := time.Now()
	orders, err := s.entClient.PaymentOrder.Query().Where(paymentorder.Or(paymentorder.StatusIn(OrderStatusScheduled, OrderStatusPaid, OrderStatusFailed), paymentorder.And(paymentorder.StatusEQ(OrderStatusRecharging), paymentorder.UpdatedAtLTE(now.Add(-paymentFulfillmentLeaseDuration)))), paymentorder.PaidAtNotNil(), paymentorder.OrderTypeEQ("subscription"), func(sel *sql.Selector) {
		sel.Where(sqljson.ValueLTE(paymentorder.FieldProviderSnapshot, now.UnixMilli(), sqljson.Path(renewalSnapshotKey, "activation_unix_ms"), sqljson.Cast("bigint")))
	}).Order(dbent.Asc(paymentorder.FieldUpdatedAt), dbent.Asc(paymentorder.FieldID)).Limit(100).All(ctx)
	if err != nil {
		return 0, err
	}
	done := 0
	var failures []error
	for _, o := range orders {
		if err := s.ExecuteSubscriptionFulfillment(ctx, o.ID); err != nil {
			failures = append(failures, fmt.Errorf("order %d: %w", o.ID, err))
			continue
		}
		done++
	}
	return done, errors.Join(failures...)
}

func (s *PaymentService) moveLegacySubscriptionKeys(ctx context.Context, client *dbent.Client, o *dbent.PaymentOrder, gid int64) error {
	snap, err := readSubscriptionRenewal(o)
	if err != nil || snap == nil {
		return err
	}
	// Only this owner's keys still attached to retired groups move. Secrets,
	// expiry, key quotas, IP rules and permissions remain intact.
	_, err = client.APIKey.Update().Where(apikey.UserIDEQ(o.UserID), apikey.GroupIDIn(snap.LegacyGroupIDs...)).SetGroupID(gid).Save(ctx)
	return err
}

var errSubscriptionRenewalDeferred = errors.New("grandfathered subscription is still active")

func (s *PaymentService) lockRenewalSource(ctx context.Context, client *dbent.Client, o *dbent.PaymentOrder) error {
	snap, err := readSubscriptionRenewal(o)
	if err != nil || snap == nil {
		return err
	}
	if _, err = client.User.Query().Where(user.IDEQ(o.UserID)).ForUpdate().Only(ctx); err != nil {
		return err
	}
	sources, err := client.UserSubscription.Query().Where(usersubscription.UserIDEQ(o.UserID), usersubscription.GroupIDIn(snap.LegacyGroupIDs...)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	for _, source := range sources {
		if source.Status == SubscriptionStatusActive && source.ExpiresAt.After(time.Now()) {
			return errSubscriptionRenewalDeferred
		}
	}
	return nil
}
func (s *PaymentService) invalidateRenewalKeys(ctx context.Context, uid int64) {
	if s.redeemService != nil && s.redeemService.authCacheInvalidator != nil {
		s.redeemService.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, uid)
	}
}

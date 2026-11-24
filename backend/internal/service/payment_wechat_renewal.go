package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/payment/provider"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const wechatRenewalSnapshotKey = "wechat_renewal"
const wechatRenewalWebhookBase = "/api/v1/payment/webhook/wechat-renewal/"

type wechatRenewalConfig struct {
	client                    *provider.WeChatRecurringClient
	appID, merchantID, origin string
	templates                 map[string]string
}

func loadWeChatRenewalConfig() *wechatRenewalConfig {
	origin, err := url.Parse(os.Getenv("WECHAT_RENEWAL_ORIGIN"))
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
		return nil
	}
	appID, merchantID := os.Getenv("WECHAT_RENEWAL_APP_ID"), os.Getenv("WECHAT_RENEWAL_MERCHANT_ID")
	client, err := provider.NewWeChatRecurringClient(appID, merchantID, os.Getenv("WECHAT_RENEWAL_API_V2_KEY"))
	if err != nil {
		return nil
	}
	var templates map[string]string
	if err := json.Unmarshal([]byte(os.Getenv("WECHAT_RENEWAL_PLAN_TEMPLATES")), &templates); err != nil {
		return nil
	}
	for planID, templateID := range templates {
		pid, err := strconv.ParseInt(planID, 10, 64)
		tid, tidErr := strconv.ParseInt(templateID, 10, 64)
		if err != nil || tidErr != nil || pid <= 0 || tid <= 0 {
			return nil
		}
	}
	if len(templates) == 0 {
		return nil
	}
	return &wechatRenewalConfig{client: client, appID: appID, merchantID: merchantID, templates: templates, origin: strings.TrimRight(origin.String(), "/")}
}

func (s *PaymentService) WeChatRenewalPlanIDs() []int64 {
	if s == nil || s.wechatRenewal == nil {
		return nil
	}
	var ids []int64
	for key := range s.wechatRenewal.templates {
		id, _ := strconv.ParseInt(key, 10, 64)
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (s *PaymentService) validateWeChatRenewalCheckout(req CreateOrderRequest, cfg *PaymentConfig, sel *payment.InstanceSelection) error {
	if !req.WeChatAutoRenew {
		return nil
	}
	r := s.wechatRenewal
	if r == nil || !cfg.BalanceDisabled || req.OrderType != payment.OrderTypeSubscription || r.templates[strconv.FormatInt(req.PlanID, 10)] == "" ||
		sel == nil || sel.ProviderKey != payment.TypeWxpay || r.merchantID != sel.Config["mchId"] ||
		(!req.IsMobile && !req.IsWeChatBrowser) {
		return infraerrors.BadRequest("WECHAT_RENEWAL_UNAVAILABLE", "WeChat automatic renewal is unavailable for this checkout")
	}
	appID := paymentOrderSnapshotWxpayAppID(sel, req)
	if req.IsWeChatBrowser {
		appID = provider.ResolveWxpayJSAPIAppID(sel.Config)
	}
	if r.appID != appID {
		return infraerrors.BadRequest("WECHAT_RENEWAL_UNAVAILABLE", "WeChat automatic renewal app is unavailable")
	}
	return nil
}

func isWeChatRenewalOrder(order *dbent.PaymentOrder) bool {
	marked, _ := order.ProviderSnapshot[wechatRenewalSnapshotKey].(bool)
	return marked
}

func wechatPayerHash(openID string) string {
	hash := sha256.Sum256([]byte(openID))
	return hex.EncodeToString(hash[:])
}

func wechatRenewalContractCode(id int64) string { return "sub9" + strconv.FormatInt(id, 10) }

func (s *PaymentService) createWeChatRenewalContract(ctx context.Context, tx *dbent.Tx, order *dbent.PaymentOrder, req CreateOrderRequest) error {
	if !req.WeChatAutoRenew {
		return nil
	}
	template := s.wechatRenewal.templates[strconv.FormatInt(req.PlanID, 10)]
	// Retire checkouts with no recorded payment or signature after their window.
	// Late signatures for retired orders are explicitly terminated by the callback.
	if _, err := tx.Client().ExecContext(ctx, `UPDATE wechat_renewal_contracts c SET state='cancelled', updated_at=now()
  FROM payment_orders o WHERE o.id=c.initial_order_id AND c.user_id=$1 AND c.group_id=$2
  AND c.state='pending' AND c.contract_id IS NULL AND o.status IN ('EXPIRED','CANCELLED','FAILED') AND o.paid_at IS NULL`, order.UserID, *order.SubscriptionGroupID); err != nil {
		return err
	}
	result, err := tx.Client().ExecContext(ctx, `INSERT INTO wechat_renewal_contracts(initial_order_id,user_id,group_id,template_id,contract_code,state)
 VALUES($1,$2,$3,$4,$5,'pending') ON CONFLICT DO NOTHING`, order.ID, order.UserID, *order.SubscriptionGroupID, template, wechatRenewalContractCode(order.ID))
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return infraerrors.Conflict("WECHAT_RENEWAL_ALREADY_EXISTS", "A WeChat renewal agreement already exists for this subscription")
	}
	return nil
}

func (s *PaymentService) startWeChatRenewalCheckout(ctx context.Context, order *dbent.PaymentOrder, req CreateOrderRequest, body, returnURL string) (*payment.CreatePaymentResponse, error) {
	r := s.wechatRenewal
	openID := ""
	if req.IsWeChatBrowser {
		openID = req.OpenID
	}
	response, err := r.client.Checkout(ctx, provider.WeChatContractCheckout{
		OrderNo: order.OutTradeNo, ContractCode: wechatRenewalContractCode(order.ID), Serial: order.ID,
		TemplateID: r.templates[strconv.FormatInt(req.PlanID, 10)], Body: body, AmountFen: int64(math.Round(order.PayAmount * 100)),
		ClientIP: req.ClientIP, OpenID: openID, ExpiresAt: order.ExpiresAt,
		PaymentNotifyURL: r.origin + wechatRenewalWebhookBase + "payment", ContractNotifyURL: r.origin + wechatRenewalWebhookBase + "contract",
	})
	if err != nil {
		return nil, err
	}
	if response.PayURL != "" && returnURL != "" {
		redirect, _ := url.Parse(response.PayURL)
		query := redirect.Query()
		query.Set("redirect_url", returnURL)
		redirect.RawQuery = query.Encode()
		response.PayURL = redirect.String()
	}
	return response, nil
}

func (s *PaymentService) HandleWeChatRenewalNotification(ctx context.Context, kind string, body io.Reader) error {
	if s.wechatRenewal == nil {
		return errors.New("WeChat renewal is disabled")
	}
	fields, err := s.wechatRenewal.client.VerifyNotification(body)
	if err != nil {
		return err
	}
	switch kind {
	case "contract":
		return s.applyWeChatRenewalContract(ctx, fields)
	case "payment":
		return s.applyWeChatRenewalPayment(ctx, fields)
	default:
		return errors.New("unknown WeChat renewal callback")
	}
}

func (s *PaymentService) applyWeChatRenewalContract(ctx context.Context, fields map[string]string) error {
	code, contractID := fields["contract_code"], fields["contract_id"]
	if fields["result_code"] != "SUCCESS" || code == "" || contractID == "" || fields["plan_id"] == "" {
		return errors.New("invalid WeChat contract event")
	}
	switch fields["change_type"] {
	case "ADD":
		if fields["openid"] == "" {
			return errors.New("missing WeChat signer")
		}
		if serial := fields["request_serial"]; serial != "" && "sub9"+serial != code {
			return errors.New("WeChat signing serial mismatch")
		}
		result, err := s.entClient.ExecContext(ctx, `UPDATE wechat_renewal_contracts SET contract_id=$1,signed_payer_hash=$4,updated_at=now()
   WHERE contract_code=$2 AND template_id=$3 AND state IN ('pending','active') AND (contract_id IS NULL OR (contract_id=$1 AND signed_payer_hash=$4))`, contractID, code, fields["plan_id"], wechatPayerHash(fields["openid"]))
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows > 0 {
			return err
		}
		// A late ADD can arrive after the customer cancelled or a checkout expired.
		var retired bool
		if err := wechatQueryRow(ctx, s.entClient, `SELECT EXISTS(SELECT 1 FROM wechat_renewal_contracts WHERE contract_code=$1 AND template_id=$2 AND state='cancelled')`, code, fields["plan_id"]).scan(&retired); err != nil {
			return err
		}
		if retired {
			return s.wechatRenewal.client.CancelContract(ctx, contractID)
		}
		return errors.New("unknown or mismatched WeChat contract")
	case "DELETE":
		_, err := s.entClient.ExecContext(ctx, `UPDATE wechat_renewal_contracts SET state='cancelled',updated_at=now()
   WHERE contract_code=$1 AND template_id=$2 AND (contract_id=$3 OR contract_id IS NULL) AND state IN ('pending','active')`, code, fields["plan_id"], contractID)
		return err
	default:
		return errors.New("unknown WeChat contract change")
	}
}

func (s *PaymentService) applyWeChatRenewalPayment(ctx context.Context, fields map[string]string) error {
	order, err := s.entClient.PaymentOrder.Query().Where(paymentorder.OutTradeNoEQ(fields["out_trade_no"])).Only(ctx)
	if err != nil {
		return err
	}
	if !isWeChatRenewalOrder(order) {
		return errors.New("WeChat renewal order mismatch")
	}
	success := fields["result_code"] == "SUCCESS" && (fields["trade_state"] == "SUCCESS" || (fields["trade_state"] == "" && fields["transaction_id"] != ""))
	finalFailure := fields["trade_state"] == "PAYERROR" || fields["trade_state"] == "CLOSED" || fields["trade_state"] == "REVOKED"
	if fields["result_code"] == "FAIL" && fields["contract_id"] != "" && isFinalWeChatDebitFailure(fields["err_code"]) {
		finalFailure = true
		fields["trade_state"] = fields["err_code"]
	}
	if !success && !finalFailure {
		return nil
	}
	if success {
		fen, err := strconv.ParseInt(fields["total_fee"], 10, 64)
		if err != nil || fen != int64(math.Round(order.PayAmount*100)) || fields["transaction_id"] == "" || (fields["fee_type"] != "" && fields["fee_type"] != "CNY") {
			return errors.New("WeChat renewal payment amount or transaction mismatch")
		}
	}
	var sourceID int64
	err = wechatQueryRow(ctx, s.entClient, `SELECT initial_order_id FROM wechat_renewal_cycles WHERE order_id=$1`, order.ID).scan(&sourceID)
	if errors.Is(err, sql.ErrNoRows) {
		if !success {
			return nil
		}
		if fields["openid"] == "" {
			return errors.New("missing first-payment WeChat payer")
		}
		result, err := s.entClient.ExecContext(ctx, `UPDATE wechat_renewal_contracts SET paid_payer_hash=$2 WHERE initial_order_id=$1 AND (paid_payer_hash IS NULL OR paid_payer_hash=$2)`, order.ID, wechatPayerHash(fields["openid"]))
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return errors.New("unrecognized first-payment WeChat payer")
		}
		return s.confirmPayment(ctx, order.ID, fields["transaction_id"], order.PayAmount, payment.TypeWxpay, map[string]string{"appid": fields["appid"], "mchid": fields["mch_id"], "currency": "CNY"})
	}
	if err != nil {
		return err
	}
	return s.settleWeChatRenewalCycle(ctx, order, sourceID, fields, success)
}

func (s *PaymentService) settleWeChatRenewalCycle(ctx context.Context, order *dbent.PaymentOrder, sourceID int64, fields map[string]string, success bool) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	db := tx.Client()
	var contractID, contractState, cycleState string
	var periodEnd time.Time
	if err := wechatQueryRow(ctx, db, `SELECT contract_id,state FROM wechat_renewal_contracts WHERE initial_order_id=$1 FOR UPDATE`, sourceID).scan(&contractID, &contractState); err != nil {
		return err
	}
	if fields["contract_id"] != "" && fields["contract_id"] != contractID {
		return errors.New("WeChat renewal agreement mismatch")
	}
	if err := wechatQueryRow(ctx, db, `SELECT state,period_end FROM wechat_renewal_cycles WHERE order_id=$1 FOR UPDATE`, order.ID).scan(&cycleState, &periodEnd); err != nil {
		return err
	}
	if cycleState == "completed" || cycleState == "failed" {
		if cycleState == "failed" && success {
			return errors.New("WeChat returned a payment after a terminal failure")
		}
		if cycleState == "failed" {
			_ = tx.Rollback()
			return s.subscriptionSvc.invalidateSubscriptionCaches(order.UserID, *order.SubscriptionGroupID)
		}
		return nil
	}
	if success {
		// Use the same entitlement assignment and auditing as a one-time purchase.
		// The existing fulfillment lease and assignment audit deduplicate callbacks.
		if err := s.confirmPayment(ctx, order.ID, fields["transaction_id"], order.PayAmount, payment.TypeWxpay, map[string]string{"appid": fields["appid"], "mchid": fields["mch_id"], "currency": "CNY"}); err != nil {
			return err
		}
		latest, err := s.entClient.PaymentOrder.Get(ctx, order.ID)
		if err != nil {
			return err
		}
		if latest.Status != OrderStatusCompleted {
			return errors.New("WeChat renewal fulfillment pending")
		}
		if _, err := db.ExecContext(ctx, `UPDATE wechat_renewal_cycles SET state='completed' WHERE order_id=$1`, order.ID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `UPDATE wechat_renewal_contracts SET period_end=NULL,updated_at=now() WHERE initial_order_id=$1`, sourceID); err != nil {
			return err
		}
	} else {
		latest, err := db.PaymentOrder.Get(ctx, order.ID)
		if err != nil {
			return err
		}
		if latest.PaidAt != nil {
			return errors.New("WeChat terminal failure conflicts with confirmed payment")
		}
		if _, err := db.ExecContext(ctx, `UPDATE wechat_renewal_cycles SET state='failed'  WHERE order_id=$1`, order.ID); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `UPDATE payment_orders SET status='FAILED',failed_at=now(),failed_reason='WeChat renewal debit failed',updated_at=now() WHERE id=$1 AND status='PENDING'`, order.ID); err != nil {
			return err
		}
		if contractState == "active" {
			if _, err := db.ExecContext(ctx, `UPDATE wechat_renewal_contracts SET state='failed',failure_code=$2,updated_at=now() WHERE initial_order_id=$1`, sourceID, fields["trade_state"]); err != nil {
				return err
			}
			// Preserve a separately purchased or manually extended entitlement.
			if _, err := db.ExecContext(ctx, `UPDATE user_subscriptions SET status=$4,updated_at=now() WHERE user_id=$1 AND group_id=$2 AND expires_at=$3 AND deleted_at IS NULL AND status='active'`, order.UserID, *order.SubscriptionGroupID, periodEnd, SubscriptionStatusExpired); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if !success {
		s.invalidateRenewalKeys(ctx, order.UserID)
		return s.subscriptionSvc.invalidateSubscriptionCaches(order.UserID, *order.SubscriptionGroupID)
	}
	return nil
}

type wechatRow struct {
	rows *sql.Rows
	err  error
}

func wechatQueryRow(ctx context.Context, db *dbent.Client, query string, args ...any) wechatRow {
	rows, err := db.QueryContext(ctx, query, args...)
	return wechatRow{rows, err}
}
func (row wechatRow) scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	defer func() { _ = row.rows.Close() }()
	if !row.rows.Next() {
		if err := row.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return row.rows.Scan(dest...)
}

func (s *PaymentService) reconcileWeChatRenewalOrder(ctx context.Context, order *dbent.PaymentOrder) string {
	if s.wechatRenewal == nil {
		return ""
	}
	fields, err := s.wechatRenewal.client.QueryDebit(ctx, order.OutTradeNo)
	if err != nil {
		return ""
	}
	if err := s.applyWeChatRenewalPayment(ctx, fields); err != nil {
		slog.Warn("WeChat renewal reconciliation failed", "order", order.ID, "error", err)
		return ""
	}
	if fields["trade_state"] == "SUCCESS" {
		return checkPaidResultAlreadyPaid
	}
	return ""
}

func (s *PaymentService) requireWeChatRenewal() error {
	if s.wechatRenewal == nil {
		return infraerrors.ServiceUnavailable("WECHAT_RENEWAL_UNAVAILABLE", "WeChat automatic renewal is unavailable")
	}
	return nil
}

func (s *PaymentService) CancelWeChatRenewal(ctx context.Context, id, userID int64) error {
	if err := s.requireWeChatRenewal(); err != nil {
		return err
	}
	var contractID sql.NullString
	if err := wechatQueryRow(ctx, s.entClient, `UPDATE wechat_renewal_contracts SET state='cancelled',updated_at=now()
  WHERE initial_order_id=$1 AND ($2=0 OR user_id=$2) RETURNING contract_id`, id, userID).scan(&contractID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return infraerrors.NotFound("NOT_FOUND", "WeChat agreement not found")
		}
		return err
	}
	if contractID.Valid {
		return s.wechatRenewal.client.CancelContract(ctx, contractID.String)
	}
	return nil
}

func (s *PaymentService) RunWeChatRenewals(ctx context.Context) error {
	if s.wechatRenewal == nil {
		return nil
	}
	// Only signed agreements with fulfilled first payments can become billable.
	if _, err := s.entClient.ExecContext(ctx, `UPDATE wechat_renewal_contracts c SET state='active', period_end=u.expires_at,updated_at=now()
  FROM payment_orders o,user_subscriptions u WHERE c.initial_order_id=o.id AND c.user_id=u.user_id AND c.group_id=u.group_id
  AND o.status='COMPLETED' AND u.deleted_at IS NULL AND u.status='active' AND c.contract_id IS NOT NULL
  AND c.signed_payer_hash=c.paid_payer_hash
  AND c.state IN ('pending','active') AND c.period_end IS NULL`); err != nil {
		return err
	}
	if err := s.recoverWeChatAgreement(ctx); err != nil {
		slog.Warn("WeChat agreement reconciliation failed", "error", err)
	}
	if err := s.enqueueWeChatRenewal(ctx); err != nil {
		return err
	}
	var id int64
	err := wechatQueryRow(ctx, s.entClient, `UPDATE wechat_renewal_cycles SET next_check_at=now()+interval '1 minute'
  WHERE order_id=(SELECT order_id FROM wechat_renewal_cycles WHERE state IN ('queued','processing') AND next_check_at<=now()
  ORDER BY next_check_at,order_id LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING order_id`).scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.processWeChatRenewalCycle(ctx, id)
}

func (s *PaymentService) enqueueWeChatRenewal(ctx context.Context) error {
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled || !cfg.BalanceDisabled {
		return nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	db := tx.Client()
	var sourceID int64
	var periodEnd time.Time
	// The merchant template sends notice immediately and normally debits after 24 hours.
	// A five-minute margin covers reconciliation before the paid term ends.
	err = wechatQueryRow(ctx, db, `SELECT c.initial_order_id,u.expires_at FROM wechat_renewal_contracts c
  JOIN user_subscriptions u ON u.user_id=c.user_id AND u.group_id=c.group_id AND u.deleted_at IS NULL
  JOIN users customer ON customer.id=c.user_id AND customer.status='active' AND customer.deleted_at IS NULL
  WHERE c.state='active' AND u.status='active' AND u.expires_at>now() AND u.expires_at<=now()+interval '24 hours 5 minutes'
  AND NOT EXISTS (SELECT 1 FROM wechat_renewal_cycles w WHERE w.initial_order_id=c.initial_order_id AND (w.period_end=u.expires_at OR w.state IN ('queued','processing')))
  ORDER BY u.expires_at LIMIT 1 FOR UPDATE OF c SKIP LOCKED`).scan(&sourceID, &periodEnd)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	initial, err := db.PaymentOrder.Get(ctx, sourceID)
	if err != nil {
		return err
	}
	if initial.SubscriptionDays == nil || initial.PlanID == nil || initial.SubscriptionGroupID == nil {
		return errors.New("incomplete WeChat renewal source order")
	}
	no, err := s.allocateOutTradeNo(ctx, tx)
	if err != nil {
		return err
	}
	snapshot := map[string]any{}
	for k, v := range initial.ProviderSnapshot {
		snapshot[k] = v
	}
	delete(snapshot, "subscription_renewal")
	cycle, err := db.PaymentOrder.Create().SetUserID(initial.UserID).SetUserName(initial.UserName).SetUserEmail(initial.UserEmail).
		SetClientIP(initial.ClientIP).SetSrcHost(initial.SrcHost).SetRechargeCode("").SetPaymentTradeNo("").
		SetOutTradeNo(no).SetAmount(initial.Amount).SetPayAmount(initial.PayAmount).SetFeeRate(initial.FeeRate).
		SetOrderType(payment.OrderTypeSubscription).SetPaymentType(initial.PaymentType).SetProviderKey(payment.TypeWxpay).
		SetNillableProviderInstanceID(initial.ProviderInstanceID).
		SetProviderSnapshot(snapshot).SetStatus(OrderStatusPending).SetExpiresAt(periodEnd.Add(48 * time.Hour)).
		SetPlanID(*initial.PlanID).SetSubscriptionGroupID(*initial.SubscriptionGroupID).SetSubscriptionDays(*initial.SubscriptionDays).Save(ctx)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO wechat_renewal_cycles(order_id,initial_order_id,period_end,state) VALUES($1,$2,$3,'queued')`, cycle.ID, sourceID, periodEnd); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PaymentService) processWeChatRenewalCycle(ctx context.Context, id int64) error {
	var contractID, contractState, cycleState string
	if err := wechatQueryRow(ctx, s.entClient, `SELECT c.contract_id,c.state,w.state FROM wechat_renewal_cycles w JOIN wechat_renewal_contracts c ON c.initial_order_id=w.initial_order_id WHERE w.order_id=$1`, id).scan(&contractID, &contractState, &cycleState); err != nil {
		return err
	}
	order, err := s.entClient.PaymentOrder.Get(ctx, id)
	if err != nil {
		return err
	}
	if cycleState == "processing" {
		fields, err := s.wechatRenewal.client.QueryDebit(ctx, order.OutTradeNo)
		if err == nil {
			return s.applyWeChatRenewalPayment(ctx, fields)
		}
		var papErr *provider.WeChatPAPError
		if !errors.As(err, &papErr) || papErr.Code != "ORDERNOTEXIST" {
			return err
		}
		// Reuse the original merchant order number after transport uncertainty.
	} else if cycleState != "queued" {
		return nil
	}
	if contractState != "active" {
		if cycleState == "queued" {
			return s.applyWeChatRenewalPayment(ctx, map[string]string{"out_trade_no": order.OutTradeNo, "trade_state": "CLOSED"})
		}
		return nil
	}
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled || !cfg.BalanceDisabled {
		return nil
	}
	// Synchronize debit submission with web and WeChat cancellation. Persist the
	// processing state first so an ambiguous reply is always recovered by query.
	result, err := s.entClient.ExecContext(ctx, `UPDATE wechat_renewal_cycles SET state='processing' WHERE order_id=$1 AND state IN ('queued','processing')`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return err
	}
	err = s.submitWeChatRenewalDebit(ctx, order, contractID)
	if err != nil {
		var rejection *provider.WeChatPAPError
		if errors.As(err, &rejection) && isFinalWeChatDebitFailure(rejection.Code) {
			return s.applyWeChatRenewalPayment(ctx, map[string]string{"out_trade_no": order.OutTradeNo, "contract_id": contractID, "result_code": "FAIL", "err_code": rejection.Code})
		}
		return fmt.Errorf("request WeChat renewal order %d: %w", id, err)
	}
	return nil
}

// WeChatRenewalAgreement is safe to display to users and payment administrators.
// Merchant agreement identifiers and signing data stay server-side.
type WeChatRenewalAgreement struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	PlanID      int64      `json:"plan_id"`
	State       string     `json:"state"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`
	FailureCode string     `json:"failure_code,omitempty"`
	Processing  bool       `json:"processing"`
}

func (s *PaymentService) ListWeChatRenewals(ctx context.Context, userID int64, page int) ([]WeChatRenewalAgreement, error) {
	if page < 1 {
		page = 1
	}
	if page > 1000000 {
		page = 1000000
	}
	rows, err := s.entClient.QueryContext(ctx, `SELECT c.initial_order_id,c.user_id,o.plan_id,c.state,c.period_end,c.failure_code,
 EXISTS(SELECT 1 FROM wechat_renewal_cycles w WHERE w.initial_order_id=c.initial_order_id AND w.state IN ('queued','processing'))
 FROM wechat_renewal_contracts c JOIN payment_orders o ON o.id=c.initial_order_id WHERE $1=0 OR c.user_id=$1
 ORDER BY c.initial_order_id DESC LIMIT 50 OFFSET $2`, userID, int64(page-1)*50)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make([]WeChatRenewalAgreement, 0)
	for rows.Next() {
		var row WeChatRenewalAgreement
		if err := rows.Scan(&row.ID, &row.UserID, &row.PlanID, &row.State, &row.PeriodEnd, &row.FailureCode, &row.Processing); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func isFinalWeChatDebitFailure(code string) bool {
	switch code {
	case "ACCOUNTERROR", "CONTRACT_NOT_EXIST", "RULELIMIT", "BANKERROR", "NOTENOUGH", "USER_ACCOUNT_ABNORMAL", "USER_NOT_EXIST", "TRADE_ERROR", "USER_ERROR":
		return true
	default:
		return false
	}
}

func (s *PaymentService) submitWeChatRenewalDebit(ctx context.Context, order *dbent.PaymentOrder, contractID string) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var state string
	err = wechatQueryRow(ctx, tx.Client(), `SELECT c.state FROM wechat_renewal_cycles w JOIN wechat_renewal_contracts c ON c.initial_order_id=w.initial_order_id WHERE w.order_id=$1 AND w.state='processing' FOR UPDATE OF c`, order.ID).scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if state != "active" {
		return nil
	}
	return s.wechatRenewal.client.RequestDebit(ctx, order.OutTradeNo, contractID, "Subscription renewal", s.wechatRenewal.origin+wechatRenewalWebhookBase+"payment", int64(math.Round(order.PayAmount*100)))
}

func (s *PaymentService) expireWeChatInitialCheckout(ctx context.Context, order *dbent.PaymentOrder) (string, error) {
	if s.wechatRenewal == nil {
		return "", nil
	}
	var initial bool
	if err := wechatQueryRow(ctx, s.entClient, `SELECT EXISTS(SELECT 1 FROM wechat_renewal_contracts WHERE initial_order_id=$1)`, order.ID).scan(&initial); err != nil || !initial {
		return "", err
	}
	fields, err := s.wechatRenewal.client.QueryDebit(ctx, order.OutTradeNo)
	var missing *provider.WeChatPAPError
	if err != nil && (!errors.As(err, &missing) || missing.Code != "ORDERNOTEXIST") {
		return "", err
	}
	if err == nil && fields["trade_state"] != "CLOSED" && fields["trade_state"] != "PAYERROR" && fields["trade_state"] != "REVOKED" {
		return "", nil
	}
	count, err := s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(order.ID), paymentorder.StatusEQ(OrderStatusPending)).SetStatus(OrderStatusExpired).Save(ctx)
	if err != nil || count != 1 {
		return "", err
	}
	return checkPaidResultCancelled, nil
}

func (s *PaymentService) recoverWeChatAgreement(ctx context.Context) error {
	var code, template string
	// Rotate through completed checkouts whose signing callback may have been lost.
	err := wechatQueryRow(ctx, s.entClient, `UPDATE wechat_renewal_contracts SET updated_at=now()
 WHERE initial_order_id=(SELECT c.initial_order_id FROM wechat_renewal_contracts c JOIN payment_orders o ON o.id=c.initial_order_id
  WHERE c.state='pending' AND c.contract_id IS NULL AND o.status='COMPLETED' AND c.updated_at<now()-interval '1 minute'
  ORDER BY c.updated_at LIMIT 1 FOR UPDATE OF c SKIP LOCKED) RETURNING contract_code,template_id`).scan(&code, &template)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	fields, err := s.wechatRenewal.client.QueryContract(ctx, template, code)
	if err != nil {
		return err
	}
	if fields["contract_state"] != "0" {
		return nil
	}
	if fields["contract_code"] != code || fields["plan_id"] != template {
		return errors.New("WeChat agreement query mismatch")
	}
	fields["change_type"] = "ADD"
	return s.applyWeChatRenewalContract(ctx, fields)
}

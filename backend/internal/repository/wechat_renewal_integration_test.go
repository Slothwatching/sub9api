//go:build integration

package repository

import (
	"context"
	"crypto/md5" // #nosec G501 -- WeChat Pay v2 fixture signing.
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestWeChatRenewalPaymentSettlement(t *testing.T) {
	const key = "0123456789abcdef0123456789abcdef"
	t.Setenv("WECHAT_RENEWAL_ORIGIN", "https://checkout.example.test")
	t.Setenv("WECHAT_RENEWAL_APP_ID", "wx-test")
	t.Setenv("WECHAT_RENEWAL_MERCHANT_ID", "merchant-test")
	t.Setenv("WECHAT_RENEWAL_API_V2_KEY", key)
	t.Setenv("WECHAT_RENEWAL_PLAN_TEMPLATES", `{"1":"123"}`)
	ctx := t.Context()
	client := integrationEntClient
	groups := NewGroupRepository(client, integrationDB)
	subs := NewUserSubscriptionRepository(client)
	subSvc := service.NewSubscriptionService(groups, subs, nil, client, nil)
	t.Cleanup(subSvc.Stop)
	svc := service.NewPaymentService(client, nil, nil, nil, subSvc, nil, nil, groups, nil)
	for _, final := range []string{"NOTENOUGH", "PAY_FAIL", "SUCCESS"} {
		t.Run(final, func(t *testing.T) {
			stamp := fmt.Sprint(time.Now().UnixNano())
			user := client.User.Create().SetEmail("papay-" + stamp + "@example.test").SetPasswordHash("fixture").SaveX(ctx)
			group := client.Group.Create().SetName("papay-" + stamp).SetPlatform("openai").SetSubscriptionType("subscription").SaveX(ctx)
			expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
			client.UserSubscription.Create().SetUserID(user.ID).SetGroupID(group.ID).SetStartsAt(time.Now().Add(-time.Hour)).SetExpiresAt(expiry).SetStatus(service.SubscriptionStatusActive).SaveX(ctx)
			old := client.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName(user.Username).SetClientIP("127.0.0.1").SetSrcHost("test.local").SetRechargeCode("").SetPaymentTradeNo("").SetOutTradeNo("old" + stamp).SetPaymentType("wxpay").SetAmount(12).SetPayAmount(12).SetOrderType("subscription").SetPlanID(1).SetSubscriptionGroupID(group.ID).SetSubscriptionDays(30).SetStatus(service.OrderStatusCompleted).SetExpiresAt(expiry).SaveX(ctx)
			cycle := client.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName(user.Username).SetClientIP("127.0.0.1").SetSrcHost("test.local").SetRechargeCode("").SetPaymentTradeNo("").SetOutTradeNo("new" + stamp).SetAmount(12).SetPayAmount(12).SetPaymentType("wxpay").SetProviderKey("wxpay").SetProviderSnapshot(map[string]any{"wechat_renewal": true}).SetOrderType("subscription").SetPlanID(1).SetSubscriptionGroupID(group.ID).SetSubscriptionDays(30).SetStatus(service.OrderStatusPending).SetExpiresAt(expiry).SaveX(ctx)
			_, err := client.ExecContext(ctx, `INSERT INTO wechat_renewal_contracts(initial_order_id,user_id,group_id,template_id,contract_code,contract_id,state,period_end) VALUES ($1,$2,$3,'123',$4,$5,'active',$6)`, old.ID, user.ID, group.ID, "c"+stamp, "w"+stamp, expiry)
			require.NoError(t, err)
			_, err = client.ExecContext(ctx, `INSERT INTO wechat_renewal_cycles (order_id,initial_order_id,period_end,state) VALUES ($1,$2,$3,'processing')`, cycle.ID, old.ID, expiry)
			require.NoError(t, err)
			agreements, err := svc.ListWeChatRenewals(ctx, user.ID, 1)
			require.NoError(t, err)
			require.Len(t, agreements, 1)
			require.True(t, agreements[0].Processing)
			require.NotNil(t, agreements[0].NextDebitAt)
			require.WithinDuration(t, expiry.Add(-5*time.Minute), *agreements[0].NextDebitAt, time.Second)
			// Prime the authorization path before any callback.
			_, err = subSvc.GetActiveSubscription(ctx, user.ID, group.ID)
			require.NoError(t, err)
			send := func(fields map[string]string) error {
				fields["return_code"], fields["appid"], fields["mch_id"] = "SUCCESS", "wx-test", "merchant-test"
				fields["out_trade_no"], fields["contract_id"] = cycle.OutTradeNo, "w"+stamp
				names := make([]string, 0, len(fields))
				for name := range fields {
					names = append(names, name)
				}
				sort.Strings(names)
				var query, body strings.Builder
				_, _ = body.WriteString("<xml>")
				for _, name := range names {
					_, _ = fmt.Fprintf(&query, "%s=%s&", name, fields[name])
					_, _ = fmt.Fprintf(&body, "<%s>%s</%s>", name, fields[name], name)
				}
				_, _ = fmt.Fprintf(&body, "<sign>%X</sign></xml>", md5.Sum([]byte(query.String()+"key="+key))) // #nosec G401 -- WeChat Pay v2 fixture signing.
				return svc.HandleWeChatRenewalNotification(context.Background(), "payment", strings.NewReader(body.String()))
			}
			require.NoError(t, send(map[string]string{"result_code": "SUCCESS", "trade_state": "USERPAYING"}))
			require.NoError(t, send(map[string]string{"result_code": "SUCCESS", "trade_state": "ACCEPT"}))
			require.Equal(t, service.OrderStatusPending, client.PaymentOrder.GetX(ctx, cycle.ID).Status)
			if final == "NOTENOUGH" || final == "PAY_FAIL" {
				for range 2 {
					if final == "PAY_FAIL" {
						require.NoError(t, send(map[string]string{"result_code": "SUCCESS", "trade_state": "PAY_FAIL"}))
					} else {
						require.NoError(t, send(map[string]string{"result_code": "FAIL", "err_code": "NOTENOUGH"}))
					}
				}
				_, err = subSvc.GetActiveSubscription(ctx, user.ID, group.ID)
				require.ErrorIs(t, err, service.ErrSubscriptionNotFound)
				require.Equal(t, service.OrderStatusFailed, client.PaymentOrder.GetX(ctx, cycle.ID).Status)
			} else {
				require.Error(t, send(map[string]string{"result_code": "SUCCESS", "trade_state": "SUCCESS", "transaction_id": "tx" + stamp, "total_fee": "1201"}))
				for range 2 {
					require.NoError(t, send(map[string]string{"result_code": "SUCCESS", "trade_state": "SUCCESS", "transaction_id": "tx" + stamp, "total_fee": "1200"}))
				}
				require.Equal(t, service.OrderStatusCompleted, client.PaymentOrder.GetX(ctx, cycle.ID).Status)
				entitlement := client.UserSubscription.Query().Where(usersubscription.UserIDEQ(user.ID), usersubscription.GroupIDEQ(group.ID)).OnlyX(ctx)
				require.WithinDuration(t, expiry.AddDate(0, 0, 30), entitlement.ExpiresAt, time.Second)
			}
		})
	}
}

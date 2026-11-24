package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5" // #nosec G501 -- WeChat Pay v2 protocol requirement.
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// WeChatRecurringClient implements the direct-merchant WeChat Pay v2 PAP API.
// The v2 signing key is separate from the APIv3 credentials used by Wxpay.
type WeChatRecurringClient struct {
	appID, merchantID, key string
	httpClient             *http.Client
	endpoint               string
}

func NewWeChatRecurringClient(appID, merchantID, key string) (*WeChatRecurringClient, error) {
	if appID == "" || merchantID == "" || len(key) != 32 {
		return nil, errors.New("invalid WeChat PAP credentials")
	}
	return &WeChatRecurringClient{appID: appID, merchantID: merchantID, key: key,
		endpoint: "https://api.mch.weixin.qq.com", httpClient: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}}, nil
}

type WeChatContractCheckout struct {
	OrderNo, ContractCode, TemplateID, Body, ClientIP, OpenID string
	PaymentNotifyURL, ContractNotifyURL                       string
	Serial, AmountFen                                         int64
	ExpiresAt                                                 time.Time
}

func (w *WeChatRecurringClient) Checkout(ctx context.Context, in WeChatContractCheckout) (*payment.CreatePaymentResponse, error) {
	if in.AmountFen <= 0 || in.Serial <= 0 || in.OrderNo == "" || in.ContractCode == "" || in.TemplateID == "" {
		return nil, errors.New("invalid WeChat contract checkout")
	}
	tradeType := "MWEB"
	if in.OpenID != "" {
		tradeType = "JSAPI"
	}
	fields := map[string]string{
		"contract_appid": w.appID, "contract_mchid": w.merchantID,
		"out_trade_no": in.OrderNo, "contract_code": in.ContractCode,
		"plan_id": in.TemplateID, "request_serial": strconv.FormatInt(in.Serial, 10),
		"contract_display_account": "Subscription", "body": in.Body, "trade_type": tradeType,
		"total_fee": strconv.FormatInt(in.AmountFen, 10), "openid": in.OpenID,
		"spbill_create_ip": in.ClientIP, "notify_url": in.PaymentNotifyURL,
		"contract_notify_url": in.ContractNotifyURL,
	}
	if !in.ExpiresAt.IsZero() {
		fields["time_expire"] = in.ExpiresAt.In(time.FixedZone("UTC+8", 8*60*60)).Format("20060102150405")
	}
	result, err := w.call(ctx, "/pay/contractorder", fields)
	if err != nil {
		return nil, err
	}
	if result["out_trade_no"] != "" && result["out_trade_no"] != in.OrderNo {
		return nil, errors.New("WeChat PAP checkout order mismatch")
	}
	if tradeType == "MWEB" {
		redirect, err := url.Parse(result["mweb_url"])
		if err != nil || !strings.EqualFold(redirect.Host, "wx.tenpay.com") || redirect.Scheme != "https" || redirect.User != nil {
			return nil, errors.New("WeChat PAP returned an invalid H5 checkout URL")
		}
		return &payment.CreatePaymentResponse{PayURL: redirect.String()}, nil
	}
	if result["prepay_id"] == "" {
		return nil, errors.New("WeChat PAP missing prepayment ID")
	}
	nonce, err := papayNonce()
	if err != nil {
		return nil, err
	}
	args := map[string]string{"appId": w.appID, "timeStamp": strconv.FormatInt(time.Now().Unix(), 10),
		"nonceStr": nonce, "package": "prepay_id=" + result["prepay_id"], "signType": "MD5"}
	return &payment.CreatePaymentResponse{ResultType: payment.CreatePaymentResultJSAPIReady,
		JSAPI: &payment.WechatJSAPIPayload{AppID: w.appID, TimeStamp: args["timeStamp"], NonceStr: nonce,
			Package: args["package"], SignType: "MD5", PaySign: papaySign(args, w.key, "MD5")}}, nil
}

func (w *WeChatRecurringClient) RequestDebit(ctx context.Context, orderNo, contractID, body, notifyURL string, amountFen int64) error {
	if orderNo == "" || contractID == "" || amountFen <= 0 {
		return errors.New("invalid WeChat PAP debit")
	}
	_, err := w.call(ctx, "/pay/pappayapply", map[string]string{
		"out_trade_no": orderNo, "contract_id": contractID, "body": body,
		"notify_url": notifyURL, "total_fee": strconv.FormatInt(amountFen, 10), "trade_type": "PAP", "fee_type": "CNY",
	})
	return err
}

func (w *WeChatRecurringClient) QueryDebit(ctx context.Context, orderNo string) (map[string]string, error) {
	if orderNo == "" {
		return nil, errors.New("missing WeChat order number")
	}
	result, err := w.call(ctx, "/pay/orderquery", map[string]string{"out_trade_no": orderNo})
	if err != nil {
		return nil, err
	}
	if result["out_trade_no"] != orderNo {
		return nil, errors.New("WeChat PAP query order mismatch")
	}
	return result, nil
}

// CloseUnpaidCheckout closes an abandoned first-payment order. Renewal debit
// orders are reconciled independently and must never use this path.
func (w *WeChatRecurringClient) CloseUnpaidCheckout(ctx context.Context, orderNo string) error {
	_, err := w.call(ctx, "/pay/closeorder", map[string]string{"out_trade_no": orderNo})
	var rejection *WeChatPAPError
	if errors.As(err, &rejection) && rejection.Code == "ORDERCLOSED" {
		return nil
	}
	return err
}

func (w *WeChatRecurringClient) QueryContract(ctx context.Context, templateID, contractCode string) (map[string]string, error) {
	return w.call(ctx, "/papay/querycontract", map[string]string{"plan_id": templateID, "contract_code": contractCode, "version": "1.0"})
}

func (w *WeChatRecurringClient) CancelContract(ctx context.Context, contractID string) error {
	if contractID == "" {
		return errors.New("missing WeChat contract ID")
	}
	result, err := w.call(ctx, "/papay/deletecontract", map[string]string{
		"contract_id": contractID, "version": "1.0", "contract_termination_remark": "Cancel automatic renewal",
	})
	if err == nil && result["contract_id"] != contractID {
		return errors.New("WeChat PAP cancellation contract mismatch")
	}
	return err
}

func (w *WeChatRecurringClient) VerifyNotification(body io.Reader) (map[string]string, error) {
	fields, err := decodePapayXML(body)
	if err != nil {
		return nil, err
	}
	if err := w.verify(fields, fields["change_type"] != ""); err != nil {
		return nil, err
	}
	if fields["return_code"] != "SUCCESS" {
		return nil, errors.New("invalid WeChat PAP notification status")
	}
	return fields, nil
}

func (w *WeChatRecurringClient) call(ctx context.Context, path string, fields map[string]string) (map[string]string, error) {
	fields["appid"], fields["mch_id"] = w.appID, w.merchantID
	if path != "/papay/deletecontract" && path != "/papay/querycontract" {
		nonce, err := papayNonce()
		if err != nil {
			return nil, err
		}
		fields["nonce_str"] = nonce
	}
	fields["sign"] = papaySign(fields, w.key, "MD5")
	var body bytes.Buffer
	encoder := xml.NewEncoder(&body)
	root := xml.StartElement{Name: xml.Name{Local: "xml"}}
	if err := encoder.EncodeToken(root); err != nil {
		return nil, err
	}
	for key, value := range fields {
		if value == "" {
			continue
		}
		if err := encoder.EncodeElement(value, xml.StartElement{Name: xml.Name{Local: key}}); err != nil {
			return nil, err
		}
	}
	if err := encoder.EncodeToken(root.End()); err != nil {
		return nil, err
	}
	if err := encoder.Flush(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint+path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	response, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WeChat PAP HTTP status %d", response.StatusCode)
	}
	result, err := decodePapayXML(response.Body)
	if err != nil {
		return nil, err
	}
	if result["return_code"] != "SUCCESS" {
		return nil, errors.New("WeChat PAP communication failure")
	}
	if err := w.verify(result, false); err != nil {
		return nil, err
	}
	if result["result_code"] != "SUCCESS" {
		return nil, &WeChatPAPError{Code: result["err_code"]}
	}
	return result, nil
}

// WeChatPAPError is returned only for a signed business rejection. Transport
// errors and signatures that fail verification leave the payment outcome unknown.
type WeChatPAPError struct{ Code string }

func (e *WeChatPAPError) Error() string { return "WeChat PAP business rejection" }

func (w *WeChatRecurringClient) verify(fields map[string]string, contractEvent bool) error {
	algorithm := fields["sign_type"]
	if algorithm == "" {
		algorithm = "MD5"
	}
	if algorithm != "MD5" && algorithm != "HMAC-SHA256" {
		return errors.New("invalid WeChat PAP signature algorithm")
	}
	if !hmac.Equal([]byte(fields["sign"]), []byte(papaySign(fields, w.key, algorithm))) {
		return errors.New("invalid WeChat PAP signature")
	}
	if fields["mch_id"] != w.merchantID || (fields["appid"] != w.appID && !(contractEvent && fields["appid"] == "")) {
		return errors.New("WeChat PAP merchant identity mismatch")
	}
	return nil
}

func papayNonce() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(random[:]), nil
}

func papaySign(fields map[string]string, key, algorithm string) string {
	names := make([]string, 0, len(fields))
	for name, value := range fields {
		if name != "sign" && value != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var message strings.Builder
	for _, name := range names {
		message.WriteString(name + "=" + fields[name] + "&")
	}
	message.WriteString("key=" + key)
	if algorithm == "HMAC-SHA256" {
		mac := hmac.New(sha256.New, []byte(key))
		_, _ = io.WriteString(mac, message.String())
		return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	}
	sum := md5.Sum([]byte(message.String())) // #nosec G401 -- Required by WeChat Pay v2.
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func decodePapayXML(body io.Reader) (map[string]string, error) {
	const maxBody = 64 << 10
	raw, err := io.ReadAll(io.LimitReader(body, maxBody+1))
	if err != nil || len(raw) > maxBody {
		return nil, errors.New("invalid WeChat PAP body size")
	}
	decoder := xml.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	fields := make(map[string]string)
	var current string
	var value strings.Builder
	depth := 0
	seenRoot := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			if seenRoot && depth == 0 {
				return fields, nil
			}
			return nil, errors.New("missing WeChat PAP XML root")
		}
		if err != nil {
			return nil, errors.New("malformed WeChat PAP XML")
		}
		switch item := token.(type) {
		case xml.StartElement:
			depth++
			if len(item.Attr) != 0 || item.Name.Space != "" || depth > 2 {
				return nil, errors.New("invalid WeChat PAP XML element")
			}
			if depth == 1 {
				if seenRoot || item.Name.Local != "xml" {
					return nil, errors.New("invalid WeChat PAP XML root")
				}
				seenRoot = true
				continue
			}
			current = item.Name.Local
			if _, exists := fields[current]; exists {
				return nil, errors.New("duplicate WeChat PAP XML field")
			}
			value.Reset()
		case xml.CharData:
			if depth == 2 {
				value.Write(item)
			} else if len(bytes.TrimSpace(item)) != 0 {
				return nil, errors.New("unexpected WeChat PAP XML text")
			}
		case xml.EndElement:
			if depth == 2 {
				fields[current] = value.String()
			}
			depth--
		case xml.ProcInst:
			if seenRoot || item.Target != "xml" {
				return nil, errors.New("unexpected WeChat PAP XML instruction")
			}
		default:
			return nil, errors.New("unexpected WeChat PAP XML token")
		}
	}
}

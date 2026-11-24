package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const papayTestKey = "0123456789abcdef0123456789abcdef"

func TestWeChatRecurringCheckout(t *testing.T) {
	client, err := NewWeChatRecurringClient("app", "merchant", papayTestKey)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields, err := decodePapayXML(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/pay/contractorder" || fields["total_fee"] != "1200" || fields["contract_code"] != "order1" || fields["sign"] != papaySign(fields, papayTestKey, "MD5") {
			t.Errorf("invalid signed checkout fields: %v", fields)
		}
		want := "MWEB"
		if fields["openid"] != "" {
			want = "JSAPI"
		}
		if fields["trade_type"] != want {
			t.Errorf("trade_type = %s, want %s", fields["trade_type"], want)
		}
		if fields["spbill_create_ip"] != "203.0.113.10" {
			t.Errorf("spbill_create_ip = %s, want 203.0.113.10", fields["spbill_create_ip"])
		}
		if want == "MWEB" {
			var scene struct {
				H5Info struct {
					Type    string `json:"type"`
					WapURL  string `json:"wap_url"`
					WapName string `json:"wap_name"`
				} `json:"h5_info"`
			}
			if err := json.Unmarshal([]byte(fields["scene_info"]), &scene); err != nil {
				t.Errorf("invalid MWEB scene_info: %v (%q)", err, fields["scene_info"])
			} else if scene.H5Info.Type != "Wap" || scene.H5Info.WapURL != "https://checkout.example.test" || scene.H5Info.WapName != "Monthly Plan" {
				t.Errorf("unexpected MWEB scene_info: %+v", scene.H5Info)
			}
		} else if fields["scene_info"] != "" {
			t.Errorf("JSAPI should not send scene_info, got %q", fields["scene_info"])
		}
		result := map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "appid": "app", "mch_id": "merchant", "prepay_id": "prepay", "mweb_url": "https://wx.tenpay.com/cgi-bin/mmpayweb-bin/checkmweb"}
		_, _ = fmt.Fprint(w, "<xml>")
		for key, value := range result {
			_, _ = fmt.Fprintf(w, "<%s>%s</%s>", key, value, key)
		}
		_, _ = fmt.Fprintf(w, "<sign>%s</sign></xml>", papaySign(result, papayTestKey, "MD5"))
	}))
	defer server.Close()
	client.endpoint = server.URL
	for _, openID := range []string{"", "payer"} {
		in := WeChatContractCheckout{
			OrderNo: "order1", ContractCode: "order1", TemplateID: "2", Serial: 1, AmountFen: 1200,
			OpenID: openID, ClientIP: "203.0.113.10, 10.0.0.1",
			WapURL: "https://checkout.example.test", WapName: "Monthly Plan",
		}
		result, err := client.Checkout(t.Context(), in)
		if err != nil {
			t.Fatal(err)
		}
		if openID == "" && result.PayURL == "" {
			t.Fatal("missing H5 URL")
		}
		if openID != "" && (result.JSAPI == nil || result.JSAPI.Package != "prepay_id=prepay" || result.JSAPI.PaySign == "") {
			t.Fatal("missing JSAPI parameters")
		}
	}
}

func TestWeChatRecurringQueryAndCancelEndpoints(t *testing.T) {
	client, err := NewWeChatRecurringClient("app", "merchant", papayTestKey)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		fields, err := decodePapayXML(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var result map[string]string
		switch r.URL.Path {
		case "/pay/pappayapply":
			if fields["contract_id"] != "c-1" || fields["total_fee"] != "1200" || fields["trade_type"] != "PAP" || fields["fee_type"] != "CNY" || fields["spbill_create_ip"] != "127.0.0.1" {
				t.Errorf("invalid pappayapply fields: %v", fields)
			}
			result = map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "appid": "app", "mch_id": "merchant"}
		case "/pay/orderquery":
			result = map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "appid": "app", "mch_id": "merchant", "out_trade_no": fields["out_trade_no"], "trade_state": "NOTPAY"}
		case "/pay/paporderquery":
			result = map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "appid": "app", "mch_id": "merchant", "out_trade_no": fields["out_trade_no"], "trade_state": "PAY_FAIL"}
		case "/papay/deletecontract":
			if fields["contract_id"] == "missing" {
				result = map[string]string{"return_code": "SUCCESS", "result_code": "FAIL", "appid": "app", "mch_id": "merchant", "err_code": "CONTRACT_NOT_EXIST"}
			} else {
				result = map[string]string{"return_code": "SUCCESS", "result_code": "FAIL", "appid": "app", "mch_id": "merchant", "err_code": "SYSTEMERROR"}
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprint(w, "<xml>")
		for key, value := range result {
			_, _ = fmt.Fprintf(w, "<%s>%s</%s>", key, value, key)
		}
		_, _ = fmt.Fprintf(w, "<sign>%s</sign></xml>", papaySign(result, papayTestKey, "MD5"))
	}))
	defer server.Close()
	client.endpoint = server.URL

	if err := client.RequestDebit(t.Context(), "cycle-1", "c-1", "Subscription renewal", "https://checkout.example.test/cb", "", 1200); err != nil {
		t.Fatalf("RequestDebit = %v, want nil", err)
	}
	checkoutRes, err := client.QueryCheckout(t.Context(), "init-1")
	if err != nil || checkoutRes["trade_state"] != "NOTPAY" {
		t.Fatalf("QueryCheckout = %v, %v", checkoutRes, err)
	}
	debitRes, err := client.QueryDebit(t.Context(), "cycle-1")
	if err != nil || debitRes["trade_state"] != "PAY_FAIL" {
		t.Fatalf("QueryDebit = %v, %v", debitRes, err)
	}
	if err := client.CancelContract(t.Context(), "missing"); err != nil {
		t.Fatalf("CancelContract(CONTRACT_NOT_EXIST) = %v, want nil", err)
	}
	err = client.CancelContract(t.Context(), "transient")
	var papErr *WeChatPAPError
	if !errors.As(err, &papErr) || papErr.Code != "SYSTEMERROR" {
		t.Fatalf("CancelContract(SYSTEMERROR) = %v, want SYSTEMERROR", err)
	}
	wantPaths := []string{"/pay/pappayapply", "/pay/orderquery", "/pay/paporderquery", "/papay/deletecontract", "/papay/deletecontract"}
	if strings.Join(paths, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
}

func TestWeChatRecurringNotificationVerification(t *testing.T) {
	client, err := NewWeChatRecurringClient("app", "merchant", papayTestKey)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "change_type": "ADD", "mch_id": "merchant"}
	signature := papaySign(fields, papayTestKey, "MD5")
	body := fmt.Sprintf("<xml><return_code>SUCCESS</return_code><result_code>SUCCESS</result_code><change_type>ADD</change_type><mch_id>merchant</mch_id><sign>%s</sign></xml>", signature)
	if _, err := client.VerifyNotification(strings.NewReader(body)); err != nil {
		t.Fatal(err)
	}
	badBodies := []string{
		strings.Replace(body, "merchant", "different", 1),
		strings.Replace(body, "</xml>", "<mch_id>merchant</mch_id></xml>", 1),
		strings.Replace(body, "<change_type>ADD</change_type>", "", 1),
		body + "<xml/>",
		strings.Repeat(" ", 64<<10) + body,
	}
	for i, bad := range badBodies {
		if _, err := client.VerifyNotification(strings.NewReader(bad)); err == nil {
			t.Errorf("invalid notification %d accepted", i)
		}
	}
}

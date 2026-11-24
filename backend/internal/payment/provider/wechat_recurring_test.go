package provider

import (
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
		in := WeChatContractCheckout{OrderNo: "order1", ContractCode: "order1", TemplateID: "2", Serial: 1, AmountFen: 1200, OpenID: openID}
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

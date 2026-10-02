//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaymentReceiptEmail(t *testing.T) {
	cases := map[string]string{
		"payer@example.com":         "payer@example.com",
		"  payer@example.com  ":     "payer@example.com",
		"":                          "",
		"not-an-email":              "",
		"Payer <payer@example.com>": "",
		"linuxdo-1" + LinuxDoConnectSyntheticEmailDomain: "",
		"OIDC-abc@OIDC-CONNECT.INVALID":                  "",
		"wx-1" + WeChatConnectSyntheticEmailDomain:       "",
		"dt-1" + DingTalkConnectSyntheticEmailDomain:     "",
	}
	for input, want := range cases {
		require.Equal(t, want, paymentReceiptEmail(input), "input %q", input)
	}
}

package service

import "testing"

func TestIsFinalWeChatDebitFailure(t *testing.T) {
	finalCodes := []string{
		"ACCOUNTERROR",
		"RULELIMIT",
		"BANKERROR",
		"NOTENOUGH",
		"USER_ACCOUNT_ABNORMAL",
		"USER_NOT_EXIST",
		"TRADE_ERROR",
		"USER_ERROR",
	}
	for _, code := range finalCodes {
		if !isFinalWeChatDebitFailure(code) {
			t.Errorf("isFinalWeChatDebitFailure(%q) = false, want true", code)
		}
	}
	nonRevokingCodes := []string{
		"CONTRACT_NOT_EXIST",
		"SYSTEMERROR",
		"FREQUENCY_LIMITED",
		"ACCEPT",
		"USERPAYING",
		"",
	}
	for _, code := range nonRevokingCodes {
		if isFinalWeChatDebitFailure(code) {
			t.Errorf("isFinalWeChatDebitFailure(%q) = true, want false", code)
		}
	}
}

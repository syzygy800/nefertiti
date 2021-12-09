package huobi

import (
	"net/url"
	"testing"
)

func TestSignature(t *testing.T) {
	params := url.Values{}
	params.Add("currency", "btcusdt")
	params.Add("account-id", "1")

	signature := sign("secret", "GET", "api.huobi.pro", "/v1/account/history", params)
	expected := "HUP3n78npIuTzVKyjEOrPictRKEUTRoYs7Ld5y38hmA="

	if signature != expected {
		t.Errorf("TestSignature failed, got: %v, want: %v.", signature, expected)
	}
}

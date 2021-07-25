package woo

import (
	"net/url"
	"testing"
)

func TestSignature(t *testing.T) {
	params := url.Values{}
	params.Add("param1", "val1")
	params.Add("param2", "val2")
	params.Add("symbol", "SPOT_BTC_USDT")
	params.Add("order_type", "LIMIT")
	params.Add("order_price", "9000")
	params.Add("order_quantity", "0.11")
	params.Add("side", "BUY")

	calculated := signature("QHKRXHPAW1MC9YGZMAT8YDJG2HPR", params, 1578565539808)
	expected := "9766aac353561a08111db564f9228ab60b92fef0de565f5a092f0bfbf6547e26"

	if calculated != expected {
		t.Errorf("TestSignature failed, got: %v, want: %v.", calculated, expected)
	}
}

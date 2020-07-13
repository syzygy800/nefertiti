package model

import (
	"strings"
)

const (
	EUR  = "EUR"
	USD  = "USD"
	BTC  = "BTC"
	BCH  = "BCH"
	ETH  = "ETH"
	LTC  = "LTC"
	XRP  = "XRP"
	DASH = "DASH"
	ZEC  = "ZEC"
)

func Fiat(curr string) bool {
	return strings.EqualFold(curr, EUR) || strings.EqualFold(curr, USD)
}

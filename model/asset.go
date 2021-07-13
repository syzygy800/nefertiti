package model

import (
	"strings"
)

const (
	EUR = "EUR"
	USD = "USD"
	BTC = "BTC"
	ETH = "ETH"
	LTC = "LTC"
	XRP = "XRP"
)

func Fiat(asset string) bool {
	return strings.EqualFold(asset, EUR) || strings.EqualFold(asset, USD)
}

type (
	Assets []string
)

func (assets Assets) IndexOf(asset string) int {
	for i, a := range assets {
		if strings.EqualFold(a, asset) {
			return i
		}
	}
	return -1
}

func (assets Assets) HasAsset(asset string) bool {
	return assets.IndexOf(asset) > -1
}

func (assets Assets) IsEmpty() bool {
	return len(assets) == 0 || len(assets) == 1 && assets[0] == ""
}

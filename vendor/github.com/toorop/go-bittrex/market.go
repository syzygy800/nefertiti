package bittrex

import (
	"fmt"
)

type Market1 struct {
	MarketCurrency     string  `json:"MarketCurrency"`
	BaseCurrency       string  `json:"BaseCurrency"`
	MarketCurrencyLong string  `json:"MarketCurrencyLong"`
	BaseCurrencyLong   string  `json:"BaseCurrencyLong"`
	MinTradeSize       float64 `json:"MinTradeSize"`
	MarketName         string  `json:"MarketName"`
	IsActive           bool    `json:"IsActive"`
}

type Market3 struct {
	Symbol              string  `json:"symbol"`
	BaseCurrencySymbol  string  `json:"baseCurrencySymbol"`
	QuoteCurrencySymbol string  `json:"quoteCurrencySymbol"`
	MinTradeSize        float64 `json:"minTradeSize,string"`
	Precision           int     `json:"precision"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"createdAt"`
}

// MarketName returns the old (v1) market name that was reversed.
func (market *Market3) MarketName() string {
	return fmt.Sprintf("%s-%s", market.QuoteCurrencySymbol, market.BaseCurrencySymbol)
}

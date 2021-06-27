package bittrex

import (
	"fmt"
)

type Market struct {
	Symbol              string  `json:"symbol"`
	BaseCurrencySymbol  string  `json:"baseCurrencySymbol"`
	QuoteCurrencySymbol string  `json:"quoteCurrencySymbol"`
	MinTradeSize        float64 `json:"minTradeSize,string"`
	Precision           int     `json:"precision"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"createdAt"`
}

// MarketName returns the old (v1) market name that was reversed.
func (market *Market) MarketName() string {
	return fmt.Sprintf("%s-%s", market.QuoteCurrencySymbol, market.BaseCurrencySymbol)
}

// true if this market is currently active, otherwise false.
func (market *Market) Online() bool {
	return market.Status != "OFFLINE"
}

package bittrex

import (
	"fmt"
	"strings"
)

type Market struct {
	Symbol              string   `json:"symbol"`
	BaseCurrencySymbol  string   `json:"baseCurrencySymbol"`
	QuoteCurrencySymbol string   `json:"quoteCurrencySymbol"`
	MinTradeSize        float64  `json:"minTradeSize,string"`
	Precision           int      `json:"precision"`
	Status              string   `json:"status"`
	CreatedAt           string   `json:"createdAt"`
	Notice              string   `json:"notice,omitempty"`
	ProhibitedIn        []string `json:"prohibitedIn,omitempty"`
}

// MarketName returns the old (v1) market name that was reversed.
func (market *Market) MarketName() string {
	return fmt.Sprintf("%s-%s", market.QuoteCurrencySymbol, market.BaseCurrencySymbol)
}

func (market *Market) Online() bool {
	return market.Status != "OFFLINE"
}

// true if this market is currently online (and not about to be removed), otherwise false.
func (market *Market) Active() bool {
	return market.Online() && !strings.Contains(market.Notice, "will be removed")
}

func (market *Market) IsProhibited(regions []string) bool {
	for _, r := range regions {
		for _, s := range market.ProhibitedIn {
			if strings.EqualFold(s, r) {
				return true
			}
		}
	}
	return false
}

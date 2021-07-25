package woo

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Symbol struct {
	Symbol      string  `json:"symbol"`
	QuoteMin    float64 `json:"quote_min"`
	QuoteMax    float64 `json:"quote_max"`
	QuoteTick   float64 `json:"quote_tick"`
	BaseMin     float64 `json:"base_min"`
	BaseMax     float64 `json:"base_max"`
	BaseTick    float64 `json:"base_tick"`
	MinNotional float64 `json:"min_notional"`
	PriceRange  float64 `json:"price_range"`
	CreatedTime string  `json:"created_time"`
	UpdatedTime string  `json:"updated_time"`
}

type Symbols struct {
	Rows []Symbol `json:"rows"`
}

func (client *Client) Symbols() ([]Symbol, error) {
	var (
		err  error
		out  Symbols
		body []byte
	)

	if body, err = client.get("/v1/public/info", nil, false, 30); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &out); err != nil {
		return nil, err
	}

	return out.Rows, nil
}

func FormatSymbol(base, quote string) string {
	return "SPOT_" + strings.ToUpper(base) + "_" + strings.ToUpper(quote)
}

func ParseSymbol(symbol string) (string, string, error) { // -> (base, quote, error)
	subs := strings.Split(symbol, "_")
	if len(subs) > 2 {
		return subs[1], subs[2], nil
	}
	return "", "", fmt.Errorf("cannot parse symbol %s", symbol)
}

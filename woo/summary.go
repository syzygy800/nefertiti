package woo

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Ticker struct {
	Symbol                string  `json:"trading_pairs"`
	LastPrice             float64 `json:"last_price"`
	LowestAsk             float64 `json:"lowest_ask"`
	HighestBid            float64 `json:"highest_bid"`
	BaseVolume            float64 `json:"base_volume"`
	QuoteVolume           float64 `json:"quote_volume"`
	PriceChangePercent24h float64 `json:"price_change_percent_24h"`
	HighestPrice24h       float64 `json:"highest_price_24h"`
	LowestPrice24h        float64 `json:"lowest_price_24h"`
}

type Summary []Ticker

func (client *Client) Summary() (Summary, error) {
	var (
		err  error
		body []byte
		out  Summary
	)
	if body, err = client.get("/md/cmc/summary", nil, false, 30); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (client *Client) Ticker(symbol string) (*Ticker, error) {
	base, quote, err := ParseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	sum, err := client.Summary()
	if err != nil {
		return nil, err
	}

	for _, ticker := range sum {
		if ticker.Symbol == fmt.Sprintf("%s_%s", strings.ToUpper(base), strings.ToUpper(quote)) {
			return &ticker, nil
		}
	}

	return nil, fmt.Errorf("symbol %s does not exist", symbol)
}

package huobi

import (
	"encoding/json"
	"net/url"
)

type Summary struct {
	Low    float64 `json:"low"`    // the lowest price of last 24 hours
	High   float64 `json:"high"`   // the highest price of last 24 hours
	Open   float64 `json:"open"`   // the opening price of last 24 hours
	Close  float64 `json:"close"`  // the closing price of last 24 hours
	Volume float64 `json:"vol"`    // the trading volume in base currency of last 24 hours
	Amount float64 `json:"amount"` // the aggregated trading volume in USDT of last 24 hours
}

func (client *Client) Summary(symbol string) (*Summary, error) {
	type Response struct {
		Tick Summary `json:"tick"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	params := url.Values{}
	params.Add("symbol", symbol)

	if body, err = client.get("/market/detail", params, false); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp.Tick, nil
}

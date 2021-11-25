package huobi

import (
	"encoding/json"
	"net/url"
)

type Ticker struct {
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	Direction string  `json:"direction"`
}

func (client *Client) Ticker(symbol string) (*Ticker, error) {
	type Response struct {
		Tick struct {
			Data []Ticker `json:"data"`
		} `json:"tick"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	params := url.Values{}
	params.Add("symbol", symbol)

	if body, err = client.get("/market/trade", params); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp.Tick.Data[0], nil
}

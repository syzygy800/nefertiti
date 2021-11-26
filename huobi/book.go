package huobi

import (
	"encoding/json"
	"net/url"
)

type (
	BookEntry []float64
)

func (be *BookEntry) Price() float64 {
	return (*be)[0]
}

func (be *BookEntry) Size() float64 {
	return (*be)[1]
}

type OrderBook struct {
	Bids []BookEntry `json:"bids"`
	Asks []BookEntry `json:"asks"`
}

func (client *Client) OrderBook(symbol string) (*OrderBook, error) {
	type Response struct {
		Tick OrderBook `json:"tick"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	params := url.Values{}
	params.Add("type", "step0")
	params.Add("symbol", symbol)

	if body, err = client.get("/market/depth", params); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp.Tick, nil
}

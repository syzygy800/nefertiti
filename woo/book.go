package woo

import (
	"encoding/json"
)

type BookEntry struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

type OrderBook struct {
	Bids []BookEntry `json:"bids"`
	Asks []BookEntry `json:"asks"`
}

func (client *Client) OrderBook(symbol string) (*OrderBook, error) {
	var (
		err  error
		body []byte
		out  OrderBook
	)
	if body, err = client.get(("/v1/orderbook/" + symbol), nil, true, 10); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

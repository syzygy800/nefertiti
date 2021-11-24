package huobi

import (
	"encoding/json"
)

type Symbol struct {
	BaseCurrency           string  `json:"base-currency"`
	QuoteCurrency          string  `json:"quote-currency"`
	PricePrecision         int     `json:"price-precision"`  // quote currency precision
	AmountPrecision        int     `json:"amount-precision"` // base currency precision
	Symbol                 string  `json:"symbol"`
	State                  string  `json:"state"`                      // "online" or "offline" or "suspended"
	ValuePrecision         int     `json:"value-precision"`            // precision of value in quote currency (value = price * amount)
	LimitOrderMinOrderAmt  float64 `json:"limit-order-min-order-amt"`  // minimum order amount of limit order in base currency
	LimitOrderMaxOrderAmt  float64 `json:"limit-order-max-order-amt"`  // max order amount of limit order in base currency
	SellMarketMinOrderAmt  float64 `json:"sell-market-min-order-amt"`  // minimum order amount of sell-market order in base currency
	SellMarketMaxOrderAmt  float64 `json:"sell-market-max-order-amt"`  // max order amount of sell-market order in base currency
	BuyMarketMaxOrderValue float64 `json:"buy-market-max-order-value"` // max order value of buy-market order in quote currency
	MinOrderValue          float64 `json:"min-order-value"`            // minimum order value of limit order and buy-market order in quote currency
	MaxOrderValue          float64 `json:"max-order-value"`            // max order value of limit order and buy-market order in usdt
	ApiTrading             string  `json:"api-trading"`                // "enabled" or "disabled"
}

func (symbol *Symbol) Online() bool {
	return symbol.State == "online"
}

func (symbol *Symbol) Enabled() bool {
	return symbol.ApiTrading == "enabled"
}

func (client *Client) Symbols() ([]Symbol, error) {
	type Response struct {
		Data []Symbol `json:"data"`
	}

	var (
		err  error
		body []byte
		resp Response
	)

	if body, err = client.get("/v1/common/symbols", nil); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

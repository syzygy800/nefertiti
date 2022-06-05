package gdax

import (
	"strconv"

	exchange "github.com/svanas/go-coinbasepro"
	"github.com/svanas/nefertiti/precision"
)

func ParseFloat(value string) float64 {
	out, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return out
	}
	return 0
}

func GetMinOrderSize(client *Client, product *exchange.Product) (float64, error) {
	// step #1: get minimum notional value (aka price * quantity) allowed for an order
	min, err := strconv.ParseFloat(product.MinMarketFunds, 64)
	if err != nil {
		return 0, err
	}
	// step #2: get the ticker price
	ticker, err := client.GetTicker(product.ID)
	if err != nil {
		return 0, err
	}
	// step #3: divide notional by ticker price, round to size precision.
	return precision.Round((min / ParseFloat(ticker.Price)), precision.Parse(product.BaseIncrement, 0)), nil
}

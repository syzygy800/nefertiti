package gdax

import (
	"strconv"

	exchange "github.com/svanas/go-coinbasepro"
)

func ParseFloat(value string) float64 {
	out, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return out
	}
	return 0
}

func GetMinOrderSize(product *exchange.Product) (float64, error) {
	return strconv.ParseFloat(product.BaseMinSize, 64)
}

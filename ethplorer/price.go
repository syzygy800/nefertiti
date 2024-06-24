package ethplorer

import (
	"strconv"

	"github.com/svanas/nefertiti/any"
)

type (
	Price struct {
		Rate interface{} `json:"rate"`
		Diff float64     `json:"diff"`
	}
)

func (price *Price) ParseRate() float64 {
	if out, err := strconv.ParseFloat(any.AsString(price.Rate), 64); err != nil {
		return out
	}
	return any.AsFloat64(price.Rate)
}

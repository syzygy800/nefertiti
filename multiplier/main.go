package multiplier

import (
	"fmt"
	"strconv"

	"bitbucket.com/svanas/cryptotrader/flag"
	"bitbucket.com/svanas/cryptotrader/precision"
)

const (
	FIVE_PERCENT = 1.05
)

func Scale(mult, x float64) float64 {
	return precision.Round((1 + ((mult - 1) * x)), 5)
}

func Get(def float64) (float64, error) {
	var (
		err error
		out float64 = def
	)
	arg := flag.Get("mult")
	if !arg.Exists {
		flag.Set("mult", strconv.FormatFloat(out, 'f', -1, 64))
	} else {
		if out, err = arg.Float64(); err != nil {
			return out, fmt.Errorf("mult %v is invalid", arg)
		}
	}
	return out, nil
}

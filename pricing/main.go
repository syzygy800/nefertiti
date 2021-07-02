package pricing

import (
	"fmt"
	"strconv"
)

func Multiply(price, mult float64, prec int) float64 {
	var (
		multiplied float64
		rounded    float64
	)
	multiplied = price * mult
	if multiplied != 0 {
		for {
			rounded, _ = strconv.ParseFloat(fmt.Sprintf("%.[2]*[1]f", multiplied, prec), 64)
			if (mult > 1 && rounded > price) || (mult < 1 && rounded < price) || (mult == 1) {
				break
			} else {
				multiplied = multiplied * 1.01
			}
		}
	}
	return rounded
}

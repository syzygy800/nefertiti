package precision

import (
	"fmt"
	"math"
	"strconv"
)

func Format(prec int) string {
	var out string
	if prec > 0 {
		out = "0."
		for i := 0; i < (prec - 1); i++ {
			out += "0"
		}
	}
	out = out + "1"
	return out
}

func Round(value float64, prec int) float64 {
	out, _ := strconv.ParseFloat(fmt.Sprintf("%.[2]*[1]f", value, prec), 64)
	return out
}

func Floor(value float64, prec int) float64 {
	str := "1"
	for i := 0; i < prec; i++ {
		str += "0"
	}
	fac, err := strconv.ParseFloat(str, 64)
	if err == nil {
		return math.Floor((value * fac)) / fac
	}
	return value
}

func Ceil(value float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	return math.Ceil((value * pow)) / pow
}

package pricing

import (
	"fmt"
	"math"
	"strconv"
)

const (
	FIVE_PERCENT = 1.05
)

func FormatPrecision(prec int) string {
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

func NewMult(old, mult float64) float64 {
	return RoundToPrecision((1 + ((old - 1) * mult)), 5)
}

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

// RoundToNearest rounds [x] to to nearest multiple of [unit]
func RoundToNearest(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}

// RoundToPrecision sets the number of places after the decimal
func RoundToPrecision(x float64, prec int) float64 {
	out, _ := strconv.ParseFloat(fmt.Sprintf("%.[2]*[1]f", x, prec), 64)
	return out
}

// FloorToPrecision sets the number of places after the decimal
func FloorToPrecision(x float64, prec int) float64 {
	str := "1"
	for i := 0; i < prec; i++ {
		str += "0"
	}
	fac, err := strconv.ParseFloat(str, 64)
	if err == nil {
		return math.Floor((x * fac)) / fac
	}
	return x
}

func CeilToPrecision(x float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	return math.Ceil((x * pow)) / pow
}

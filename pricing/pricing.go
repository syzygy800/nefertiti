package pricing

import (
	"fmt"
	"math"
	"strconv"
)

const (
	FIVE_PERCENT = 1.05
)

func NewMult(old, mult float64) float64 {
	return RoundToPrecision((1 + ((old - 1) * mult)), 5)
}

func Multiply(price, mult float64, prec int) float64 {
	var out float64
	out = price * mult
	// sets the number of places after the decimal
	out, _ = strconv.ParseFloat(fmt.Sprintf("%.[2]*[1]f", out, prec), 64)
	return out
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

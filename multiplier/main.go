package multiplier

import (
	"fmt"
	"strconv"

	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/precision"
)

const (
	FIVE_PERCENT = 1.05
)

type Mult float64

func Get(def float64) (Mult, error) {
	var (
		err error
		out float64 = def
	)
	arg := flag.Get("mult")
	if !arg.Exists {
		flag.Set("mult", strconv.FormatFloat(out, 'f', -1, 64))
	} else {
		if out, err = arg.Float64(); err != nil {
			return Mult(out), fmt.Errorf("mult %v is invalid", arg)
		}
		if out <= 1 || out >= 2 {
			return Mult(out), fmt.Errorf("mult %v is not in the 1..2 range", arg)
		}
	}
	return Mult(out), nil
}

func Stop() (Mult, error) {
	// the default value is twice the mult value
	def := func() (float64, error) {
		mult, err := Get(FIVE_PERCENT)
		if err != nil {
			return 0, err
		}
		return 1 - ((float64(mult) - 1) * 2), nil
	}
	var (
		err error
		out float64
	)
	if out, err = def(); err != nil {
		return 0, err
	}
	// get the --stop=[0..1] value
	arg := flag.Get("stop")
	if !arg.Exists {
		flag.Set("stop", strconv.FormatFloat(out, 'f', -1, 64))
	} else {
		if out, err = arg.Float64(); err != nil {
			return Mult(out), fmt.Errorf("stop %v is invalid", arg)
		}
		if out <= 0 || out >= 1 {
			return Mult(out), fmt.Errorf("stop %v is not in the 0..1 range", arg)
		}
	}
	return Mult(out), nil
}

func Format(mult Mult) string {
	if mult >= 1 {
		return fmt.Sprintf("%.2f%%", ((mult - 1) * 100))
	} else {
		return fmt.Sprintf("-%.2f%%", ((1 - mult) * 100))
	}
}

// after a stop-loss got filled, pass in...
// 1) the order size, and
// 2) the size precision, and
// 3) what price you got stopped at.
// assuming -mult and -stop do not change, this func will return your new stop-loss order *size*
func DoubleOrNothing(size float64, prec int, stoppedAt float64) (float64, error) {
	out := precision.Round((size * 2.2), prec)

	if stoppedAt == 0 {
		return out, nil
	}

	mult, err := Get(FIVE_PERCENT)
	if err != nil {
		return out, err
	}

	stop, err := Stop()
	if err != nil {
		return out, err
	}

	inc, err := strconv.ParseFloat(precision.Format(prec), 64)
	if err != nil {
		return out, err
	}

	loss := size * ((stoppedAt / float64(stop)) - stoppedAt)
	for (out * ((stoppedAt * float64(mult)) - stoppedAt)) < loss {
		out = out + inc
	}

	return precision.Round(out, prec), nil
}

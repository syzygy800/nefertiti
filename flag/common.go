package flag

import (
	"strconv"

	"github.com/svanas/nefertiti/errors"
)

// --dca (aka dollar-cost-averaging)
func Dca() bool {
	return Exists("dca")
}

// --debug
func Debug() bool {
	return Exists("debug")
}

// --dip=[0..99]
func Dip(def float64) (float64, error) {
	var err error
	dip := def
	arg := Get("dip")
	if !arg.Exists {
		Set("dip", strconv.FormatFloat(dip, 'f', -1, 64))
	} else {
		if dip, err = arg.Float64(); err != nil {
			return dip, errors.Errorf("dip %v is invalid", arg)
		}
		if dip < 0 || dip >= 100 {
			return dip, errors.Errorf("dip %v is invalid", arg)
		}
	}
	return dip, nil
}

// --pip=[dip..100]
func Pip(def float64) (float64, error) {
	var err error
	dip := 5.0
	pip := def
	arg := Get("pip")
	if !arg.Exists {
		Set("pip", strconv.FormatFloat(pip, 'f', -1, 64))
	} else {
		if pip, err = arg.Float64(); err != nil {
			return pip, errors.Errorf("pip %v is invalid", arg)
		}
		if dip, err = Dip(dip); err != nil {
			return pip, err
		}
		if pip <= dip || pip > 100 {
			return pip, errors.Errorf("pip %v is invalid", arg)
		}
	}
	return pip, nil
}

// --max=X
func Max() (float64, error) {
	var (
		err error
		max float64 = 0
	)
	arg := Get("max")
	if arg.Exists {
		if max, err = arg.Float64(); err != nil {
			return max, errors.Errorf("max %v is invalid", arg)
		}
	}
	return max, nil
}

// --min=X
func Min() (float64, error) {
	var (
		err error
		min float64 = 0
	)
	arg := Get("min")
	if arg.Exists {
		if min, err = arg.Float64(); err != nil {
			return min, errors.Errorf("min %v is invalid", arg)
		}
	}
	return min, nil
}

// --dist=X
func Dist(def int64) (int64, error) {
	var err error
	out := def
	arg := Get("dist")
	if arg.Exists {
		if out, err = arg.Int64(); err != nil {
			return out, errors.Errorf("dist %v is invalid", arg)
		}
	}
	return out, nil
}

// --sandbox=[Y|N]
func Sandbox() bool {
	arg := Get("sandbox")
	if arg.Exists {
		str := arg.String()
		return len(str) > 0 && (str[0] == 'Y' || str[0] == 'y')
	}
	return false
}

// when included, then the bot will respect the --dip setting (or the default 5% value) and not be smart about it.
func Strict() bool {
	return Exists("strict")
}

// when included, then the bot runs as a local web server and is listening to a port
func Listen() bool {
	return Exists("listen")
}

func Interactive() bool {
	return !Listen()
}

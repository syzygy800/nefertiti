package flag

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type (
	Flag struct {
		Exists bool
		value  string
	}
)

func New(exists bool, val string) *Flag {
	return &Flag{
		Exists: exists,
		value:  val,
	}
}

func (flg *Flag) String() string {
	return flg.value
}

func (flg *Flag) Int64() (out int64, err error) {
	out, err = strconv.ParseInt(flg.value, 0, 0)
	return
}

func (flg *Flag) Float64() (out float64, err error) {
	out, err = strconv.ParseFloat(flg.value, 64)
	return
}

// Get() finds a named flag in the args list and returns its value
func Get(name string) *Flag {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-"+name) || strings.HasPrefix(arg, "--"+name) {
			i := strings.Index(arg, "=")
			if i > -1 {
				if strings.Contains(arg, "-"+name+"=") {
					return New(true, arg[i+1:])
				}
			} else {
				return New(true, "")
			}
		}
	}
	return New(false, "")
}

func GetAsInt(name string, def int64) (int64, error) {
	out := def
	flg := Get(name)
	if flg.Exists {
		var err error
		if out, err = flg.Int64(); err != nil {
			return 0, fmt.Errorf("%s=%v is invalid", name, flg)
		}
	}
	return out, nil
}

func Set(name, value string) {
	args := os.Args
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			for true {
				arg = arg[1:]
				if !strings.HasPrefix(arg, "-") {
					break
				}
			}
			n := strings.Index(arg, "=")
			if n > -1 {
				arg = arg[:n]
			}
			if arg == name {
				args = append(args[:i], args[i+1:]...)
			} else {
				i++
			}
		} else {
			i++
		}
	}
	if value == "" {
		args = append(args, ("--" + name))
	} else {
		args = append(args, ("--" + name + "=" + value))
	}
	os.Args = args
}

// Exists() determines if a flag exists, even if it doesn't have a value
func Exists(name string) bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-"+name) || strings.HasPrefix(arg, "--"+name) {
			return true
		}
	}
	return false
}

func Debug() bool {
	return Exists("debug")
}

// when included, then the bot will respect the --dip setting (or the default 5% value) and not be smart about it.
func Strict() bool {
	return Exists("strict")
}

func Listen() bool {
	return Exists("listen")
}

func Interactive() bool {
	return Listen() == false
}

//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package flag

import (
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

func (self *Flag) String() string {
	return self.value
}

func (self *Flag) Split() []string {
	return self.SplitEx(",")
}

func (self *Flag) SplitEx(sep string) []string {
	return strings.Split(self.value, sep)
}

func (self *Flag) Contains(value string) bool {
	return self.ContainsEx(",", value)
}

func (self *Flag) ContainsEx(sep, value string) bool {
	for _, sub := range self.SplitEx(sep) {
		if strings.EqualFold(sub, value) {
			return true
		}
	}
	return false
}

func (self *Flag) Int64() (int64, error) {
	return strconv.ParseInt(self.value, 0, 0)
}

func (self *Flag) Float64() (float64, error) {
	return strconv.ParseFloat(self.value, 64)
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

func Set(name, value string) {
	args := os.Args
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			for {
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

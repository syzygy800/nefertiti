package coinbase

import (
	"fmt"
	"strings"
	"time"
)

type ServerTime struct {
	ISO   string  `json:"iso"`
	Epoch float64 `json:"epoch,number"`
}

func (c *Client) GetTime() (ServerTime, error) {
	var serverTime ServerTime

	url := fmt.Sprintf("/time")
	_, err := c.Request("GET", url, nil, &serverTime)
	return serverTime, err
}

type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	var (
		err error
		out time.Time
		str = strings.Replace(string(data), "\"", "", -1)
	)

	if str == "null" {
		*t = Time(time.Time{})
		return nil
	}

	parse := func(input string) (time.Time, error) {
		var (
			err error
			out time.Time
		)
		layouts := []string{
			"2006-01-02 15:04:05+00",
			"2006-01-02T15:04:05.999999Z",
			"2006-01-02 15:04:05.999999",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05.999999+00"}
		for _, layout := range layouts {
			out, err = time.Parse(layout, input)
			if err == nil {
				return out, nil
			}
		}
		return out, err
	}

	out, err = parse(str)

	if err != nil && strings.Contains(str, "NaN") {
		out, err = parse(strings.Replace(str, "NaN", "000", -1))
		if err != nil {
			out, err = parse(strings.Replace(str, "NaN", "00", -1))
			if err != nil {
				out, err = parse(strings.Replace(str, "NaN", "0", -1))
				if err != nil {
					out, err = parse(strings.Replace(str, "NaN", "", -1))
				}
			}
		}
	}

	if err != nil {
		return err
	}
	if out.IsZero() {
		return fmt.Errorf("cannot parse %s into time", str)
	}

	*t = Time(out)

	return nil
}

// MarshalJSON marshal time back to time.Time for json encoding
func (t Time) MarshalJSON() ([]byte, error) {
	return t.Time().MarshalJSON()
}

func (t *Time) Time() time.Time {
	return time.Time(*t)
}

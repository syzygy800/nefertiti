package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/precision"
)

type (
	Buy struct {
		Market string  `json:"market"`
		Price  float64 `json:"price"`
		Size   float64 `json:"size,omitempty"`
	}
)

type (
	Call struct {
		*Buy
		Skip   bool   `json:"-"`
		Reason string `json:"-"`
		Stop   string `json:"stop,omitempty"`
		Target string `json:"target,omitempty"`
	}
	Calls []Call
)

func Call2File(call *Call, name string) error {
	var (
		err error
		raw []byte
	)
	if raw, err = json.Marshal(call); err != nil {
		return errors.Wrap(err, 1)
	}
	if _, err = os.Stat(name); err == nil {
		if err = os.Truncate(name, 0); err != nil {
			return errors.Wrap(err, 1)
		}
	}
	if err = ioutil.WriteFile(name, raw, 0600); err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}

func File2Call(name string) (*Call, error) {
	var (
		err error
		raw []byte
		out Call
	)
	if raw, err = ioutil.ReadFile(name); err != nil {
		return nil, errors.Wrap(err, 1)
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return &out, nil
}

func (c *Call) Ignore(reason string, a ...interface{}) {
	c.Skip = true
	if c.Reason == "" {
		if a == nil {
			c.Reason = reason
		} else {
			c.Reason = fmt.Sprintf(reason, a...)
		}
	}
}

func (c *Call) HasStop() bool {
	return c.Stop != "" && c.ParseStop() > 0
}

func (c *Call) ParseStop() float64 {
	out, err := strconv.ParseFloat(c.Stop, 64)
	if err == nil {
		return out
	}
	return 0
}

func (c *Call) HasTarget() bool {
	return c.Target != "" && c.ParseTarget() > 0
}

func (c *Call) ParseTarget() float64 {
	out, err := strconv.ParseFloat(c.Target, 64)
	if err == nil {
		return out
	}
	return 0
}

func (c *Call) Corrupt(orderType OrderType) (bool, string) { // (corrupt, reason)
	if c.Size == 0 {
		return true, "nothing to buy"
	}

	if c.Price == 0 && orderType == LIMIT {
		return true, "limit order without a limit"
	}

	if c.HasTarget() {
		target := c.ParseTarget()
		if target < c.Price {
			return true, fmt.Sprintf("sell target %.8f is lower than buy zone %.8f", target, c.Price)
		}
	}

	// just because we will want to skip a signal, does not make it corrupt. there should be a good reason for it.
	return c.Skip && c.Reason != "", c.Reason
}

// Multiply the buy target. Returns (new order type, deviated buy target). Does not modify the buy signal itself.
func (c *Call) Deviate(exchange Exchange, client interface{}, kind OrderType, mult float64) (OrderType, float64) {
	if mult != 1.0 {
		limit := c.Price
		if limit == 0 {
			ticker, err := exchange.GetTicker(client, c.Market)
			if err == nil {
				limit = ticker
			}
		}
		if limit > 0 {
			prec, err := exchange.GetPricePrec(client, c.Market)
			if err == nil {
				limit = precision.Round((limit * mult), prec)
				return LIMIT, limit
			}
		}
	}
	return kind, c.Price
}

func (c Calls) HasAnythingToDo() bool {
	for _, e := range c {
		if !e.Skip {
			return true
		}
	}
	return false
}

func (c Calls) IndexByMarket(market string) int {
	for i, e := range c {
		if e.Market == market {
			return i
		}
	}
	return -1
}

func (c Calls) IndexByPrice(price float64) int {
	for i, e := range c {
		if e.Price == price {
			return i
		}
	}
	return -1
}

func (c Calls) IndexByMarketPrice(market string, price float64) int {
	for i, e := range c {
		if e.Market == market && e.Price == price {
			return i
		}
	}
	return -1
}

type (
	Book []Buy
)

type BookSide int

const (
	BOOK_SIDE_BIDS BookSide = iota
	BOOK_SIDE_ASKS
)

func (b Book) Calls() Calls {
	var out Calls
	for _, e := range b {
		out = append(out, Call{
			Buy: &Buy{
				Market: e.Market,
				Price:  e.Price,
				Size:   e.Size,
			},
		})
	}
	return out
}

func (b Book) IndexByPrice(price float64) int {
	for i, e := range b {
		if e.Price == price {
			return i
		}
	}
	return -1
}

func (b Book) EntryByPrice(price float64) *Buy {
	i := b.IndexByPrice(price)
	if i != -1 {
		return &b[i]
	}
	return nil
}

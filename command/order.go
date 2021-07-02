package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/multiplier"
	"github.com/svanas/nefertiti/pricing"
)

type (
	OrderCommand struct {
		*CommandMeta
	}
)

func (c *OrderCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	sandbox := false
	flg = flag.Get("sandbox")
	if flg.Exists {
		sandbox = flg.String() == "Y"
	}

	flg = flag.Get("exchange")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: exchange"))
	}
	exchange := exchanges.New().FindByName(flg.String())
	if exchange == nil {
		return c.ReturnError(fmt.Errorf("exchange %v does not exist", flg))
	}

	flg = flag.Get("side")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: side"))
	}
	side := model.NewOrderSide(flg.String())
	if side == model.ORDER_SIDE_NONE {
		return c.ReturnError(fmt.Errorf("side %v is invalid", flg))
	}

	var kind model.OrderType = model.LIMIT
	flg = flag.Get("type")
	if flg.Exists {
		kind = model.NewOrderType(flg.String())
		if kind == model.ORDER_TYPE_NONE {
			return c.ReturnError(fmt.Errorf("type %v is invalid", flg))
		}
	}

	var markets []model.Market
	if markets, err = exchange.GetMarkets(true, sandbox); err != nil {
		return c.ReturnError(err)
	}

	flg = flag.Get("market")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: market"))
	}
	market := flg.String()
	if model.HasMarket(markets, market) == false {
		return c.ReturnError(fmt.Errorf("market %s does not exist", market))
	}

	var size float64
	flg = flag.Get("size")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: size"))
	}
	if size, err = flg.Float64(); err != nil {
		return c.ReturnError(fmt.Errorf("size %v is invalid", flg))
	}

	var price float64 = 0
	flg = flag.Get("price")
	if flg.Exists {
		if price, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("price %v is invalid", flg))
		}
	} else if kind == model.LIMIT {
		return c.ReturnError(errors.New("missing argument: price"))
	}

	var client interface{}
	if client, err = exchange.GetClient(model.PRIVATE, sandbox); err != nil {
		return c.ReturnError(err)
	}

	var mult float64 = 1.0
	if mult, err = multiplier.Get(mult); err != nil {
		return c.ReturnError(err)
	} else if mult != 1.0 {
		var prec int
		if prec, err = exchange.GetPricePrec(client, market); err != nil {
			return c.ReturnError(err)
		} else {
			price = pricing.Multiply(price, mult, prec)
		}
	}

	var out []byte
	if _, out, err = exchange.Order(
		client,
		side,
		market,
		size,
		price,
		kind, "",
	); err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(string(out))

	return 0
}

func (c *OrderCommand) Help() string {
	text := `
Usage: ./nefertiti order [options]

The order command places an order with the specified exchange.

Options:
  --exchange = name
  --side     = [buy|sell]
  --type     = [limit|market]
  --market   = a valid market pair
  --size     = amount of cryptocurrency to buy or sell
  --price    = price per unit (optional, not needed for market orders)
  --mult     = vector to multiply price with (optional, defaults to 1.0)
`
	return strings.TrimSpace(text)
}

func (c *OrderCommand) Synopsis() string {
	return "Place an order with the specified exchange."
}

package command

import (
	"fmt"
	"strings"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
)

type (
	OrdersCommand struct {
		*CommandMeta
	}
)

func (c *OrdersCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	var exchange model.Exchange
	if exchange, err = exchanges.GetExchange(); err != nil {
		return c.ReturnError(err)
	}

	var side = model.ORDER_SIDE_NONE
	flg = flag.Get("side")
	if flg.Exists {
		side = model.NewOrderSide(flg.String())
		if side == model.ORDER_SIDE_NONE {
			return c.ReturnError(errors.Errorf("side %v is invalid", flg))
		}
	}

	var prices = false
	flg = flag.Get("prices")
	if flg.Exists {
		prices = true
	}

	var kind model.OrderType = model.LIMIT
	flg = flag.Get("type")
	if flg.Exists {
		kind = model.NewOrderType(flg.String())
		if kind == model.ORDER_TYPE_NONE {
			return c.ReturnError(errors.Errorf("type %v is invalid", flg))
		}
	}

	flg = flag.Get("market")
	if !flg.Exists {
		return c.ReturnError(errors.New("missing argument: market"))
	}

	var market string
	if market, err = model.GetMarket(exchange); err != nil {
		return c.ReturnError(err)
	}

	var client interface{}
	if client, err = exchange.GetClient(model.PRIVATE, flag.Sandbox()); err != nil {
		return c.ReturnError(err)
	}

	var orders model.Orders

	// Get open orders for a symbol
	orders, err = exchange.GetOpened(client, market)

	// Filter orders
	var openSellOrders model.Orders
	var openBuyOrders model.Orders
	for _, order := range orders {
		if model.FormatOrderSide(order.Side) == "Sell" {
			openSellOrders = append(openSellOrders, order)
		} else {
			openBuyOrders = append(openBuyOrders, order)
		}
	}

	// Select orders according to flag "side"
	if side == model.BUY {
		orders = openBuyOrders
	} else if side == model.SELL {
		orders = openSellOrders
	}

	// Print order information
	for _, order := range orders {
		if prices {
			fmt.Printf("%f\n", order.Price)
		} else {
			fmt.Printf("%v\n", order)
		}
	}

	return 0
}

func (c *OrdersCommand) Help() string {
	text := `
Usage: ./nefertiti trades [options]

The orders command lists open orders of the market from the specified exchange.

Options:
  --exchange = name
  --side     = [buy|sell]
             = (optional)
  --market   = selects the market pair for which the info is queried
  --prices   = prints only the prices
`
	return strings.TrimSpace(text)
}

func (c *OrdersCommand) Synopsis() string {
	return "Lists your open orders for a market."
}

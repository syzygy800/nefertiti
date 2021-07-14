package command

import (
	"strings"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
)

type (
	CancelCommand struct {
		*CommandMeta
	}
)

func (c *CancelCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	var exchange model.Exchange
	if exchange, err = exchanges.GetExchange(); err != nil {
		return c.ReturnError(err)
	}

	var market string
	if market, err = model.GetMarket(exchange); err != nil {
		return c.ReturnError(err)
	}

	flg = flag.Get("side")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: side"))
	}
	side := model.NewOrderSide(flg.String())
	if side == model.ORDER_SIDE_NONE {
		return c.ReturnError(errors.Errorf("side %v is invalid", flg))
	}

	var client interface{}
	if client, err = exchange.GetClient(model.PRIVATE, flag.Sandbox()); err != nil {
		return c.ReturnError(err)
	}

	if err = exchange.Cancel(client, market, side); err != nil {
		return c.ReturnError(err)
	}

	return 0
}

func (c *CancelCommand) Help() string {
	text := `
Usage: ./nefertiti cancel [options]

The cancel command cancels all your buy or sell orders on a given market.

Options:
  --exchange = name
  --market   = a valid market pair
  --side     = [buy|sell]
`
	return strings.TrimSpace(text)
}

func (c *CancelCommand) Synopsis() string {
	return "Cancels all your buy or sell orders on a given market."
}

package command

import (
	"strings"

	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
)

type (
	CoalesceCommand struct {
		*CommandMeta
	}
)

func (c *CoalesceCommand) Run(args []string) int {
	exchange, err := exchanges.GetExchange()
	if err != nil {
		return c.ReturnError(err)
	}

	market, err := model.GetMarket(exchange)
	if err != nil {
		return c.ReturnError(err)
	}

	side, err := model.Side()
	if err != nil {
		return c.ReturnError(err)
	}

	client, err := exchange.GetClient(model.PRIVATE, flag.Sandbox())
	if err != nil {
		return c.ReturnError(err)
	}

	if market != "all" {
		if err := exchange.Coalesce(client, market, side); err != nil {
			return c.ReturnError(err)
		}
		return 0
	}

	markets, err := exchange.GetMarkets(true, flag.Sandbox(), flag.Get("ignore").Split())
	if err != nil {
		return c.ReturnError(err)
	}

	quotes := model.Assets(flag.Get("quote").Split())

	for _, market := range markets {
		if quotes.IsEmpty() || func() bool {
			for _, quote := range quotes {
				if strings.EqualFold(market.Quote, quote) {
					return true
				}
			}
			return false
		}() {
			if err := exchange.Coalesce(client, market.Name, side); err != nil {
				return c.ReturnError(err)
			}
		}
	}

	return 0
}

func (c *CoalesceCommand) Help() string {
	text := `
Usage: ./nefertiti coalesce [options]

The coalesce command joins orders having the same price.

Options:
  --exchange = name
  --market   = a valid market pair, or [all]
  --side     = [buy|sell]
`
	return strings.TrimSpace(text)
}

func (c *CoalesceCommand) Synopsis() string {
	return "Coalesce orders having the same price."
}

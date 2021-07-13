package command

import (
	"fmt"
	"strings"

	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
)

type (
	BaseCommand struct {
		*CommandMeta
	}
)

func (c *BaseCommand) Run(args []string) int {
	exchange, err := exchanges.GetExchange()
	if err != nil {
		return c.ReturnError(err)
	}

	markets, err := exchange.GetMarkets(true, flag.Sandbox())
	if err != nil {
		return c.ReturnError(err)
	}

	market, err := model.GetMarket(exchange)
	if err != nil {
		return c.ReturnError(err)
	}

	base, err := model.GetBaseCurr(markets, market)
	if err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(strings.ToUpper(base))

	return 0
}

func (c *BaseCommand) Help() string {
	text := `
Usage: ./nefertiti base [options]

The base command returns the base symbol for a given market pair.

Options:
  --exchange = name
  --market   = a valid market pair
`
	return strings.TrimSpace(text)
}

func (c *BaseCommand) Synopsis() string {
	return "Get the base symbol for a given market pair."
}

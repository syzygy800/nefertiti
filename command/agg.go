package command

import (
	"fmt"
	"strings"

	"github.com/svanas/nefertiti/aggregation"
	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
)

type (
	AggCommand struct {
		*CommandMeta
	}
)

func (c *AggCommand) Run(args []string) int {
	exchange, err := exchanges.GetExchange()
	if err != nil {
		return c.ReturnError(err)
	}

	market, err := model.GetMarket(exchange)
	if err != nil {
		return c.ReturnError(err)
	}

	dip, err := flag.Dip(5)
	if err != nil {
		return c.ReturnError(err)
	}

	pip, err := flag.Pip(30)
	if err != nil {
		return c.ReturnError(err)
	}

	max, err := flag.Max()
	if err != nil {
		return c.ReturnError(err)
	}

	min, err := flag.Min()
	if err != nil {
		return c.ReturnError(err)
	}

	dist, err := flag.Dist(2)
	if err != nil {
		return c.ReturnError(err)
	}

	agg, _, _, err := aggregation.Get(exchange, market, dip, pip, max, min, int(dist), 2, flag.Strict(), flag.Sandbox())
	if err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(agg)

	return 0
}

func (c *AggCommand) Help() string {
	text := `
Usage: ./nefertiti agg [options]

The agg command calculates the aggregation level for a given market pair.

Options:
  --exchange = name
  --market   = a valid market pair
  --dip      = percentage that will kick the bot into action.
               (optional, defaults to 5%)
  --pip      = range in where the market is suspected to move up and down.
               the bot will ignore supports outside of this range.
               (optional, defaults to 30%)
  --max      = maximum price that you will want to pay for the coins.
               (optional)
  --min      = minimum price that you will want to pay for the coins.
               (optional)
`
	return strings.TrimSpace(text)
}

func (c *AggCommand) Synopsis() string {
	return "Calculates the aggregation level for a given market pair."
}

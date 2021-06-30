package command

import (
	"errors"
	"fmt"
	"strings"

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

	var dip float64 = 5
	flg = flag.Get("dip")
	if flg.Exists {
		if dip, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("dip %v is invalid", flg))
		}
	}

	var pip float64 = 30
	flg = flag.Get("pip")
	if flg.Exists {
		if pip, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("pip %v is invalid", flg))
		}
	}

	var max float64 = 0
	flg = flag.Get("max")
	if flg.Exists {
		if max, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("max %v is invalid", flg))
		}
	}

	var min float64 = 0
	flg = flag.Get("min")
	if flg.Exists {
		if min, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("min %v is invalid", flg))
		}
	}

	var agg float64
	if agg, _, err = model.GetAgg(exchange, market, dip, pip, max, min, 4, flag.Strict(), sandbox); err != nil {
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

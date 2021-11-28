package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
)

type (
	TradesCommand struct {
		*CommandMeta
	}
)

func (c *TradesCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	var exchange model.Exchange
	if exchange, err = exchanges.GetExchange(); err != nil {
		return c.ReturnError(err)
	}

	flg = flag.Get("side")
	if !flg.Exists {
		return c.ReturnError(errors.New("missing argument: side"))
	}
	side := model.NewOrderSide(flg.String())
	if side == model.ORDER_SIDE_NONE {
		return c.ReturnError(errors.Errorf("side %v is invalid", flg))
	}

	var date time.Time
	flg = flag.Get("date")
	if !flg.Exists {
		date = time.Now()
	} else if strings.ToLower(flg.String()) == "all" {
		date = time.Time{}
	} else if strings.ToLower(flg.String()) == "y" {
		date = time.Now().AddDate(0, 0, -1)
	} else {
		date, err = time.Parse("2006-01-02", flg.String())
		if err != nil {
			return c.ReturnError(err)
		}
	}

	var verbose = false
	flg = flag.Get("verbose")
	if flg.Exists {
		verbose = true
	}

	var kind model.OrderType = model.LIMIT
	flg = flag.Get("type")
	if flg.Exists {
		kind = model.NewOrderType(flg.String())
		if kind == model.ORDER_TYPE_NONE {
			return c.ReturnError(errors.Errorf("type %v is invalid", flg))
		}
	}

	var market string
	if market, err = model.GetMarket(exchange); err != nil {
		return c.ReturnError(err)
	}

	var client interface{}
	if client, err = exchange.GetClient(model.PRIVATE, flag.Sandbox()); err != nil {
		return c.ReturnError(err)
	}

	return 0
}

func (c *TradesCommand) Help() string {
	text := `
Usage: ./nefertiti trades [options]

The trades command shows information about trades done with the specified exchange.
NOTE:
  Command is work in progress! Not everything works as described here, yet!
  The feature set might change (i.e. selecting weeks/month/year with --date)
  The date argument does not care about timezones (planned!).

Options:
  --exchange = name (currently Binance only!)
  --side     = [buy|sell] (Not used ATM)
  --market   = selects the market pair for which the info is queried
  --quote    = (NOT IMPLEMENTED YET) selects the markets by base currency
  --date     = the day of interest. Either 'Y' for yesterday or "YYYY-MM-DD". Default: Today
  --verbose  = show more detailed info
`
	return strings.TrimSpace(text)
}

func (c *TradesCommand) Synopsis() string {
	return "Query information about your trades."
}

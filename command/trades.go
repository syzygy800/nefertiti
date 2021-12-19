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

	var orders model.Orders

	// Get all closed orders
	orders, err = exchange.GetClosed(client, market)

	// Filter sell orders
	var filledsells model.Orders
	for _, order := range orders {
		if model.FormatOrderSide(order.Side) == "Sell" {
			if !date.IsZero() {
				if dateEqual(order.UpdatedAt, date) {
					filledsells = append(filledsells, order)
				}
			} else {
				filledsells = append(filledsells, order)
			}
		}
	}

	// Calc and output profits
	var totalProceeds = 0.0
	var timestring string
	for _, order := range filledsells {
		var proceeds = order.Size * (order.Price - order.BoughtAt)
		totalProceeds += proceeds

		if verbose {
			if date.IsZero() {
				timestring = order.UpdatedAt.Format("2006-01-02 15:04")
			} else {
				timestring = order.UpdatedAt.Format("15:04")
			}
			fmt.Printf("%s %s: %.2f\n", timestring, order.Market, proceeds)
		}
	}

	if verbose {
		fmt.Printf("Total: %.2f\n", totalProceeds)
	} else {
		fmt.Printf("%.2f\n", totalProceeds)
	}

	return 0
}

// Check if two days (year, month, day) are equal
func dateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()

	return y1 == y2 && m1 == m2 && d1 == d2
}

func (c *TradesCommand) Help() string {
	text := `
Usage: ./nefertiti trades [options]

The trades command shows information about trades done with the specified exchange.

WARNING:
Only for informational purpose. Some values are only estimated or plain wrong.
In no way is the displayed information useable for tax purposes!

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

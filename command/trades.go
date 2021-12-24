package command

import (
	"fmt"
	"strconv"
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

var (
	exchange model.Exchange
	date     time.Time
	verbose  = false
)

func (c *TradesCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

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

	var prec uint64 = 7
	flg = flag.Get("prec")
	if flg.Exists {
		prec, err = strconv.ParseUint(flg.String(), 10, 64)
		if err != nil {
			return c.ReturnError(err)
		}
	}

	// Flag --market takes precedence over --quote
	var symwidth = 9
	var symbols []string
	var quote string
	var market string
	flg = flag.Get("market")
	if flg.Exists {
		if market, err = model.GetMarket(exchange); err != nil {
			return c.ReturnError(err)
		} else {
			symbols = append(symbols, market)
			symwidth = len(market)
		}
	} else {
		flg = flag.Get("quote")
		if flg.Exists {
			quote = flg.String()
			symbols = getMarketsWithQuote(quote)
		} else {
			err = errors.New("missing argument: Either '--market' or '--quote' is mandantory")
		}
	}

	var client interface{}
	if client, err = exchange.GetClient(model.PRIVATE, flag.Sandbox()); err != nil {
		return c.ReturnError(err)
	}

	// Get all closed orders
	var orders model.Orders
	var filledsells model.Orders

	for _, m := range symbols {
		orders, err = exchange.GetClosed(client, m)
		if err != nil {
			fmt.Printf("%s ERROR: %v\n", m, err)
		}

		// Filter sell orders
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
	}

	// Calc and output profits
	var totalProceeds = 0.0
	var timestring string
	for _, order := range filledsells {
		var proceeds = order.Size * (order.Price - order.BoughtAt)
		if proceeds > 0 {
			totalProceeds += proceeds
		} else {
			if flag.Debug() {
				fmt.Printf("[DEBUG] Profit < 0! Order: %+v.", order)
			}
		}

		if verbose {
			if date.IsZero() {
				timestring = order.UpdatedAt.Format("2006-01-02 15:04")
			} else {
				timestring = order.UpdatedAt.Format("15:04")
			}
			if proceeds > 0 {
				fmt.Printf("%s %-*s %-.*f\n", timestring, symwidth, order.Market, prec, proceeds)
			}
		}
	}

	if verbose {
		fmt.Printf("Total: %.*f\n", prec, totalProceeds)
	} else {
		fmt.Printf("%.*f\n", prec, totalProceeds)
	}

	return 0
}

// Filter all exchange's markets by the quote currency
func getMarketsWithQuote(quote string) []string {
	var symbols []string
	var blacklist []string

	if len(quote) > 0 {
		markets, _ := exchange.GetMarkets(true, false, blacklist)
		for _, m := range markets {
			if m.Quote == quote {
				symbols = append(symbols, m.Name)
			}
		}
	}

	return symbols
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
  --quote    = selects the markets by base currency
  --date     = the day of interest. Either 'Y' for yesterday or "YYYY-MM-DD". Default: Today
  --mult     = use this multiplicator if sell order doesn't contain the buy price
  --prec     = specifies the number of decimal digits in the output. (optional, defaults to 7)
  --verbose  = show more detailed info
`
	return strings.TrimSpace(text)
}

func (c *TradesCommand) Synopsis() string {
	return "Query information about your trades."
}

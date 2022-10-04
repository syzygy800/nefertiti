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
	TickerCommand struct {
		*CommandMeta
	}
)

func (c *TickerCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	var exchange model.Exchange
	if exchange, err = exchanges.GetExchange(); err != nil {
		return c.ReturnError(err)
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

	// Get current ticker for a symbol
	ticker, err := exchange.GetTicker(client, market)
	if err != nil {
		return c.ReturnError(err)
	}

	var verbose = false
	flg = flag.Get("verbose")
	if flg.Exists {
		verbose = true
	}

	// Print order information
	if verbose {
		fmt.Printf("%s Value of %s is %f\n", time.Now().Format("2006-01-02 15:04:05"), market, ticker)
	} else {
		fmt.Println(ticker)
	}

	return 0
}

func (c *TickerCommand) Help() string {
	text := `
Usage: ./nefertiti ticker [options]

The current price of the market at the specified exchange.

Options:
  --exchange = name
  --market   = selects the market pair for which the info is queried
  --verbose  = show more detailed info
`
	return strings.TrimSpace(text)
}

func (c *TickerCommand) Synopsis() string {
	return "Displays the current ticker value for a market."
}

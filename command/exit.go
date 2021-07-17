package command

import (
	"fmt"
	"math"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/precision"
)

type (
	ExitCommand struct {
		*CommandMeta
	}
)

func exit(
	exchange model.Exchange,
	market string,
	startAtPrice float64,
	stopAtPrice float64,
	startWithSize float64,
	stopWithSize float64,
	test bool,
) error {
	steps := math.Round((stopAtPrice - startAtPrice) / ((stopAtPrice * stopWithSize) - (startAtPrice * startWithSize)))
	if steps < 1 {
		return errors.New("Cannot open any orders. Please widen your arguments")
	}

	client, err := exchange.GetClient(func() model.Permission {
		if test {
			return model.PUBLIC
		} else {
			return model.PRIVATE
		}
	}(), false)
	if err != nil {
		return err
	}

	orders, err := func() (model.Orders, error) {
		if test {
			return nil, nil
		} else {
			return exchange.GetOpened(client, market)
		}
	}()
	if err != nil {
		return err
	}

	hasLimitSell := func(size float64, price float64) bool {
		for _, order := range orders {
			if order.Side == model.SELL && order.Size == size && order.Price == price {
				return true
			}
		}
		return false
	}

	markets, err := exchange.GetMarkets(true, false)
	if err != nil {
		return err
	}

	baseCurr, err := model.GetBaseCurr(markets, market)
	if err != nil {
		return err
	}

	quoteCurr, err := model.GetQuoteCurr(markets, market)
	if err != nil {
		return err
	}

	sizePrec, err := exchange.GetSizePrec(client, market)
	if err != nil {
		return err
	}

	pricePrec, err := exchange.GetPricePrec(client, market)
	if err != nil {
		return err
	}

	sizeDeltaPerStep := precision.Round(((stopWithSize - startWithSize) / steps), sizePrec)
	priceDeltaPerStep := precision.Round(((stopAtPrice - startAtPrice) / steps), pricePrec)

	currSize := stopWithSize
	currPrice := stopAtPrice

	var (
		totalSize     float64
		totalProceeds float64
	)

	tbl := table.NewWriter()
	tbl.AppendHeader(table.Row{"", "Price", "Size", "Proceeds"})

	for currPrice >= startAtPrice {
		if !test {
			ticker, err := exchange.GetTicker(client, market)
			if err != nil {
				return err
			}
			if currPrice <= ticker {
				break
			}
		}

		totalSize += currSize
		totalProceeds += currPrice * currSize

		tbl.AppendRow(table.Row{"",
			fmt.Sprintf("%.[2]*[1]f %[3]v", currPrice, pricePrec, baseCurr),
			fmt.Sprintf("%.[2]*[1]f", currSize, sizePrec),
			fmt.Sprintf("%.[2]*[1]f %[3]v", (currPrice * currSize), pricePrec, quoteCurr),
		})

		if !test {
			if !hasLimitSell(currSize, currPrice) {
				if _, _, err := exchange.Order(client, model.SELL, market, currSize, currPrice, model.LIMIT); err != nil {
					return err
				}
			}
		}

		currSize = precision.Round((currSize - sizeDeltaPerStep), sizePrec)
		currPrice = precision.Round((currPrice - priceDeltaPerStep), pricePrec)
	}

	tbl.AppendSeparator()
	tbl.AppendRow(table.Row{"TOTAL", "",
		fmt.Sprintf("%.[2]*[1]f", totalSize, sizePrec),
		fmt.Sprintf("%.[2]*[1]f %[3]v", totalProceeds, pricePrec, quoteCurr),
	})

	fmt.Println(tbl.Render())

	return nil
}

func startAtPrice() (float64, error) {
	var (
		err error
		out float64
	)
	arg := flag.Get("start-at-price")
	if !arg.Exists {
		return out, errors.New("missing argument: start-at-price")
	}
	if out, err = arg.Float64(); err != nil {
		return out, errors.Errorf("start-at-price %v is invalid", arg)
	}
	return out, nil
}

func stopAtPrice() (float64, error) {
	var (
		err error
		out float64
	)
	arg := flag.Get("stop-at-price")
	if !arg.Exists {
		return out, errors.New("missing argument: stop-at-price")
	}
	if out, err = arg.Float64(); err != nil {
		return out, errors.Errorf("stop-at-price %v is invalid", arg)
	}
	return out, nil
}

func startWithSize() (float64, error) {
	var (
		err error
		out float64
	)
	arg := flag.Get("start-with-size")
	if !arg.Exists {
		return out, errors.New("missing argument: start-with-size")
	}
	if out, err = arg.Float64(); err != nil {
		return out, errors.Errorf("start-with-size %v is invalid", arg)
	}
	return out, nil
}

func stopWithSize() (float64, error) {
	var (
		err error
		out float64
	)
	arg := flag.Get("stop-with-size")
	if !arg.Exists {
		return out, errors.New("missing argument: stop-with-size")
	}
	if out, err = arg.Float64(); err != nil {
		return out, errors.Errorf("stop-with-size %v is invalid", arg)
	}
	return out, nil
}

// examples:
// ./nefertiti exit --exchange=GDAX --market=BTC-USDC --start-at-price=200000 --stop-at-price=240000 --start-with-size=0.015 --stop-with-size=0.020
// ./nefertiti exit --exchange=GDAX --market=BTC-USDC --start-at-price=100000 --stop-at-price=200000 --start-with-size=0.010 --stop-with-size=0.015
func (c *ExitCommand) Run(args []string) int {
	exchange, err := exchanges.GetExchange()
	if err != nil {
		return c.ReturnError(err)
	}

	market, err := model.GetMarket(exchange)
	if err != nil {
		return c.ReturnError(err)
	}

	startAtPrice, err := startAtPrice()
	if err != nil {
		return c.ReturnError(err)
	}

	stopAtPrice, err := stopAtPrice()
	if err != nil {
		return c.ReturnError(err)
	}

	if startAtPrice > stopAtPrice {
		stopAtPrice, startAtPrice = startAtPrice, stopAtPrice
	}

	startWithSize, err := startWithSize()
	if err != nil {
		return c.ReturnError(err)
	}

	stopWithSize, err := stopWithSize()
	if err != nil {
		return c.ReturnError(err)
	}

	if startWithSize > stopWithSize {
		stopWithSize, startWithSize = startWithSize, stopWithSize
	}

	if err = exit(exchange, market, startAtPrice, stopAtPrice, startWithSize, stopWithSize, !flag.Exists("not-a-drill")); err != nil {
		return c.ReturnError(err)
	}

	return 0
}

func (c *ExitCommand) Help() string {
	return "Usage: ./nefertiti exit"
}

func (c *ExitCommand) Synopsis() string {
	return "Exit the specified exchange/market."
}

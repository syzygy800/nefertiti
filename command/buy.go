package command

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/svanas/nefertiti/command/buy"
	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/exchanges"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/logger"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/multiplier"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/signals"
)

type (
	BuyCommand struct {
		*CommandMeta
	}
)

func (c *BuyCommand) Run(args []string) int {
	exchange, err := exchanges.GetExchange()
	if err != nil {
		return c.ReturnError(err)
	}

	test := flag.Exists("test")

	client, err := exchange.GetClient(model.PRIVATE, flag.Sandbox())
	if err != nil {
		return c.ReturnError(err)
	}

	notifier, err := func() (model.Notify, error) {
		if test {
			return nil, nil
		}
		return notify.New().Init(flag.Interactive(), true)
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	level, err := notify.Level()
	if err != nil {
		return c.ReturnError(err)
	}

	min, err := flag.Min()
	if err != nil {
		return c.ReturnError(err)
	}

	// --volume=X
	volume, err := func() (float64, error) {
		arg := flag.Get("volume")
		if arg.Exists {
			out, err := arg.Float64()
			if err != nil {
				return 0, errors.Errorf("volume %v is invalid", arg)
			}
			return out, nil
		}
		return 0, nil
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	// --devn=X
	devn, err := func() (float64, error) {
		arg := flag.Get("devn")
		if arg.Exists {
			out, err := arg.Float64()
			if err != nil {
				return 0, errors.Errorf("devn %v is invalid", arg)
			}
			return out, nil
		}
		return 1, nil
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	arg := flag.Get("signals")
	if arg.Exists {
		channel, err := func() (model.Channel, error) {
			out, err := signals.New().FindByName(arg.String())
			if err != nil {
				return nil, err
			}
			if out == nil {
				return nil, errors.Errorf("signals %v does not exist", arg)
			}
			return out, nil
		}()
		if err != nil {
			return c.ReturnError(err)
		}
		// --price=X
		price, err := func() (float64, error) {
			arg := flag.Get("price")
			if !arg.Exists {
				return 0, errors.New("missing argument: price")
			}
			out, err := arg.Float64()
			if err != nil {
				return 0, errors.Errorf("price %v is invalid", arg)
			}
			return out, nil
		}()
		if err != nil {
			return c.ReturnError(err)
		}
		// --valid=X
		valid, err := func() (time.Duration, error) {
			out, err := channel.GetValidity()
			if err != nil {
				return 0, err
			}
			arg := flag.Get("valid")
			if arg.Exists {
				valid, err := arg.Float64()
				if err != nil {
					return 0, errors.Errorf("valid %v is incorrect", arg)
				}
				out = time.Duration(valid * float64(time.Hour))
			}
			return out, nil
		}()
		if err != nil {
			return c.ReturnError(err)
		}
		// initial run starts here
		calls, err := buy.Signals(channel, client, exchange, price, valid, nil, min, volume, devn, notifier, level, test)
		if err != nil {
			if flag.Get("ignore").Contains("error") {
				log.Printf("[ERROR] %v\n", err)
			} else {
				return c.ReturnError(err)
			}
		}
		// iterations start here
		if !test {
			// --repeat=X
			repeat, err := func() (time.Duration, error) {
				arg := flag.Get("repeat")
				if arg.Exists {
					out, err := arg.Float64()
					if err != nil {
						return 0, errors.Errorf("repeat %v is invalid", arg)
					}
					return time.Duration(out * float64(time.Hour)), nil
				}
				return 0, nil
			}()
			if err != nil {
				return c.ReturnError(err)
			}
			if repeat > 0 {
				limit := channel.GetRateLimit()
				if limit > 0 {
					if repeat < limit {
						if flag.Exists("backdoor") {
							// bypass the rate limit
						} else {
							return c.ReturnError(errors.Errorf("repeat is invalid. min value is %g", (float64(limit) / float64(time.Hour))))
						}
					}
				}
				logger.InfoEx(
					(exchange.GetInfo().Name + " - INFO"),
					fmt.Sprintf("Listening to %s...", channel.GetName()),
					level, notifier,
				)
				if err = c.ReturnSuccess(); err != nil {
					return c.ReturnError(err)
				}
				buy.SignalsEvery(repeat, channel, client, exchange, price, valid, calls, min, volume, devn, notifier, level)
			}
		}
		return 0
	}

	all, err := exchange.GetMarkets(true, flag.Sandbox(), flag.Get("ignore").Split())
	if err != nil {
		return c.ReturnError(err)
	}

	// --market=X,Y,Z
	markets, err := func() ([]string, error) {
		arg := flag.Get("market")
		if !arg.Exists {
			return nil, errors.New("missing argument: market")
		}
		out := arg.Split()
		if len(out) > 1 || (len(out) == 1 && out[0] != "all") {
			for _, market := range out {
				if !model.HasMarket(all, market) {
					return nil, errors.Errorf("market %s does not exist", market)
				}
			}
		}
		return out, nil
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	// --hold=X,Y,Z
	hold := flag.Get("hold").Split()
	if len(hold) > 0 && hold[0] != "" {
		for _, market := range hold {
			if market != "" && !model.HasMarket(all, market) {
				return c.ReturnError(fmt.Errorf("market %s does not exist", market))
			}
		}
	}

	// --agg=X
	agg, err := func() (float64, error) {
		arg := flag.Get("agg")
		if arg.Exists {
			out, err := arg.Float64()
			if err != nil {
				return 0, errors.Errorf("agg %v is invalid", arg)
			}
			return out, nil
		}
		return 0, nil
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	size, price, err := func() (float64, float64, error) {
		var (
			err  error
			arg  *flag.Flag
			out1 float64 = 0
			out2 float64 = 0
		)
		// if we have an arg named --size, then that one will take precedence
		arg = flag.Get("size")
		if arg.Exists {
			if out1, err = arg.Float64(); err != nil {
				return 0, 0, errors.Errorf("size %v is invalid", arg)
			}
		} else {
			// if we have an arg named --price, then we'll calculate the desired size later
			arg = flag.Get("price")
			if !arg.Exists {
				return 0, 0, errors.New("missing argument: size")
			} else {
				if out2, err = arg.Float64(); err != nil {
					return 0, 0, errors.Errorf("price %v is invalid", arg)
				}
			}
		}
		return out1, out2, nil
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	// --dip-5
	dip, err := flag.Dip(5)
	if err != nil {
		return c.ReturnError(err)
	}

	// --pip=30
	pip, err := flag.Pip(30)
	if err != nil {
		return c.ReturnError(err)
	}

	// --mult=1.05
	mult, err := multiplier.Get(multiplier.FIVE_PERCENT)
	if err != nil {
		return c.ReturnError(err)
	}

	// --dist=2
	dist, err := flag.Dist(2)
	if err != nil {
		return c.ReturnError(err)
	}

	// --top=2
	top, err := func() (int64, error) {
		arg := flag.Get("top")
		if arg.Exists {
			out, err := arg.Int64()
			if err != nil {
				return 0, errors.Errorf("top %v is invalid", arg)
			}
			return out, nil
		}
		return 2, nil
	}()
	if err != nil {
		return c.ReturnError(err)
	}

	// --max=X
	max, err := flag.Max()
	if err != nil {
		return c.ReturnError(err)
	}

	if err, _ = buy.Standard(client, exchange, markets, hold, agg, size, dip, pip, mult, dist, top, max, min, price, volume, devn, notifier, level, test); err != nil {
		if flag.Get("ignore").Contains("error") {
			log.Printf("[ERROR] %v\n", err)
		} else {
			return c.ReturnError(err)
		}
	}

	if !test {
		// --repeat=X
		repeat, err := func() (time.Duration, error) {
			arg := flag.Get("repeat")
			if arg.Exists {
				out, err := arg.Float64()
				if err != nil {
					return 0, errors.Errorf("repeat %v is invalid", arg)
				}
				return time.Duration(out * float64(time.Hour)), nil
			}
			return 0, nil
		}()
		if err != nil {
			return c.ReturnError(err)
		}
		if repeat > 0 {
			if err = c.ReturnSuccess(); err != nil {
				return c.ReturnError(err)
			}
			buy.StandardEvery(repeat, client, exchange, markets, hold, agg, size, dip, pip, mult, dist, top, max, min, price, volume, devn, notifier, level)
		}
	}

	return 0
}

func (c *BuyCommand) Help() string {
	text := `
Usage: ./nefertiti buy [options]

The buy command opens new limit buy orders on the specified exchange/market(s).

Global options:
  --exchange = name, for example: Bittrex
  --price    = the price in quote asset you will want to pay for one order.
  --volume   = minimum BTC volume on the market(s) over the last 24 hours.
               (optional, for example: --volume=10)
  --min      = minimum price that you will want to pay for the base asset.
               (optional, for example: --min=0.00000050)
  --dca      = if included, then slowly but surely, the bot will proportionally
               increase your bag while lowering your average buying price.
               (optional, defaults to false)
  --repeat   = if included, repeats this command every X hours.
               (optional, for example: --repeat=1)	

Standard (built-in) strategy:
  --market   = a comma-separated list of valid market pair(s).
  --dip      = percentage that will kick the bot into action.
               (optional, defaults to 5%)
  --pip      = range in where the market is suspected to move up and down.
               the bot will ignore supports outside of this range.
               (optional, defaults to 30%)
  --dist     = distribution/distance between your orders.
               (optional, defaults to 2%)
  --top      = number of orders to place in your book.
               (optional, defaults to 2)   
  --max-orders=if included, then no buy order is set, when there are already
               [max-orders] open sell orders for the market
               (optional, default to 0 -> check disabled)
  --test     = if included, merely reports what it would do.
               (optional, defaults to false)

Alternative strategy:
  The trading bot can listen to signals (for example: Telegram bots) as an
  alternative to the built-in strategy. Please refer to the below options.

Alternative strategy options:
  --signals  = provider, for example: MiningHamster
  --quote    = asset that is used as the reference, for example: BTC or USDT              
  --devn     = buy price deviation. this multiplier is applied to the suggested
               price from the signal, to calculate your actual limit price.
               (optional, for example: --devn=1.01)
  --valid    = if included, specifies the time (in hours, defaults to 1 hour)
               that the signal is "active" for. after this timeout elapses, the
               bot will cancel the (non-filled) limit buy order(s) associated
               with the signal.
               (optional, defaults to 1 hour)
`
	return strings.TrimSpace(text)
}

func (c *BuyCommand) Synopsis() string {
	return "Opens new limit buy orders on the specified exchange/market(s)."
}

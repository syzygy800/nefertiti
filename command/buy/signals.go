package buy

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/logger"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/precision"
)

func SignalsEvery(
	repeat time.Duration,
	channel model.Channel,
	client interface{},
	exchange model.Exchange,
	price float64,
	valid time.Duration,
	calls model.Calls,
	min,
	volume,
	devn float64,
	notifier model.Notify,
	level int64,
) {
	var err error
	for range time.Tick(repeat) {
		calls, err = Signals(channel, client, exchange, price, valid, calls, min, volume, devn, notifier, level, false)
		if err != nil {
			logger.Error(
				exchange.GetInfo().Name,
				errors.Append(err, fmt.Sprintf("Channel: %s", channel.GetName())),
				level, notifier,
			)
		}
	}
}

func Signals(
	channel model.Channel,
	client interface{},
	exchange model.Exchange,
	price float64,
	valid time.Duration,
	old model.Calls,
	min,
	volume,
	devn float64,
	notifier model.Notify,
	level int64,
	test bool,
) (model.Calls, error) {
	var (
		err error
		new model.Calls
	)

	quote := model.Assets(flag.Get("quote").Split())
	if quote.IsEmpty() {
		return old, errors.New("missing argument: quote")
	}

	all, err := exchange.GetMarkets(true, flag.Sandbox(), flag.Get("ignore").Split())
	if err != nil {
		return old, err
	}

	markets, err := channel.GetMarkets(exchange, quote, volume, valid, flag.Sandbox(), flag.Debug(), flag.Get("ignore").Split())
	if err != nil {
		return old, err
	}

	// allow for signals to buy new listings
	for _, market := range markets {
		if !model.HasMarket(all, market) {
			if all, err = exchange.GetMarkets(false, flag.Sandbox(), flag.Get("ignore").Split()); err != nil {
				return old, err
			}
			break
		}
	}

	for _, market := range markets {
		if !model.HasMarket(all, market) {
			if flag.Debug() {
				log.Printf("[DEBUG] %s trading is not available in your region.\n", market)
			}
		} else {
			if flag.Get("ignore").Contains("leveraged") {
				base, err := model.GetBaseCurr(all, market)
				if err == nil {
					if exchange.IsLeveragedToken(base) {
						log.Printf("[INFO] Ignoring %s because %s is a leveraged token.\n", market, base)
						continue
					}
				}
			}

			ticker, err := exchange.GetTicker(client, market)
			if err != nil {
				return old, err
			}

			prec, err := exchange.GetSizePrec(client, market)
			if err != nil {
				return old, err
			}

			calls, err := channel.GetCalls(exchange, market, flag.Sandbox(), flag.Debug())
			if err != nil {
				return old, err
			}

			for i := range calls {
				calls[i].Size = precision.Round(price/ticker, prec)

				if flag.Dca() {
					hasOpenSell := 0
					opened, err := exchange.GetOpened(client, market)
					if err != nil {
						return old, err
					}
					for _, order := range opened {
						if order.Side == model.SELL {
							hasOpenSell++
						}
					}
					calls[i].Size = precision.Round((calls[i].Size * (1 + (float64(hasOpenSell) * 0.2))), prec)
				}

				if !calls[i].Skip {
					corrupt, reason := calls[i].Corrupt(channel.GetOrderType())
					if corrupt {
						calls[i].Ignore(reason)
					}
				}
			}

			if min > 0 {
				for i := range calls {
					if !calls[i].Skip {
						limit := func() float64 {
							if calls[i].Price == 0 {
								return ticker
							}
							return calls[i].Price
						}()
						if limit < min {
							calls[i].Ignore("price %.8f is lower than %.8f", calls[i].Price, min)
						}
					}
				}
			}

			if volume > 0 {
				for i := range calls {
					if !calls[i].Skip {
						stats, err := exchange.Get24h(client, calls[i].Market)
						if err != nil {
							return old, err
						}
						if stats.BtcVolume > 0 && stats.BtcVolume < volume {
							calls[i].Ignore("volume %.2f is lower than %.2f %s", stats.BtcVolume, volume, quote)
						}
					}
				}
			}

			if len(calls) > 0 {
				new = append(new, calls...)
				if !test {
					// do not (re)open buy orders for signals that we already know about
					if old != nil {
						for i := range calls {
							if !calls[i].Skip {
								if func() int {
									if channel.GetOrderType() == model.MARKET {
										return old.IndexByMarket(calls[i].Market)
									} else {
										return old.IndexByMarketPrice(calls[i].Market, calls[i].Price)
									}
								}() > -1 {
									calls[i].Skip = true
								}
							}
						}
					}
					// log the signals that we are skipping. possible reasons are: they are corrupt, or they don't meet your settings
					for _, call := range calls {
						if call.Skip && call.Reason != "" {
							logger.Info("Ignoring %s. Reason: %s", call.Market, call.Reason)
						}
					}
					// buy the signals (if we got any at this point)
					if calls.HasAnythingToDo() {
						err = exchange.Buy(client, false, market, calls, devn, channel.GetOrderType())
						if err != nil {
							logger.Error(
								exchange.GetInfo().Name,
								errors.Append(
									errors.Append(err,
										fmt.Sprintf("Market: %s", market),
									), fmt.Sprintf("Channel: %s", channel.GetName())),
								level, notifier,
							)
						}
					}
				}
			}
		}
	}

	// cancel open orders that are removed from the channel
	for _, entry := range old {
		if func() int {
			if channel.GetOrderType() == model.MARKET {
				return new.IndexByMarket(entry.Market)
			} else {
				return new.IndexByMarketPrice(entry.Market, entry.Price)
			}
		}() == -1 {
			if buf, err := json.Marshal(entry); err == nil {
				log.Println("[CANCELLED] " + string(buf))
			}
			if err = exchange.Cancel(client, entry.Market, model.BUY); err != nil {
				return new, err
			}
		}
	}

	// echo the new book to the console
	if len(new) > 0 {
		out, err := json.Marshal(new)
		if err != nil {
			return new, err
		}
		log.Println(string(out))
	} else {
		log.Println("[INFO] No new signals available.")
	}

	return new, nil
}

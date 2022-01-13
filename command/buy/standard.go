package buy

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/svanas/nefertiti/aggregation"
	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/logger"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/multiplier"
	"github.com/svanas/nefertiti/precision"
	"github.com/svanas/nefertiti/pricing"
)

func StandardEvery(
	repeat time.Duration,
	client interface{},
	exchange model.Exchange,
	markets []string,
	hold model.Markets,
	agg float64,
	size float64,
	dip float64,
	pip float64,
	mult multiplier.Mult,
	dist,
	top int64,
	max,
	min,
	price,
	volume,
	devn float64,
	notifier model.Notify,
	level int64,
) {
	for range time.Tick(repeat) {
		err, market := Standard(client, exchange, markets, hold, agg, size, dip, pip, mult, dist, top, max, min, price, volume, devn, notifier, level, false)
		if err != nil {
			logger.Error(
				exchange.GetInfo().Name,
				errors.Append(err, fmt.Sprintf("Market: %s", market)),
				level, notifier,
			)
		}
	}
}

func Standard(
	client interface{},
	exchange model.Exchange,
	markets []string,
	hold model.Markets,
	agg,
	size,
	dip,
	pip float64,
	mult multiplier.Mult,
	dist,
	top int64,
	max,
	min,
	price,
	volume,
	devn float64,
	notifier model.Notify,
	level int64,
	test bool,
	//lint:ignore ST1008 error should be returned as the last argument
) (error, string) { // -> (error, market)
	var err error

	// true if we're told to open buys for every market, otherwise false.
	all := len(markets) == 1 && markets[0] == "all"

	// all markets that are available to us
	available, err := exchange.GetMarkets(!all, flag.Sandbox(), flag.Get("ignore").Split())
	if err != nil {
		return err, ""
	}

	// the markets we enumerate/buy
	enumerable, err := func() ([]string, error) {
		if !all {
			return markets, nil
		} else {
			quote := flag.Get("quote").String()
			if quote == "" {
				return nil, errors.New("missing argument: quote")
			}
			var out []string
			for _, market := range available {
				if strings.EqualFold(market.Quote, quote) {
					out = append(out, market.Name)
				}
			}
			return out, nil
		}
	}()
	if err != nil {
		return err, ""
	}

	for _, market := range enumerable {
		// "algo" orders are...
		// 1) stop-loss, and
		// 2) take-profit, and
		// 3) OCO (aka one-cancels-the-other)
		if hasAlgoOrder, _ := exchange.HasAlgoOrder(client, market); hasAlgoOrder {
			log.Printf("[INFO] Ignoring %s because you have at least one \"algo\" order open on this market.\n", market)
			continue
		}

		ticker, err := exchange.GetTicker(client, market)
		if err != nil {
			return err, market
		}

		stats, err := exchange.Get24h(client, market) // 24-hour statistics
		if err != nil {
			return err, market
		}

		if stats.BtcVolume > 0 && stats.BtcVolume < volume {
			log.Printf("[INFO] Ignoring %s because volume %.2f is lower than %.2f BTC\n", market, stats.BtcVolume, volume)
			continue
		}

		avg, err := stats.Avg(exchange, flag.Sandbox()) // 24-hour average
		if err != nil {
			return err, market
		}

		pricePrec, err := exchange.GetPricePrec(client, market)
		if err != nil {
			return err, market
		}

		magg := agg
		mdip := dip
		mpip := pip
		mmin := min
		mmax := max

		hasOpenSell := 0
		// ignore supports where the price is higher than BUY order(s) that were (a) filled and (b) not been sold (yet)
		if !test {
			opened, err := exchange.GetOpened(client, market)
			if err != nil {
				return err, market
			}
			for _, order := range opened {
				if order.Side == model.SELL {
					hasOpenSell++
				}
			}
			// step 1: loop through the filled BUY orders
			closed, err := exchange.GetClosed(client, market)
			if err != nil {
				return err, market
			}
			for _, fill := range closed {
				if fill.Side == model.BUY {
					// step 2: has this filled BUY order NOT been sold?
					if opened.IndexByPrice(model.SELL, market, pricing.Multiply(fill.Price, mult, pricePrec)) > -1 {
						if mmax == 0 || mmax >= fill.Price {
							mmax = fill.Price
						}
					}
				}
			}
		}

		if magg == 0 {
			if magg, mdip, mpip, err = aggregation.GetEx(exchange, client, market, ticker, avg, dip, pip, mmax, min, int(dist), pricePrec, int(top), flag.Strict()); err != nil {
				if errors.Is(err, aggregation.ECannotFindSupports) && (len(enumerable) > 1 || flag.Get("ignore").Contains("error")) {
					logger.Error(
						exchange.GetInfo().Name,
						errors.Append(err, fmt.Sprintf("Market: %s", market)),
						level, notifier,
					)
					continue
				} else {
					return err, market
				}
			}
		}

		book1, err := exchange.GetBook(client, market, model.BOOK_SIDE_BIDS)
		if err != nil {
			return err, market
		}

		book2, err := exchange.Aggregate(client, book1, market, magg)
		if err != nil {
			return err, market
		}

		// ignore supports that are more expensive than ticker
		i := 0
		for i < len(book2) {
			if book2[i].Price > ticker {
				book2 = append(book2[:i], book2[i+1:]...)
			} else {
				i++
			}
		}

		// ignore supports that are cheaper than ticker minus 30%
		if mmin == 0 && mpip < 100 {
			mmin = ticker - ((mpip / 100) * ticker)
		}
		if mmin > 0 {
			i = 0
			for i < len(book2) {
				if book2[i].Price < mmin {
					book2 = append(book2[:i], book2[i+1:]...)
				} else {
					i++
				}
			}
		}

		// ignore supports that are more expensive than 24h average minus 5%
		if mdip > 0 {
			i = 0
			for i < len(book2) {
				if book2[i].Price > (avg - ((mdip / 100) * avg)) {
					book2 = append(book2[:i], book2[i+1:]...)
				} else {
					i++
				}
			}
		}

		// ignore supports that are more expensive than max (optional)
		if mmax > 0 {
			i = 0
			for i < len(book2) {
				if book2[i].Price >= mmax {
					book2 = append(book2[:i], book2[i+1:]...)
				} else {
					i++
				}
			}
		}

		// sort the order book by size (highest order size first)
		sort.Slice(book2, func(i1, i2 int) bool {
			return book2[i1].Size > book2[i2].Size
		})

		// we need at least one support
		if len(book2) == 0 {
			if len(enumerable) > 1 || flag.Get("ignore").Contains("error") {
				logger.Error(
					exchange.GetInfo().Name,
					errors.Append(aggregation.ECannotFindSupports, fmt.Sprintf("Market: %s", market)),
					level, notifier,
				)
				continue
			} else {
				return aggregation.ECannotFindSupports, market
			}
		}

		sizePrec, err := exchange.GetSizePrec(client, market)
		if err != nil {
			return err, market
		}

		base, err := model.GetBaseCurr(available, market)
		if err != nil {
			return err, market
		}

		for i := 0; i < len(book2); i++ {
			book2[i].Size = size

			// if we have an arg named --price, then we'll calculate the desired size here
			if price != 0 {
				book2[i].Size = precision.Round((price / book2[i].Price), sizePrec)
			}

			// the more non-sold sell orders we have, the bigger the new buy order size
			if flag.Dca() {
				book2[i].Size = precision.Round((book2[i].Size * (1 + (float64(hasOpenSell) * 0.2))), sizePrec)
			}

			// for BTC and ETH, there is a minimum size (otherwise, we would never be hodl'ing)
			units := model.GetSizeMin(hold.HasMarket(market), base)
			if book2[i].Size < units {
				return errors.Errorf("Cannot buy %s. Size is too low. You must buy at least %f units.", market, units), market
			}
		}

		// convert the aggregated order book into signals for the exchange to buy
		calls := func() model.Calls {
			if len(book2) < int(top) {
				return book2.Calls()
			} else {
				return book2[:top].Calls()
			}
		}()

		// log the supports that are corrupt (if any). possible reasons are: qty is zero, or price is zero, or both are zero.
		for _, call := range calls {
			corrupt, reason := call.Corrupt(model.LIMIT)
			if corrupt {
				call.Skip = true
				logger.Info("Ignoring %s. Reason: %s", call.Market, reason)
			}
		}

		// cancel your open buy order(s), then place the top X buy orders
		if !test {
			err = exchange.Buy(client, true, market, calls, devn, model.LIMIT)
			if err != nil {
				if len(enumerable) > 1 || flag.Get("ignore").Contains("error") {
					logger.Error(
						exchange.GetInfo().Name,
						errors.Append(err, fmt.Sprintf("Market: %s", market)),
						level, notifier,
					)
					continue
				} else {
					return err, market
				}
			}
		}

		out, err := func() ([]byte, error) {
			if len(book2) < int(top) {
				return json.Marshal(book2)
			} else {
				return json.Marshal(book2[:top])
			}
		}()
		if err != nil {
			return err, market
		}

		log.Println(string(out))
	}

	return nil, ""
}

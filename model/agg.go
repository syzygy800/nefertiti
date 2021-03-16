package model

import (
	"math"

	"github.com/go-errors/errors"
	"github.com/svanas/nefertiti/pricing"
)

var (
	EYouAreAskingTooMuch = errors.New("Cannot find any supports. Please ask for fewer supports, or change your other settings.")
	EOrderBookTooThin    = errors.New("Cannot find any supports. Order book is too thin. Please reconsider this market.")
)

// returns (agg, dip, error)
func GetAgg(exchange Exchange, market string, dip, pip, max, min float64, top int, strict, sandbox bool) (float64, float64, error) {
	var (
		err error
		out float64
	)
	Max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	for cnt := Max(top, 4); cnt > 0; cnt-- {
		out, err = getAgg(exchange, market, dip, pip, max, min, cnt, sandbox)
		if err == nil {
			return out, dip, err
		} else {
			if errors.Is(err, EOrderBookTooThin) {
				return out, dip, err
			}
		}
	}
	if !strict && dip > 0 {
		n := math.Round(dip) - 1
		if n > 0 {
			for i := n; i >= 0; i-- {
				for cnt := Max(top, 4); cnt >= top; cnt-- {
					out, err = getAgg(exchange, market, i, pip, max, min, cnt, sandbox)
					if err == nil {
						return out, i, err
					} else {
						if errors.Is(err, EOrderBookTooThin) {
							return out, i, err
						}
					}
				}
			}
		}
	}
	if err != nil {
		return 0, dip, err
	} else {
		return 0, dip, EOrderBookTooThin
	}
}

func getAgg(exchange Exchange, market string, dip, pip, max, min float64, cnt int, sandbox bool) (float64, error) {
	var (
		ok  bool
		err error
	)

	var steps = [...]float64{
		0.5, // 250
		0.4, // 100
		0.5, // 50
		0.5, // 25
		0.8, // 20
		0.5, // 10
		0.5, // 5
	}

	var client interface{}
	if client, err = exchange.GetClient(false, sandbox); err != nil {
		return 0, err
	}

	var ticker float64
	if ticker, err = exchange.GetTicker(client, market); err != nil {
		return 0, err
	}

	var stats *Stats
	if stats, err = exchange.Get24h(client, market); err != nil {
		return 0, err
	}

	var avg float64
	if avg, err = stats.Avg(exchange, sandbox); err != nil {
		return 0, err
	}

	var book1 interface{}
	if book1, err = exchange.GetBook(client, market, BOOK_SIDE_BIDS); err != nil {
		return 0, err
	}

	var agg float64 = 500
	for {
		for _, step := range steps {
			agg = pricing.RoundToPrecision(agg*step, 8)

			var book2 Book
			if book2, err = exchange.Aggregate(client, book1, market, agg); err != nil {
				return 0, err
			}

			// ignore orders that are more expense than ticker
			i := 0
			for i < len(book2) {
				if book2[i].Price > ticker {
					book2 = append(book2[:i], book2[i+1:]...)
				} else {
					i++
				}
			}

			// ignore orders that are cheaper than ticker minus 30%
			if min == 0 {
				min = ticker - ((pip / 100) * ticker)
			}
			if min > 0 {
				i = 0
				for i < len(book2) {
					if book2[i].Price < min {
						book2 = append(book2[:i], book2[i+1:]...)
					} else {
						i++
					}
				}
			}

			// ignore orders that are more expensive than 24h high minus 5%
			if dip > 0 {
				i = 0
				for i < len(book2) {
					if book2[i].Price > (avg - ((dip / 100) * avg)) {
						book2 = append(book2[:i], book2[i+1:]...)
					} else {
						i++
					}
				}
			}

			// ignore BUY orders that are more expensive than max (optional)
			if max > 0 {
				i = 0
				for i < len(book2) {
					if book2[i].Price > max {
						book2 = append(book2[:i], book2[i+1:]...)
					} else {
						i++
					}
				}
			}

			// we need at least 4 supports
			if len(book2) > 0 {
				ok = true
			}
			if len(book2) >= cnt {
				return agg, nil
			}

			// since we're rounding to 8 decimals, prevent this func from getting stuck in an infinite loop
			if agg <= 0.00000001 {
				if len(book2) > 0 {
					return agg, nil
				} else {
					if !ok && (dip == 0) {
						return 0, EOrderBookTooThin
					} else {
						return 0, EYouAreAskingTooMuch
					}
				}
			}
		}
	}
}

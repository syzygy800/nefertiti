package aggregation

import (
	"math"
	"strconv"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/precision"
)

var (
	ECannotFindSupports = errors.New("Cannot find any supports. Please update your settings. Or reconsider this market because it might be illiquid.")
)

// rounds [input] to to nearest multiple of [agg]
func Round(input, agg float64) float64 {
	return float64(int64(((input / agg) + 0.5))) * agg
}

// returns (agg, dip, pip, error)
func Get(
	exchange model.Exchange,
	market string,
	dip, pip float64,
	max, min float64,
	dist, top int,
	strict, sandbox bool,
) (float64, float64, float64, error) {
	client, err := exchange.GetClient(model.BOOK, sandbox)
	if err != nil {
		return 0, dip, pip, err
	}

	ticker, err := exchange.GetTicker(client, market)
	if err != nil {
		return 0, dip, pip, err
	}

	// 24-hour statistics
	stats, err := exchange.Get24h(client, market)
	if err != nil {
		return 0, dip, pip, err
	}

	// 24-hour average
	avg, err := stats.Avg(exchange, sandbox)
	if err != nil {
		return 0, dip, pip, err
	}

	prec, err := exchange.GetPricePrec(client, market)
	if err != nil {
		return 0, dip, pip, err
	}

	return GetEx(exchange, client, market, ticker, avg, dip, pip, max, min, dist, prec, top, strict)
}

// returns (agg, dip, pip, error)
func GetEx(
	exchange model.Exchange,
	client interface{},
	market string,
	ticker float64,
	avg float64,
	dip, pip float64,
	max, min float64,
	dist, prec, top int,
	strict bool,
) (float64, float64, float64, error) {
	Max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}

	book, err := exchange.GetBook(client, market, model.BOOK_SIDE_BIDS)
	if err != nil {
		return 0, dip, pip, err
	}

	for cnt := Max(top, 4); cnt >= Max(top, 2); cnt-- {
		if out, err := get(exchange, client, market, ticker, avg, book, dip, pip, max, min, dist, prec, cnt); err == nil {
			return out, dip, pip, err
		}
	}

	if !strict {
		x := math.Round(dip)
		y := math.Round(pip)
		// if we cannot find any supports, upper your pip setting one percentage at a time until (a) we can, or (b) 50%
		for y < 50 {
			y++
			for cnt := Max(top, 4); cnt >= Max(top, 2); cnt-- {
				if out, err := get(exchange, client, market, ticker, avg, book, x, y, max, min, dist, prec, cnt); err == nil {
					return out, x, y, err
				}
			}
		}
		// if we cannot find any supports, lower your dip setting one percentage at a time until (a) we can, or (b) 0%
		for x > 0 {
			x--
			for cnt := Max(top, 4); cnt >= Max(top, 2); cnt-- {
				if out, err := get(exchange, client, market, ticker, avg, book, x, y, max, min, dist, prec, cnt); err == nil {
					return out, x, y, err
				}
			}
		}
		// if we cannot find any supports, upper your pip setting one percentage at a time until (a) we can, or (b) 100%
		for y < 100 {
			y++
			for cnt := Max(top, 4); cnt >= Max(top, 2); cnt-- {
				if out, err := get(exchange, client, market, ticker, avg, book, x, y, max, min, dist, prec, cnt); err == nil {
					return out, x, y, err
				}
			}
		}
	}

	if err != nil {
		return 0, dip, pip, err
	} else {
		return 0, dip, pip, ECannotFindSupports
	}
}

func get(
	exchange model.Exchange,
	client interface{},
	market string,
	ticker float64,
	avg float64,
	book1 interface{},
	dip, pip float64,
	max, min float64,
	dist, prec, cnt int,
) (float64, error) {
	agg := float64(5000)

	next := func(step float64) float64 {
		return precision.Round((agg * step), prec)
	}

	// the last step we can make
	last, err := strconv.ParseFloat(precision.Format(prec), 64)
	if err != nil {
		return 0, err
	}

	var steps = [...]float64{
		0.5, // 250
		0.4, // 100
		0.5, // 50
		0.5, // 25
		0.8, // 20
		0.5, // 10
		0.5, // 5
	}

	for {
		for _, step := range steps {
			if agg = next(step); agg == 0 {
				return 0, ECannotFindSupports
			}

			book2, err := exchange.Aggregate(client, book1, market, agg)
			if err != nil {
				return 0, err
			}

			// ignore supports that are more expense than ticker
			i := 0
			for i < len(book2) {
				if book2[i].Price > ticker {
					book2 = append(book2[:i], book2[i+1:]...)
				} else {
					i++
				}
			}

			// ignore supports that are cheaper than ticker minus 30%
			if min == 0 && pip < 100 {
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

			// ignore supports that are more expensive than 24h average minus 5%
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

			// ignore supports that are more expensive than max (optional)
			if max > 0 {
				i = 0
				for i < len(book2) {
					if book2[i].Price >= max {
						book2 = append(book2[:i], book2[i+1:]...)
					} else {
						i++
					}
				}
			}

			// distance between the supports must be at least 2%
			if dist > 0 && len(book2) > 1 {
				// returns true if the delta between 2 supports is lower than --dist=[percentage], otherwise false
				if func() bool {
					for i1 := 0; i1 < len(book2); i1++ {
						for i2 := 0; i2 < len(book2); i2++ {
							if i2 != i1 {
								hi, lo := func() (float64, float64) {
									if book2[i1].Price < book2[i2].Price {
										return book2[i2].Price, book2[i1].Price
									} else {
										return book2[i1].Price, book2[i2].Price
									}
								}()
								if (((hi - lo) / lo) * 100) < float64(dist) {
									return true
								}
							}
						}
					}
					return false
				}() {
					// continue with the next step (if there is any)
					if agg > last && func() bool {
						next := next(step)
						return next > 0 && next != agg
					}() {
						continue
					} else {
						return agg, ECannotFindSupports
					}
				}
			}

			// we need at least 2 supports
			if len(book2) >= cnt {
				return agg, nil
			}

			// since we're rounding to [prec] decimals, prevent this func from getting stuck in an infinite loop
			if agg <= last || func() bool {
				next := next(step)
				return next == 0 || next == agg
			}() {
				if len(book2) > 0 {
					return agg, nil
				} else {
					return 0, ECannotFindSupports
				}
			}
		}
	}
}

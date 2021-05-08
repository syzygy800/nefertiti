package exchanges

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	exchange "github.com/svanas/nefertiti/bitstamp"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/pricing"
	"github.com/svanas/nefertiti/session"
	filemutex "github.com/alexflint/go-filemutex"
	"github.com/go-errors/errors"
)

var (
	bitstampMutex *filemutex.FileMutex
)

const (
	bitstampSessionFile = "bitstamp.time"
	bitstampSessionLock = "bitstamp.lock"
)

func init() {
	exchange.BeforeRequest = func(method, path string) error {
		var err error

		if bitstampMutex == nil {
			if bitstampMutex, err = filemutex.New(session.GetSessionFile(bitstampSessionLock)); err != nil {
				return err
			}
		}

		if err = bitstampMutex.Lock(); err != nil {
			return err
		}

		var lastRequest *time.Time
		if lastRequest, err = session.GetLastRequest(bitstampSessionFile); err != nil {
			return err
		}

		if lastRequest != nil {
			elapsed := time.Since(*lastRequest)
			if elapsed.Seconds() < (float64(1) / exchange.RequestsPerSecond) {
				sleep := time.Duration((float64(time.Second) / exchange.RequestsPerSecond) - float64(elapsed))
				if flag.Debug() {
					log.Printf("[DEBUG] sleeping %f seconds\n", sleep.Seconds())
				}
				time.Sleep(sleep)
			}
		}

		if flag.Debug() {
			log.Printf("[DEBUG] %s %s\n", method, path)
		}

		return nil
	}
	exchange.AfterRequest = func() {
		defer func() {
			bitstampMutex.Unlock()
		}()
		session.SetLastRequest(bitstampSessionFile, time.Now())
	}
}

type Bitstamp struct {
	*model.ExchangeInfo
}

func (self *Bitstamp) info(msg string, level int64, service model.Notify) {
	pc, file, line, _ := runtime.Caller(1)
	log.Printf("[INFO] %s %s", errors.FormatCaller(pc, file, line), msg)
	if service != nil {
		if notify.CanSend(level, notify.INFO) {
			err := service.SendMessage(msg, "Bitstamp - INFO")
			if err != nil {
				log.Printf("[ERROR] %v", err)
			}
		}
	}
}

func (self *Bitstamp) error(err error, level int64, service model.Notify) {
	pc, file, line, _ := runtime.Caller(1)
	prefix := errors.FormatCaller(pc, file, line)

	msg := fmt.Sprintf("%s %v", prefix, err)
	_, ok := err.(*errors.Error)
	if ok {
		log.Printf("[ERROR] %s", err.(*errors.Error).ErrorStack(prefix, ""))
	} else {
		log.Printf("[ERROR] %s", msg)
	}

	if service != nil {
		if notify.CanSend(level, notify.ERROR) {
			err := service.SendMessage(msg, "Bitstamp - ERROR")
			if err != nil {
				log.Printf("[ERROR] %v", err)
			}
		}
	}
}

func (self *Bitstamp) GetInfo() *model.ExchangeInfo {
	return self.ExchangeInfo
}

func (self *Bitstamp) GetClient(private, sandbox bool) (interface{}, error) {
	if !private {
		return exchange.New("", "", ""), nil
	}

	var (
		err        error
		apiKey     string
		apiSecret  string
		customerId string
	)
	if apiKey, apiSecret, customerId, err = promptForApiKeysEx("Bitstamp"); err != nil {
		return nil, err
	}

	return exchange.New(apiKey, apiSecret, customerId), nil
}

func (self *Bitstamp) GetMarkets(cached, sandbox bool) ([]model.Market, error) {
	var out []model.Market

	markets, err := exchange.GetMarkets(exchange.New("", "", ""), cached)

	if err != nil {
		return nil, err
	}

	for _, market := range markets {
		out = append(out, model.Market{
			Name:  market.Name,
			Base:  market.Base,
			Quote: market.Quote,
		})
	}

	return out, nil
}

func (self *Bitstamp) FormatMarket(base, quote string) string {
	return strings.ToLower(base + quote)
}

// listens to the open orders, look for cancelled orders, send a notification.
func (self *Bitstamp) listen(
	client *exchange.Client,
	service model.Notify,
	level int64,
	old exchange.Orders,
	transactions exchange.Transactions,
) (exchange.Orders, error) {
	var err error

	// get my new open orders
	var new exchange.Orders
	if new, err = client.GetOpenOrders(); err != nil {
		return old, err
	}

	// look for cancelled orders
	for _, order := range old {
		if new.IndexById(order.Id) == -1 {
			// if this order has NOT been FILLED, then it has been cancelled.
			if transactions.IndexByOrderId(order.Id) == -1 {
				var data []byte
				if data, err = json.Marshal(order); err != nil {
					return new, errors.Wrap(err, 1)
				}

				log.Println("[CANCELLED] " + string(data))

				side := order.Side()
				if side != "" && service != nil {
					if notify.CanSend(level, notify.CANCELLED) {
						if err = service.SendMessage(string(data), fmt.Sprintf("Bitstamp - Done %s (Reason: Cancelled)", strings.Title(side))); err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
				}
			}
		}
	}

	// look for new orders
	for _, order := range new {
		if old.IndexById(order.Id) == -1 {
			var data []byte
			if data, err = json.Marshal(order); err != nil {
				return new, errors.Wrap(err, 1)
			}

			log.Println("[OPEN] " + string(data))

			side := order.Side()
			if side != "" && service != nil {
				if notify.CanSend(level, notify.OPENED) || (level == notify.LEVEL_DEFAULT && side == exchange.SELL) {
					if err = service.SendMessage(string(data), ("Bitstamp - Open " + strings.Title(side))); err != nil {
						log.Printf("[ERROR] %v", err)
					}
				}
			}
		}
	}

	return new, nil
}

// listens to the transaction history, look for newly filled orders, automatically place new LIMIT SELL orders.
func (self *Bitstamp) sell(
	client *exchange.Client,
	mult float64,
	hold model.Markets,
	service model.Notify,
	twitter *notify.TwitterKeys,
	level int64,
	old exchange.Transactions,
	sandbox bool,
) (exchange.Transactions, error) {
	var (
		err     error
		markets []model.Market
	)

	if markets, err = self.GetMarkets(true, sandbox); err != nil {
		return old, err
	}

	// get my new transaction history
	var new exchange.Transactions
	for _, market := range markets {
	outer:
		for {
			var txn exchange.Transactions
			if txn, err = client.GetUserTransactions(market.Name); err != nil {
				return old, err
			}
			if len(old) == 0 {
				new = append(new, txn...)
				break
			} else {
				this := txn.GetOrders()
				prev := old.GetOrdersEx(client, market.Name)
				if len(this) >= len(prev) {
					if len(prev) == 0 {
						new = append(new, txn...)
						break
					} else {
						for _, order := range prev {
							if this.IndexByOrderId(order.OrderId()) > -1 {
								new = append(new, txn...)
								break outer
							}
						}
					}
				}
				log.Printf("[WARN] user_transactions/%s returned %d orders, expected at least %d.", market.Name, len(this), len(prev))
			}
		}
	}

	if len(old) > 0 {
		prev := old.GetOrders()
		this := new.GetOrders()
		if len(this) >= len(prev) {
			if len(prev) == 0 {
				goto WeAreGood
			} else {
				for _, order := range prev {
					if this.IndexByOrderId(order.OrderId()) > -1 {
						goto WeAreGood
					}
				}
			}
		}
		goto WhatTheFuck
	WhatTheFuck:
		return old, errors.Errorf("/user_transactions returned %d orders, expected at least %d.", len(this), len(prev))
	WeAreGood:
		// nothing to see here, carry on
	}

	// make a list of newly filled orders
	var orders []exchange.Transaction
	for _, transaction := range new {
		if transaction.OrderId() != "" {
			if old.IndexByOrderId(transaction.OrderId()) == -1 {
				orders = append(orders, transaction)
			}
		}
	}

	// send notification(s)
	for _, order := range orders {
		var data []byte
		if data, err = json.Marshal(order); err != nil {
			self.error(err, level, service)
		} else {
			log.Println("[FILLED] " + string(data))
			if notify.CanSend(level, notify.FILLED) {
				var side string
				if side, err = order.Side(client); err != nil {
					self.error(err, level, service)
				} else {
					if service != nil {
						if err = service.SendMessage(string(data), fmt.Sprintf("Bitstamp - Done %s (Reason: Filled %f qty)", strings.Title(side), order.Amount(client))); err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
					if twitter != nil {
						notify.Tweet(twitter, fmt.Sprintf("Done %s. %s priced at %f #Bitstamp", strings.Title(side), model.TweetMarket(markets, order.Market(client)), order.Price(client)))
					}
				}
			}
		}
	}

	// has a buy order been filled? then place a sell order
	for i := 0; i < len(orders); i++ {
		this, _ := orders[i].Side(client)
		if this == exchange.BUY {
			qty := orders[i].Amount(client)

			// add up amount(s), hereby preventing a problem with partial matches
			n := i + 1
			for n < len(orders) {
				that, _ := orders[n].Side(client)
				if orders[n].Market(client) == orders[i].Market(client) && (that == this) && orders[n].Price(client) == orders[i].Price(client) {
					qty = qty + orders[n].Amount(client)
					orders = append(orders[:n], orders[n+1:]...)
				} else {
					n++
				}
			}

			var sp int
			if sp, err = self.GetSizePrec(client, orders[i].Market(client)); err != nil {
				return old, err
			} else {
				qty = pricing.FloorToPrecision(qty, sp)
			}

			// get base currency and desired size, calculate price, place sell order
			var (
				base  string
				quote string
			)
			base, quote, err = model.ParseMarket(markets, orders[i].Market(client))
			if err == nil {
				var pp int
				if pp, err = self.GetPricePrec(client, orders[i].Market(client)); err == nil {
					attempts := 0
					for {
						_, err = client.SellLimitOrder(
							orders[i].Market(client),
							self.GetMaxSize(client, base, quote, hold.HasMarket(orders[i].Market(client)), qty),
							pricing.Multiply(orders[i].Price(client), mult, pp),
						)
						if err != nil && strings.Contains(err.Error(), "Order could not be placed") {
							attempts++
							if attempts >= 10 {
								break
							}
						} else {
							break
						}
					}
				}
			}

			if err != nil {
				var data []byte
				if data, _ = json.Marshal(orders[i]); data == nil {
					self.error(err, level, service)
				} else {
					self.error(errors.Append(err, "\t", string(data)), level, service)
				}
			}
		}
	}

	return new, nil
}

func (self *Bitstamp) Sell(
	start time.Time,
	hold model.Markets,
	sandbox, tweet, debug bool,
	success model.OnSuccess,
) error {
	var err error

	strategy := model.GetStrategy()
	if strategy == model.STRATEGY_STANDARD || strategy == model.STRATEGY_TRAILING_STOP_LOSS {
		// we are OK
	} else {
		return errors.New("Strategy not implemented")
	}

	var (
		apiKey     string
		apiSecret  string
		customerId string
	)
	if apiKey, apiSecret, customerId, err = promptForApiKeysEx("bitstamp"); err != nil {
		return err
	}

	var service model.Notify = nil
	if service, err = notify.New().Init(flag.Interactive(), true); err != nil {
		return err
	}

	var twitter *notify.TwitterKeys = nil
	if tweet {
		if twitter, err = notify.TwitterPromptForKeys(flag.Interactive()); err != nil {
			return err
		}
	}

	client := exchange.New(apiKey, apiSecret, customerId)

	// get my open orders
	var open []exchange.Order
	if open, err = client.GetOpenOrders(); err != nil {
		return err
	}

	// get my transaction history
	var (
		markets      []exchange.Market
		transactions exchange.Transactions
	)
	if markets, err = exchange.GetMarkets(client, true); err != nil {
		return err
	}
	for _, market := range markets {
		var txn exchange.Transactions
		if txn, err = client.GetUserTransactions(market.Name); err != nil {
			return err
		}
		transactions = append(transactions, txn...)
	}

	if err = success(service); err != nil {
		return err
	}

	for {
		var (
			level int64   = notify.Level()
			mult  float64 = model.GetMult()
		)
		// listens to the transaction history, look for newly filled orders, automatically place new LIMIT SELL orders.
		if transactions, err = self.sell(client, mult, hold, service, twitter, level, transactions, sandbox); err != nil {
			self.error(err, level, service)
		} else {
			// listens to the open orders, look for cancelled orders, send a notification.
			if open, err = self.listen(client, service, level, open, transactions); err != nil {
				self.error(err, level, service)
			} else {
				// follow up on the "aggressive" strategy
				if model.GetStrategy() == model.STRATEGY_STANDARD && flag.Exists("dca") {
					// we won't be re-buying *unless* your most recent (non-sold) sell is older than 14 days
					const rebuyAfterDays = 14

					var markets []exchange.Market
					if markets, err = exchange.GetMarkets(client, true); err != nil {
						self.error(err, level, service)
					} else {
						for _, market := range markets {
							youngest := time.Time{} // January 1, year 1, 00:00:00.000000000 UTC

							for _, order := range open {
								side := model.NewOrderSide(order.Side())
								if side == model.SELL {
									if order.MarketEx() == market.Name {
										createdAt := order.GetDateTimeEx()
										if youngest.IsZero() || youngest.Before(createdAt) {
											youngest = createdAt
										}
									}
								}
							}

							if !youngest.IsZero() && time.Since(youngest).Hours() > 24*rebuyAfterDays {
								// did we recently sell an "aggressive" order on this market? then prevent us from buying this pump.
								var closed model.Orders
								if closed, err = self.GetClosed(client, market.Name); err != nil {
									self.error(err, level, service)
								} else {
									if time.Since(closed.Youngest(model.SELL, time.Now())).Hours() < 24*rebuyAfterDays {
										// continue
									} else {
										self.info(fmt.Sprintf(
											"Re-buying %s because your latest activity on this market (at %s) is older than %d days.",
											market.Name, youngest.Format(time.RFC1123), rebuyAfterDays,
										), level, service)
										var ticker float64
										if ticker, err = self.GetTicker(client, market.Name); err != nil {
											self.error(err, level, service)
										} else {
											var precSize int
											if precSize, err = self.GetSizePrec(client, market.Name); err != nil {
												self.error(err, level, service)
											} else {
												orderType := model.MARKET
												for {
													var qty float64
													if qty, err = exchange.GetMinOrderSize(client, market.Name, ticker, precSize); err != nil {
														self.error(err, level, service)
													} else {
														if hold.HasMarket(market.Name) {
															qty = qty * 5
														}
														if orderType == model.MARKET {
															_, err = client.BuyMarketOrder(market.Name, pricing.RoundToPrecision(qty, precSize))
														} else {
															var precPrice int
															if precPrice, err = self.GetPricePrec(client, market.Name); err == nil {
																ticker = ticker * 1.01
																_, err = client.BuyLimitOrder(market.Name,
																	pricing.RoundToPrecision(qty, precSize),
																	pricing.RoundToPrecision(ticker, precPrice),
																)
															}
														}
														if err != nil {
															// --- BEGIN --- svanas 2020-09-15 --- error: Minimum order size is ... -----------
															if strings.Contains(err.Error(), "Minimum order size") {
																lower, _ := strconv.ParseFloat(pricing.FormatPrecision(precSize), 64)
																ticker = ticker - lower
																continue
															}
															// --- BEGIN --- svanas 2021-03-26 --- error: Order could not be placed -----------
															if strings.Contains(err.Error(), "Order could not be placed") {
																if orderType == model.MARKET {
																	orderType = model.LIMIT
																	continue
																}
															}
															// ---- END ---- svanas 2020-09-15 ------------------------------------------------
															self.error(err, level, service)
														}
													}
													break
												}
											}
										}
									}
								}
							}
						}
					}
				}
				// follow up on the trailing stop loss strategy
				if model.GetStrategy() == model.STRATEGY_TRAILING_STOP_LOSS {
					for _, order := range open {
						side := model.NewOrderSide(order.Side())
						// enumerate over limit sell orders
						if side == model.SELL {
							var market string
							if market, err = order.Market(); err == nil {
								// do not replace the limit orders that are merely used as a reference for the HODL strategy
								if !hold.HasMarket(market) {
									var ticker float64
									if ticker, err = self.GetTicker(client, market); err == nil {
										// is the ticker nearing the order price? then cancel the limit sell order, and place a new one above the ticker.
										if ticker > (pricing.NewMult(mult, 0.75) * (order.Price / mult)) {
											var prec int
											if prec, err = self.GetPricePrec(client, market); err == nil {
												price := pricing.Multiply(ticker, pricing.NewMult(mult, 0.5), prec)
												if price > order.Price {
													self.info(
														fmt.Sprintf("Reopening %s (created at %s) because ticker is nearing limit sell price %f",
															market, order.DateTime, order.Price,
														),
														level, service)
													if err = client.CancelOrder(order.Id); err == nil {
														time.Sleep(time.Second * 5) // give Bitstamp some time to credit your wallet before we re-open this order
														_, err = client.SellLimitOrder(market, order.Amount, price)
													}
												}
											}
										} else {
											// has this limit sell order been created after we started this instance of the sell bot?
											var created *time.Time
											if created, err = order.GetDateTime(); err == nil {
												if created.Sub(start) > 0 {
													stop := (order.Price / mult) - (((mult - 1) * 0.5) * (order.Price / mult))
													// is the ticker below the stop loss price? then cancel the limit sell order, and place a market sell.
													if ticker < stop {
														self.info(
															fmt.Sprintf("Selling %s (created at %s) because ticker is below stop loss price %f",
																market, order.DateTime, stop,
															),
															level, service)
														if err = client.CancelOrder(order.Id); err == nil {
															time.Sleep(time.Second * 5) // give Bitstamp some time to credit your wallet before we re-open this order
															_, err = client.SellMarketOrder(market, order.Amount)
														}
													} else {
														self.info(
															fmt.Sprintf("Managing %s (created at %s). Currently placed at limit sell price %f",
																market, order.DateTime, order.Price,
															),
															level, nil)
													}
												}
											}
										}
									}
								}
							}
						}
						if err != nil {
							var data []byte
							if data, _ = json.Marshal(order); data == nil {
								self.error(err, level, service)
							} else {
								self.error(errors.Append(err, "\t", string(data)), level, service)
							}
						}
					}
				}
			}
		}
	}
}

func (self *Bitstamp) Order(
	client interface{},
	side model.OrderSide,
	market string,
	size float64,
	price float64,
	kind model.OrderType,
	meta string,
) (oid []byte, raw []byte, err error) {
	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return nil, nil, errors.New("invalid argument: client")
	}

	var order *exchange.Order
	if side == model.BUY {
		if order, err = bitstamp.BuyLimitOrder(market, size, price); err != nil {
			return nil, nil, err
		}
	} else if side == model.SELL {
		if order, err = bitstamp.SellLimitOrder(market, size, price); err != nil {
			return nil, nil, err
		}
	}

	var out []byte
	if out, err = json.Marshal(order); err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	return []byte(order.Id), out, nil
}

func (self *Bitstamp) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, meta string) ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (self *Bitstamp) OCO(client interface{}, side model.OrderSide, market string, size float64, price, stop float64, meta1, meta2 string) ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (self *Bitstamp) GetClosed(client interface{}, market string) (model.Orders, error) {
	var err error

	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var transactions []exchange.Transaction
	if transactions, err = bitstamp.GetUserTransactions(market); err != nil {
		return nil, err
	}

	var out model.Orders
	for _, transaction := range transactions {
		if transaction.OrderId() != "" {
			side, _ := transaction.Side(bitstamp)
			out = append(out, model.Order{
				Side:      model.NewOrderSide(side),
				Market:    transaction.Market(bitstamp),
				Size:      transaction.Amount(bitstamp),
				Price:     transaction.Price(bitstamp),
				CreatedAt: transaction.DateTime(),
			})
		}
	}

	return out, nil
}

func (self *Bitstamp) GetOpened(client interface{}, market string) (model.Orders, error) {
	var err error

	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var orders []exchange.Order
	if orders, err = bitstamp.GetOpenOrdersEx(market); err != nil {
		return nil, err
	}

	var out model.Orders
	for _, order := range orders {
		out = append(out, model.Order{
			Side:   model.NewOrderSide(order.Side()),
			Market: market,
			Size:   order.Amount,
			Price:  order.Price,
		})
	}

	return out, nil
}

func (self *Bitstamp) GetBook(client interface{}, market string, side model.BookSide) (interface{}, error) {
	var err error

	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var book *exchange.OrderBook
	if book, err = bitstamp.OrderBook(market); err != nil {
		return nil, err
	}

	var out []exchange.BookEntry
	if side == model.BOOK_SIDE_ASKS {
		out = book.Asks
	} else {
		out = book.Bids
	}

	return out, nil
}

func (self *Bitstamp) Aggregate(client, book interface{}, market string, agg float64) (model.Book, error) {
	bids, ok := book.([]exchange.BookEntry)
	if !ok {
		return nil, errors.New("invalid argument: book")
	}

	prec, err := self.GetPricePrec(client, market)
	if err != nil {
		return nil, err
	}

	var out model.Book
	for _, e := range bids {
		price := pricing.RoundToPrecision(pricing.RoundToNearest(e.Price(), agg), prec)
		entry := out.EntryByPrice(price)
		if entry != nil {
			entry.Size = entry.Size + e.Size()
		} else {
			entry = &model.BookEntry{
				Buy: &model.Buy{
					Market: market,
					Price:  price,
				},
				Size: e.Size(),
			}
			out = append(out, *entry)
		}

	}

	return out, nil
}

func (self *Bitstamp) GetTicker(client interface{}, market string) (float64, error) {
	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	ticker, err := bitstamp.Ticker(market)
	if err != nil {
		return 0, err
	}

	return ticker.Last, nil
}

func (self *Bitstamp) Get24h(client interface{}, market string) (*model.Stats, error) {
	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	ticker, err := bitstamp.Ticker(market)
	if err != nil {
		return nil, err
	}

	return &model.Stats{
		Market:    market,
		High:      ticker.High,
		Low:       ticker.Low,
		BtcVolume: 0,
	}, nil
}

func (self *Bitstamp) GetPricePrec(client interface{}, market string) (int, error) {
	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	markets, err := exchange.GetMarkets(bitstamp, true)
	if err != nil {
		return 0, err
	}
	for _, m := range markets {
		if m.Name == market {
			return m.PricePrec, nil
		}
	}
	return 8, nil
}

func (self *Bitstamp) GetSizePrec(client interface{}, market string) (int, error) {
	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	markets, err := exchange.GetMarkets(bitstamp, true)
	if err != nil {
		return 0, err
	}
	for _, m := range markets {
		if m.Name == market {
			return m.SizePrec, nil
		}
	}
	return 8, nil
}

func (self *Bitstamp) GetMaxSize(client interface{}, base, quote string, hold bool, def float64) float64 {
	market := self.FormatMarket(base, quote)

	fn := func() int {
		prec, err := self.GetSizePrec(client, market)
		if err != nil {
			return 8
		} else {
			return prec
		}
	}

	out := model.GetSizeMax(hold, def, fn)

	if hold {
		ticker, err := self.GetTicker(client, market)
		if err == nil {
			bitstamp, ok := client.(*exchange.Client)
			if ok {
				min, err := exchange.GetMinimumOrder(bitstamp, market)
				if err == nil {
					if (out * ticker) >= min {
						// we are good
					} else {
						stats, err := self.Get24h(client, market)
						if err == nil {
							qty := (min / stats.Low)
							if qty > out {
								out = pricing.RoundToPrecision(qty, fn())
							}
						}
					}
				}
			}
		}
	}

	return out
}

func (self *Bitstamp) Cancel(client interface{}, market string, side model.OrderSide) error {
	var err error

	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	var orders []exchange.Order
	if orders, err = bitstamp.GetOpenOrdersEx(market); err != nil {
		return err
	}

	for _, order := range orders {
		if ((side == model.BUY) && (order.Side() == exchange.BUY)) || ((side == model.SELL) && (order.Side() == exchange.SELL)) {
			if err = bitstamp.CancelOrder(order.Id); err != nil {
				return err
			}
		}
	}

	return nil
}

func (self *Bitstamp) Buy(client interface{}, cancel bool, market string, calls model.Calls, size, deviation float64, kind model.OrderType) error {
	var err error

	bitstamp, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	// step #1: delete the buy order(s) that are open in your book
	if cancel {
		var orders []exchange.Order
		if orders, err = bitstamp.GetOpenOrdersEx(market); err != nil {
			return err
		}
		for _, order := range orders {
			side := order.Side()
			if side == exchange.BUY {
				// do not cancel orders that we're about to re-place
				index := calls.IndexByPrice(order.Price)
				if index > -1 && order.Amount == size {
					calls[index].Skip = true
				} else {
					if err = bitstamp.CancelOrder(order.Id); err != nil {
						return err
					}
				}
			}
		}
	}

	// step 2: open the top X buy orders
	for _, call := range calls {
		if !call.Skip {
			var (
				min   float64
				qty   float64 = size
				limit float64 = call.Price
			)
			if deviation != 1.0 {
				kind, limit = call.Deviate(self, client, kind, deviation)
			}
			// --- BEGIN --- svanas 2020-01-06 --- Minimum order size is 25.0 EUR ---
			if min, err = exchange.GetMinimumOrder(bitstamp, market); err != nil {
				return err
			}
			if min > 0 {
				if limit == 0 {
					if limit, err = self.GetTicker(client, market); err != nil {
						return err
					}
				}
				if (qty * limit) < min {
					var prec int
					if prec, err = self.GetSizePrec(client, market); err != nil {
						return err
					}
					qty = pricing.CeilToPrecision((min / limit), prec)
				}
			}
			// ---- END ---- svanas 2020-01-06 --------------------------------------
			attempts := 0
			for {
				_, err = bitstamp.BuyLimitOrder(market, qty, limit)
				if err == nil {
					break
				} else {
					if strings.Contains(err.Error(), "Order could not be placed") {
						attempts++
						if attempts >= 10 {
							self.error(err, notify.LEVEL_DEFAULT, nil)
							break
						}
					} else {
						return err
					}
				}
			}
		}
	}

	return nil
}

func NewBitstamp() model.Exchange {
	return &Bitstamp{
		ExchangeInfo: &model.ExchangeInfo{
			Code: "BITS",
			Name: "Bitstamp",
			URL:  "https://www.bitstamp.net",
			REST: model.Endpoint{
				URI: exchange.Endpoint,
			},
			Country: "Luxembourg",
		},
	}
}

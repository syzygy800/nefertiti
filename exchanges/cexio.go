package exchanges

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	exchange "github.com/svanas/nefertiti/cexio"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/pricing"
	"github.com/svanas/nefertiti/session"
	filemutex "github.com/alexflint/go-filemutex"
)

var (
	cexioMutex *filemutex.FileMutex
)

const (
	cexioSessionFile = "cexio.time"
	cexioSessionLock = "cexio.lock"
)

func init() {
	exchange.BeforeRequest = func(path string) error {
		var err error

		if cexioMutex == nil {
			if cexioMutex, err = filemutex.New(session.GetSessionFile(cexioSessionLock)); err != nil {
				return err
			}
		}

		if err = cexioMutex.Lock(); err != nil {
			return err
		}

		var lastRequest *time.Time
		if lastRequest, err = session.GetLastRequest(cexioSessionFile); err != nil {
			return err
		}

		if lastRequest != nil {
			elapsed := time.Since(*lastRequest)
			if elapsed.Seconds() < (float64(1) / float64(exchange.RequestsPerSecond)) {
				sleep := time.Duration((float64(time.Second) / float64(exchange.RequestsPerSecond))) - elapsed
				if flag.Debug() {
					log.Printf("[DEBUG] sleeping %f seconds", sleep.Seconds())
				}
				time.Sleep(sleep)
			}
		}

		if flag.Debug() {
			log.Println("[DEBUG] GET " + path)
		}

		return nil
	}
	exchange.AfterRequest = func() {
		defer func() {
			cexioMutex.Unlock()
		}()
		session.SetLastRequest(cexioSessionFile, time.Now())
	}
}

type CexIo struct {
	*model.ExchangeInfo
}

func (self *CexIo) info(msg string, level int64, service model.Notify) {
	_, file, line, _ := runtime.Caller(1)
	log.Printf("[INFO] %s:%d %s", filepath.Base(file), line, msg)
	if service != nil {
		if notify.CanSend(level, notify.INFO) {
			service.SendMessage(msg, "CEX.IO - INFO")
		}
	}
}

func (self *CexIo) error(err error, level int64, service model.Notify) {
	_, file, line, _ := runtime.Caller(1)
	str := err.Error()
	log.Printf("[ERROR] %s:%d %s", filepath.Base(file), line, strings.Replace(str, "\n\n", " ", -1))
	if service != nil {
		if notify.CanSend(level, notify.ERROR) {
			service.SendMessage(str, "CEX.IO - ERROR")
		}
	}
}

func (self *CexIo) encodePair(symbol1, symbol2 string) (string, error) {
	if symbol1 == "" {
		return "", errors.New("Symbol1 is empty")
	}
	if symbol2 == "" {
		return "", errors.New("Symbol2 is empty")
	}
	return symbol1 + "-" + symbol2, nil
}

func (self *CexIo) decodePair(pair string) (symbol1, symbol2 string, err error) {
	symbols := strings.Split(pair, "-")
	if len(symbols) > 1 {
		return symbols[0], symbols[1], nil
	}
	return "", "", errors.New("Invalid market pair: " + pair)
}

func (self *CexIo) GetInfo() *model.ExchangeInfo {
	return self.ExchangeInfo
}

func (self *CexIo) GetClient(private, sandbox bool) (interface{}, error) {
	if !private {
		return exchange.New("", "", ""), nil
	}

	var (
		err       error
		apiKey    string
		apiSecret string
		userName  string
	)
	if apiKey, apiSecret, userName, err = promptForApiKeysEx("CEX.IO"); err != nil {
		return nil, err
	}

	return exchange.New(apiKey, apiSecret, userName), nil
}

func (self *CexIo) GetMarkets(cached, sandbox bool) ([]model.Market, error) {
	var (
		err error
		out []model.Market
	)

	client := exchange.New("", "", "")

	var pairs []exchange.Pair
	if pairs, err = client.CurrencyLimits(); err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		var market string
		if market, err = self.encodePair(pair.Symbol1, pair.Symbol2); err == nil {
			out = append(out, model.Market{
				Name:  market,
				Base:  pair.Symbol1,
				Quote: pair.Symbol2,
			})
		}
	}

	return out, nil
}

func (self *CexIo) FormatMarket(base, quote string) string {
	return fmt.Sprintf("%s-%s", base, quote)
}

// listens to the open orders, look for cancelled orders, send a notification.
func (self *CexIo) listen(
	client *exchange.Client,
	service model.Notify,
	level int64,
	old exchange.Orders,
) (exchange.Orders, error) {
	var err error

	// get my new open orders
	var new exchange.Orders
	if new, err = client.OpenOrdersAll(); err != nil {
		return old, err
	}

	// look for cancelled orders
	for _, order := range old {
		if new.IndexById(order.Id) == -1 {
			// get my archived orders
			var archive exchange.Orders
			if archive, err = client.ArchivedOrders(order.Symbol1, order.Symbol2); err != nil {
				return new, err
			}
			// if this order has NOT been FILLED, then it has been cancelled.
			if archive.IndexById(order.Id) == -1 {
				var data []byte
				if data, err = json.Marshal(order); err != nil {
					return new, err
				}

				log.Println("[CANCELLED] " + string(data))

				side := order.Side()
				if side != exchange.SIDE_UNKNOWN {
					if service != nil && notify.CanSend(level, notify.CANCELLED) {
						if err = service.SendMessage(string(data), fmt.Sprintf("CEX.IO - Done %s (Reason: Cancelled)", strings.Title(order.Type))); err != nil {
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
				return new, err
			}

			log.Println("[OPEN] " + string(data))

			if service != nil {
				side := order.Side()
				if side != exchange.SIDE_UNKNOWN {
					if notify.CanSend(level, notify.OPENED) || (level == notify.LEVEL_DEFAULT && side == exchange.SELL) {
						if err = service.SendMessage(string(data), ("CEX.IO - Open " + strings.Title(order.Type))); err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
				}
			}
		}
	}

	return new, nil
}

// listens to the archived orders, look for newly filled orders, automatically place new LIMIT SELL orders.
func (self *CexIo) sell(
	client *exchange.Client,
	mult float64,
	hold model.Markets,
	service model.Notify,
	twitter *notify.TwitterKeys,
	level int64,
	old exchange.Orders,
	sandbox bool,
) (exchange.Orders, error) {
	var err error

	var markets []model.Market
	if markets, err = self.GetMarkets(true, sandbox); err != nil {
		return old, err
	}

	// get my new archived orders
	var new []exchange.Order
	if new, err = client.ArchivedOrdersAll(); err != nil {
		return old, err
	}

	// look for filled orders
	for _, order := range new {
		if old.IndexById(order.Id) == -1 {
			var data []byte
			if data, err = json.Marshal(order); err != nil {
				return new, err
			}

			log.Println("[FILLED] " + string(data))

			side := order.Side()
			if side != exchange.SIDE_UNKNOWN {
				if notify.CanSend(level, notify.FILLED) {
					if service != nil {
						if err = service.SendMessage(string(data), fmt.Sprintf("CEX.IO - Done %s (Reason: Filled %f qty)", strings.Title(order.Type), order.Amount)); err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
					if twitter != nil {
						notify.Tweet(twitter, fmt.Sprintf("Done %s. $%s-%s priced at %f #CEXIO", strings.Title(order.Type), order.Symbol1, order.Symbol2, order.Price))
					}
				}
				// has a buy order been filled? then place a sell order
				if side == exchange.BUY {
					var market string
					if market, err = self.encodePair(order.Symbol1, order.Symbol2); err == nil {
						var (
							base  string
							quote string
						)
						base, quote, err = model.ParseMarket(markets, market)
						if err == nil {
							var prec int
							if prec, err = self.GetPricePrec(client, market); err == nil {
								_, err = client.PlaceOrder(
									order.Symbol1, order.Symbol2, exchange.SELL,
									self.GetMaxSize(client, base, quote, hold.HasMarket(market), order.Amount),
									pricing.Multiply(order.Price, mult, prec),
								)
							}
						}
					}
					if err != nil {
						return new, errors.New(err.Error() + "\n\n" + string(data))
					}
				}
			}
		}
	}

	return new, nil
}

func (self *CexIo) Sell(
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
		return errors.New("Strategy not implemented.")
	}

	var (
		apiKey    string
		apiSecret string
		userName  string
	)
	if apiKey, apiSecret, userName, err = promptForApiKeysEx("CEX.IO"); err != nil {
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

	client := exchange.New(apiKey, apiSecret, userName)

	// get my open orders
	var open []exchange.Order
	if open, err = client.OpenOrdersAll(); err != nil {
		return err
	}

	// get my archived orders
	var archive []exchange.Order
	if archive, err = client.ArchivedOrdersAll(); err != nil {
		return err
	}

	if err = success(service); err != nil {
		return err
	}

	for {
		// read the dynamic settings
		var (
			level int64   = notify.Level()
			mult  float64 = model.GetMult()
		)
		// listen to the archived orders, look for newly filled orders, automatically place new LIMIT SELL orders.
		if archive, err = self.sell(client, mult, hold, service, twitter, level, archive, sandbox); err != nil {
			self.error(err, level, service)
		} else {
			// listen to the open orders, look for cancelled orders, send a notification.
			if open, err = self.listen(client, service, level, open); err != nil {
				self.error(err, level, service)
			} else {
				// listens to the open orders, follow up on the trailing stop loss strategy
				if model.GetStrategy() == model.STRATEGY_TRAILING_STOP_LOSS {
					for _, order := range open {
						side := order.Side()
						// enumerate over limit sell orders
						if side == exchange.SELL {
							var market string
							if market, err = self.encodePair(order.Symbol1, order.Symbol2); err == nil {
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
													var created time.Time
													if created, err = order.GetTime(); err == nil {
														self.info(
															fmt.Sprintf("Reopening %s (created at %s) because ticker is nearing limit sell price %f",
																market, created.String(), order.Price,
															),
															level, service)
														if err = client.CancelOrder(order.Id); err == nil {
															time.Sleep(time.Second * 5) // give CEX.IO some time to credit your wallet before we re-open this order
															_, err = client.PlaceOrder(order.Symbol1, order.Symbol2, exchange.SELL, order.Amount, price)
														}
													}
												}
											}
										} else {
											// has this limit sell order been created after we started this instance of the sell bot?
											var created time.Time
											if created, err = order.GetTime(); err == nil {
												if created.Sub(start) > 0 {
													stop := (order.Price / mult) - (((mult - 1) * 0.5) * (order.Price / mult))
													// is the ticker below the stop loss price? then cancel the limit sell order, and place a market sell.
													if ticker < stop {
														self.info(
															fmt.Sprintf("Selling %s (created at %s) because ticker is below stop loss price %f",
																market, created.String(), stop,
															),
															level, service)
														if err = client.CancelOrder(order.Id); err == nil {
															time.Sleep(time.Second * 5) // give CEX.IO some time to credit your wallet before we re-open this order
															_, err = client.PlaceMarketOrder(
																order.Symbol1, order.Symbol2, exchange.SELL, order.Amount,
															)
														}
													} else {
														self.info(
															fmt.Sprintf("Managing %s (created at %s). Currently placed at limit sell price %f",
																market, created.String(), order.Price,
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
							msg := err.Error()
							var data []byte
							if data, err = json.Marshal(order); err == nil {
								msg = msg + "\n\n" + string(data)
							}
							self.error(errors.New(msg), level, service)
						}
					}
				}
			}
		}
	}
}

func (self *CexIo) Order(
	client interface{},
	side model.OrderSide,
	market string,
	size float64,
	price float64,
	kind model.OrderType,
	meta string,
) (oid []byte, raw []byte, err error) {
	cexio, ok := client.(*exchange.Client)
	if !ok {
		return nil, nil, errors.New("invalid argument: client")
	}

	var symbol1 string
	var symbol2 string
	if symbol1, symbol2, err = self.decodePair(market); err != nil {
		return nil, nil, err
	}

	var order *exchange.Order
	if side == model.BUY {
		if order, err = cexio.PlaceOrder(symbol1, symbol2, exchange.BUY, size, price); err != nil {
			return nil, nil, err
		}
	} else if side == model.SELL {
		if order, err = cexio.PlaceOrder(symbol1, symbol2, exchange.SELL, size, price); err != nil {
			return nil, nil, err
		}
	}

	var out []byte
	if out, err = json.Marshal(order); err != nil {
		return nil, nil, err
	}

	return []byte(order.Id), out, nil
}

func (self *CexIo) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, meta string) ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (self *CexIo) OCO(client interface{}, side model.OrderSide, market string, size float64, price, stop float64, meta1, meta2 string) ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (self *CexIo) GetClosed(client interface{}, market string) (model.Orders, error) {
	var err error

	cexio, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var symbol1 string
	var symbol2 string
	if symbol1, symbol2, err = self.decodePair(market); err != nil {
		return nil, err
	}

	var orders []exchange.Order
	if orders, err = cexio.ArchivedOrders(symbol1, symbol2); err != nil {
		return nil, err
	}

	var out model.Orders
	for _, order := range orders {
		out = append(out, model.Order{
			Side:   model.NewOrderSide(order.Type),
			Market: market,
			Size:   order.Amount,
			Price:  order.Price,
		})
	}

	return out, nil
}

func (self *CexIo) GetOpened(client interface{}, market string) (model.Orders, error) {
	var err error

	cexio, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var symbol1 string
	var symbol2 string
	if symbol1, symbol2, err = self.decodePair(market); err != nil {
		return nil, err
	}

	var orders []exchange.Order
	if orders, err = cexio.OpenOrders(symbol1, symbol2); err != nil {
		return nil, err
	}

	var out model.Orders
	for _, order := range orders {
		out = append(out, model.Order{
			Side:   model.NewOrderSide(order.Type),
			Market: market,
			Size:   order.Amount,
			Price:  order.Price,
		})
	}

	return out, nil
}

func (self *CexIo) GetBook(client interface{}, market string, side model.BookSide) (interface{}, error) {
	var err error

	cexio, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var symbol1 string
	var symbol2 string
	if symbol1, symbol2, err = self.decodePair(market); err != nil {
		return nil, err
	}

	var book *exchange.OrderBook
	if book, err = cexio.OrderBook(symbol1, symbol2); err != nil {
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

func (self *CexIo) Aggregate(client, book interface{}, market string, agg float64) (model.Book, error) {
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

func (self *CexIo) GetTicker(client interface{}, market string) (float64, error) {
	cexio, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	symbol1, symbol2, err := self.decodePair(market)
	if err != nil {
		return 0, err
	}

	ticker, err := cexio.Ticker(symbol1, symbol2)
	if err != nil {
		return 0, err
	}

	return ticker.Last, nil
}

func (self *CexIo) Get24h(client interface{}, market string) (*model.Stats, error) {
	cexio, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	symbol1, symbol2, err := self.decodePair(market)
	if err != nil {
		return nil, err
	}

	ticker, err := cexio.Ticker(symbol1, symbol2)
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

// see: https://blog.cex.io/news/engine-updates-16956
func (self *CexIo) GetPricePrec(client interface{}, market string) (int, error) {
	symbol1, symbol2, err := self.decodePair(market)
	if err != nil {
		return 0, err
	}
	if symbol1 == model.BTC {
		if model.Fiat(symbol2) {
			return 1, nil
		}
	}
	if model.Fiat(symbol2) {
		if symbol1 == model.XRP {
			return 4, nil
		}
		return 2, nil
	}
	if symbol1 == model.XRP && symbol2 == model.BTC {
		return 8, nil
	}
	return 6, nil
}

func (self *CexIo) GetSizePrec(client interface{}, market string) (int, error) {
	return 8, nil
}

func (self *CexIo) GetMaxSize(client interface{}, base, quote string, hold bool, def float64) float64 {
	fn := func() int {
		prec, err := self.GetSizePrec(client, self.FormatMarket(base, quote))
		if err != nil {
			return 8
		} else {
			return prec
		}
	}
	return model.GetSizeMax(hold, def, fn)
}

func (self *CexIo) Cancel(client interface{}, market string, side model.OrderSide) error {
	var err error

	cexio, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	var symbol1 string
	var symbol2 string
	if symbol1, symbol2, err = self.decodePair(market); err != nil {
		return err
	}

	var orders []exchange.Order
	if orders, err = cexio.OpenOrders(symbol1, symbol2); err != nil {
		return err
	}

	for _, order := range orders {
		if ((side == model.BUY) && (order.Side() == exchange.BUY)) || ((side == model.SELL) && (order.Side() == exchange.SELL)) {
			if err = cexio.CancelOrder(order.Id); err != nil {
				return err
			}
		}
	}

	return nil
}

func (self *CexIo) Buy(client interface{}, cancel bool, market string, calls model.Calls, size, deviation float64, kind model.OrderType) error {
	var err error

	cexio, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	var symbol1 string
	var symbol2 string
	if symbol1, symbol2, err = self.decodePair(market); err != nil {
		return err
	}

	// step #1: delete the buy order(s) that are open in your book
	if cancel {
		var orders []exchange.Order
		if orders, err = cexio.OpenOrders(symbol1, symbol2); err != nil {
			return err
		}
		for _, order := range orders {
			side := order.Side()
			if side == exchange.BUY {
				// do not cancel orders that we're about to re-place
				index := calls.IndexByPrice(order.Price)
				if index > -1 {
					calls[index].Skip = true
				} else {
					if err = cexio.CancelOrder(order.Id); err != nil {
						return err
					}
				}
			}
		}
	}

	// step 2: open the top X buy orders
	for _, call := range calls {
		if !call.Skip {
			limit := call.Price
			if deviation != 1.0 {
				kind, limit = call.Deviate(self, client, kind, deviation)
			}
			if _, err = cexio.PlaceOrder(symbol1, symbol2, exchange.BUY, size, limit); err != nil {
				return err
			}
		}
	}

	return nil
}

func NewCexIo() model.Exchange {
	return &CexIo{
		ExchangeInfo: &model.ExchangeInfo{
			Code: "CXIO",
			Name: "CEX.IO",
			URL:  "https://cex.io",
			REST: model.Endpoint{
				URI: exchange.Endpoint,
			},
			Country: "United Kingdom",
		},
	}
}

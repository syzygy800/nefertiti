package exchanges

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/pricing"
	"github.com/svanas/nefertiti/session"
	filemutex "github.com/alexflint/go-filemutex"
	"github.com/go-errors/errors"
	exchange "github.com/svanas/go-crypto-dot-com"
)

var (
	cryptoDotComMutex *filemutex.FileMutex
)

const (
	cryptoDotComSessionFile = "crypto.com.time"
	cryptoDotComSessionLock = "crypto.com.lock"
	cryptoDotComSessionInfo = "crypto.com.json"
)

type CryptoDotComSessionInfo struct {
	Cooldown bool `json:"cooldown"`
}

func cryptoDotComRequestsPerSecond() (float64, error) {
	var (
		err  error
		data []byte
		info CryptoDotComSessionInfo
	)
	if data, err = ioutil.ReadFile(session.GetSessionFile(cryptoDotComSessionInfo)); err == nil {
		if err = json.Unmarshal(data, &info); err == nil {
			if info.Cooldown {
				info.Cooldown = false
				if data, err = json.Marshal(info); err == nil {
					err = ioutil.WriteFile(session.GetSessionFile(cryptoDotComSessionInfo), data, 0600)
				}
				return exchange.RequestsPerSecond[exchange.RATE_LIMIT_COOL_DOWN], err
			}
		}
	}
	return exchange.RequestsPerSecond[exchange.RATE_LIMIT_NORMAL], nil
}

func init() {
	exchange.BeforeRequest = func(method, path string, params *url.Values) error {
		var (
			err error
			rps float64 = exchange.RequestsPerSecond[exchange.RATE_LIMIT_NORMAL]
		)

		if cryptoDotComMutex == nil {
			if cryptoDotComMutex, err = filemutex.New(session.GetSessionFile(cryptoDotComSessionLock)); err != nil {
				return err
			}
		}

		if err = cryptoDotComMutex.Lock(); err != nil {
			return err
		}

		var lastRequest *time.Time
		if lastRequest, err = session.GetLastRequest(cryptoDotComSessionFile); err != nil {
			return err
		}

		if lastRequest != nil {
			elapsed := time.Since(*lastRequest)
			if rps, err = cryptoDotComRequestsPerSecond(); err != nil {
				return err
			}
			if elapsed.Seconds() < (float64(1) / rps) {
				sleep := time.Duration((float64(time.Second) / rps) - float64(elapsed))
				if flag.Debug() {
					log.Printf("[DEBUG] sleeping %f seconds\n", sleep.Seconds())
				}
				time.Sleep(sleep)
			}
		}

		if flag.Debug() {
			if params == nil {
				log.Printf("[DEBUG] %s %s\n", method, path)
			} else {
				log.Printf("[DEBUG] %s %s %#v\n", method, path, params)
			}
		}

		return nil
	}
	exchange.AfterRequest = func() {
		defer func() {
			cryptoDotComMutex.Unlock()
		}()
		session.SetLastRequest(cryptoDotComSessionFile, time.Now())
	}
	exchange.OnRateLimitError = func(method, path string) error {
		var (
			err  error
			data []byte
			info CryptoDotComSessionInfo
		)
		info.Cooldown = true
		if data, err = json.Marshal(info); err == nil {
			err = ioutil.WriteFile(session.GetSessionFile(cryptoDotComSessionInfo), data, 0600)
		}
		return err
	}
}

type CryptoDotCom struct {
	*model.ExchangeInfo
	symbols []exchange.Symbol
}

//-------------------- private -------------------

func (self *CryptoDotCom) error(err error, level int64, service model.Notify) {
	pc, file, line, _ := runtime.Caller(1)
	prefix := errors.FormatCaller(pc, file, line)

	msg := fmt.Sprintf("%s %v", prefix, err)
	_, ok := err.(*errors.Error)
	if ok && flag.Debug() {
		log.Printf("[ERROR] %s", err.(*errors.Error).ErrorStack(prefix, ""))
	} else {
		log.Printf("[ERROR] %s", msg)
	}

	if service != nil {
		if notify.CanSend(level, notify.ERROR) {
			err := service.SendMessage(msg, "crypto.com - ERROR")
			if err != nil {
				log.Printf("[ERROR] %v", err)
			}
		}
	}
}

func (self *CryptoDotCom) getSymbols(client *exchange.Client, quotes []string, cached bool) ([]exchange.Symbol, error) {
	if len(self.symbols) == 0 || cached == false {
		var err error
		if self.symbols, err = client.Symbols(); err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	if len(quotes) == 0 {
		return self.symbols, nil
	}

	var filtered []exchange.Symbol
	for _, symbol := range self.symbols {
		for _, quote := range quotes {
			if strings.EqualFold(symbol.CountCoin, quote) {
				filtered = append(filtered, symbol)
			}
		}
	}

	return filtered, nil
}

func (self *CryptoDotCom) parseSymbol(symbols []exchange.Symbol, symbol string) (base, quote string, err error) {
	for _, market := range symbols {
		if market.Symbol == symbol {
			return market.BaseCoin, market.CountCoin, nil
		}
	}
	return "", "", errors.Errorf("symbol %s does not exist", symbol)
}

func (self *CryptoDotCom) getOrderSide(side exchange.OrderSide) model.OrderSide {
	switch side {
	case exchange.BUY:
		return model.BUY
	case exchange.SELL:
		return model.SELL
	default:
		return model.ORDER_SIDE_NONE
	}
}

func (self *CryptoDotCom) getOrderType(order *exchange.Order) model.OrderType {
	switch order.GetType() {
	case exchange.LIMIT:
		return model.LIMIT
	case exchange.MARKET:
		return model.MARKET
	default:
		return model.ORDER_TYPE_NONE
	}
}

func (self *CryptoDotCom) indexByOrderId(orders []exchange.Order, id int64) int {
	for i, o := range orders {
		if o.Id == id {
			return i
		}
	}
	return -1
}

func (self *CryptoDotCom) indexByTradeId(trades []exchange.Trade, id int64) int {
	for i, t := range trades {
		if t.Id == id {
			return i
		}
	}
	return -1
}

//-------------------- public --------------------

func (self *CryptoDotCom) GetInfo() *model.ExchangeInfo {
	return self.ExchangeInfo
}

func (self *CryptoDotCom) GetClient(permission model.Permission, sandbox bool) (interface{}, error) {
	if permission != model.PRIVATE {
		return exchange.New("", ""), nil
	}

	var (
		err       error
		apiKey    string
		apiSecret string
	)
	if apiKey, apiSecret, err = promptForApiKeys("crypto.com"); err != nil {
		return nil, err
	}

	return exchange.New(apiKey, apiSecret), nil
}

func (self *CryptoDotCom) GetMarkets(cached, sandbox bool) ([]model.Market, error) {
	var out []model.Market

	symbols, err := self.getSymbols(exchange.New("", ""), nil, cached)
	if err != nil {
		return nil, err
	}

	for _, symbol := range symbols {
		out = append(out, model.Market{
			Name:  symbol.Symbol,
			Base:  symbol.BaseCoin,
			Quote: symbol.CountCoin,
		})
	}

	return out, nil
}

func (self *CryptoDotCom) FormatMarket(base, quote string) string {
	return strings.ToLower(base + quote)
}

// listen to the opened orders, look for cancelled orders, send a notification.
func (self *CryptoDotCom) listen(
	client *exchange.Client,
	quotes []string,
	service model.Notify,
	level int64,
	old []exchange.Order,
	filled []exchange.Trade,
) ([]exchange.Order, error) {
	var (
		err     error
		symbols []exchange.Symbol
	)

	if symbols, err = self.getSymbols(client, quotes, true); err != nil {
		return old, err
	}

	// get my opened orders
	var new []exchange.Order
	for _, market := range symbols {
		var orders []exchange.Order
		if orders, err = client.OpenOrders(market.Symbol); err != nil {
			return old, errors.Wrap(err, 1)
		}
		for _, order := range orders {
			new = append(new, order)
		}
	}

	// look for cancelled orders
	for _, order := range old {
		if self.indexByOrderId(new, order.Id) == -1 {
			// if this order has NOT been FILLED, then it has been cancelled.
			if self.indexByTradeId(filled, order.Id) == -1 {
				var data []byte
				if data, err = json.Marshal(order); err != nil {
					return new, errors.Wrap(err, 1)
				}

				log.Println("[CANCELLED] " + string(data))

				if service != nil {
					if notify.CanSend(level, notify.CANCELLED) {
						if err = service.SendMessage(string(data), fmt.Sprintf("crypto.com - Done %s (Reason: Cancelled)", order.SideMsg)); err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
				}
			}
		}
	}

	// look for newly opened orders
	for _, order := range new {
		if self.indexByOrderId(old, order.Id) == -1 {
			var data []byte
			if data, err = json.Marshal(order); err != nil {
				return new, errors.Wrap(err, 1)
			}

			log.Println("[OPEN] " + string(data))

			if service != nil {
				if notify.CanSend(level, notify.OPENED) || (level == notify.LEVEL_DEFAULT && order.GetSide() == exchange.SELL) {
					if err = service.SendMessage(string(data), ("crypto.com - Open " + order.SideMsg)); err != nil {
						log.Printf("[ERROR] %v", err)
					}
				}
			}
		}
	}

	return new, nil
}

// listen to the filled orders, look for newly filled orders, automatically place new LIMIT SELL orders.
func (self *CryptoDotCom) sell(
	client *exchange.Client,
	quotes []string,
	mult float64,
	hold model.Markets,
	service model.Notify,
	level int64,
	old []exchange.Trade,
) ([]exchange.Trade, error) {
	var (
		err     error
		symbols []exchange.Symbol
	)

	if symbols, err = self.getSymbols(client, quotes, true); err != nil {
		return old, err
	}

	// get my filled orders
	var filled []exchange.Trade
	for _, market := range symbols {
		var trades []exchange.Trade
		if trades, err = client.MyTrades(market.Symbol); err != nil {
			return old, errors.Wrap(err, 1)
		}
		for _, trade := range trades {
			filled = append(filled, trade)
		}
	}

	// make a list of newly filled orders
	var new []exchange.Trade
	for _, trade := range filled {
		if self.indexByTradeId(old, trade.Id) == -1 {
			new = append(new, trade)
		}
	}

	// send notification(s)
	for _, trade := range new {
		var data []byte
		if data, err = json.Marshal(trade); err != nil {
			self.error(err, level, service)
		} else {
			log.Println("[FILLED] " + string(data))
			if notify.CanSend(level, notify.FILLED) && service != nil {
				if err = service.SendMessage(string(data), fmt.Sprintf("crypto.com - Done %s (Reason: Filled)", trade.Type)); err != nil {
					log.Printf("[ERROR] %v", err)
				}
			}
		}
	}

	// has a buy order been filled? then place a sell order
	for i := 0; i < len(new); i++ {
		if new[i].GetSide() == exchange.BUY {
			qty := new[i].Volume

			// add up amount(s), hereby preventing a problem with partial matches
			n := i + 1
			for n < len(new) {
				if new[n].Symbol == new[i].Symbol && new[n].Side == new[i].Side && new[n].Price == new[i].Price {
					qty = qty + new[n].Volume
					new = append(new[:n], new[n+1:]...)
				} else {
					n++
				}
			}

			if qty > new[i].Volume {
				var prec int
				if prec, err = self.GetSizePrec(client, new[i].Symbol); err == nil {
					qty = pricing.FloorToPrecision(qty, prec)
				}
			}

			// get base currency and desired size, calculate price, place sell order
			var (
				base  string
				quote string
			)
			base, quote, err = self.parseSymbol(symbols, new[i].Symbol)
			if err == nil {
				var prec int
				if prec, err = self.GetPricePrec(client, new[i].Symbol); err == nil {
					_, err = client.CreateOrder(
						new[i].Symbol,
						exchange.SELL,
						exchange.LIMIT,
						self.GetMaxSize(client, base, quote, hold.HasMarket(new[i].Symbol), qty),
						pricing.Multiply(new[i].Price, mult, prec),
					)
				}
			}

			if err != nil {
				var data []byte
				if data, _ = json.Marshal(new[i]); data == nil {
					self.error(err, level, service)
				} else {
					self.error(errors.Append(err, "\t", string(data)), level, service)
				}
			}
		}
	}

	return filled, nil
}

func (self *CryptoDotCom) Sell(
	start time.Time,
	hold model.Markets,
	sandbox, tweet, debug bool,
	success model.OnSuccess,
) error {
	var err error

	strategy := model.GetStrategy()
	if strategy != model.STRATEGY_STANDARD {
		return errors.New("Strategy not implemented")
	}

	var (
		apiKey    string
		apiSecret string
	)
	if apiKey, apiSecret, err = promptForApiKeys("crypto.com"); err != nil {
		return err
	}

	var service model.Notify = nil
	if service, err = notify.New().Init(flag.Interactive(), true); err != nil {
		return err
	}

	client := exchange.New(apiKey, apiSecret)

	var (
		quotes  []string = []string{model.BTC}
		symbols []exchange.Symbol
		filled  []exchange.Trade
		opened  []exchange.Order
	)

	flg := flag.Get("quote")
	if flg.Exists {
		quotes = flg.Split(",")
	} else {
		flag.Set("quote", strings.Join(quotes, ","))
	}

	if symbols, err = self.getSymbols(client, quotes, true); err != nil {
		return err
	}

	// get my filled orders
	for _, market := range symbols {
		var trades []exchange.Trade
		if trades, err = client.MyTrades(market.Symbol); err != nil {
			return errors.Wrap(err, 1)
		}
		for _, trade := range trades {
			filled = append(filled, trade)
		}
	}

	// get my opened orders
	for _, market := range symbols {
		var orders []exchange.Order
		if orders, err = client.OpenOrders(market.Symbol); err != nil {
			return errors.Wrap(err, 1)
		}
		for _, order := range orders {
			opened = append(opened, order)
		}
	}

	if err = success(service); err != nil {
		return err
	}

	for {
		var (
			level int64   = notify.Level()
			mult  float64 = model.GetMult()
		)
		// listen to the filled orders, look for newly filled orders, automatically place new LIMIT SELL orders.
		if filled, err = self.sell(client, quotes, mult, hold, service, level, filled); err != nil {
			self.error(err, level, service)
		} else {
			// listen to the opened orders, look for cancelled orders, send a notification.
			if opened, err = self.listen(client, quotes, service, level, opened, filled); err != nil {
				self.error(err, level, service)
			}
		}
	}
}

func (self *CryptoDotCom) Order(
	client interface{},
	side model.OrderSide,
	market string,
	size float64,
	price float64,
	kind model.OrderType,
	meta string,
) (oid []byte, raw []byte, err error) {
	var out int64

	crypto, ok := client.(*exchange.Client)
	if !ok {
		return nil, nil, errors.New("invalid argument: client")
	}

	if side == model.BUY {
		if kind == model.MARKET {
			out, err = crypto.CreateOrder(market, exchange.BUY, exchange.MARKET, size, 0)
		} else {
			out, err = crypto.CreateOrder(market, exchange.BUY, exchange.LIMIT, size, price)
		}
	} else if side == model.SELL {
		if kind == model.MARKET {
			out, err = crypto.CreateOrder(market, exchange.SELL, exchange.MARKET, size, 0)
		} else {
			out, err = crypto.CreateOrder(market, exchange.SELL, exchange.LIMIT, size, price)
		}
	}

	if err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	return []byte(strconv.FormatInt(out, 10)), nil, nil
}

func (self *CryptoDotCom) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, meta string) ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (self *CryptoDotCom) OCO(client interface{}, market string, size float64, price, stop float64, meta1, meta2 string) ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (self *CryptoDotCom) GetClosed(client interface{}, market string) (model.Orders, error) {
	var err error

	crypto, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var trades []exchange.Trade
	if trades, err = crypto.MyTrades(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var out model.Orders
	for _, trade := range trades {
		out = append(out, model.Order{
			Side:      self.getOrderSide(trade.GetSide()),
			Market:    market,
			Size:      trade.Volume,
			Price:     trade.Price,
			CreatedAt: trade.GetCreatedAt(),
		})
	}

	return out, nil
}

func (self *CryptoDotCom) GetOpened(client interface{}, market string) (model.Orders, error) {
	var err error

	crypto, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var orders []exchange.Order
	if orders, err = crypto.OpenOrders(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var out model.Orders
	for _, order := range orders {
		out = append(out, model.Order{
			Side:      self.getOrderSide(order.GetSide()),
			Market:    market,
			Size:      order.Volume,
			Price:     order.Price,
			CreatedAt: order.GetCreatedAt(),
		})
	}

	return out, nil
}

func (self *CryptoDotCom) GetBook(client interface{}, market string, side model.BookSide) (interface{}, error) {
	var err error

	crypto, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var book *exchange.OrderBook
	if book, err = crypto.OrderBook(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var out []exchange.BookEntry
	if side == model.BOOK_SIDE_ASKS {
		out = book.Tick.Asks
	} else {
		out = book.Tick.Bids
	}

	return out, nil
}

func (self *CryptoDotCom) Aggregate(client, book interface{}, market string, agg float64) (model.Book, error) {
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

func (self *CryptoDotCom) GetTicker(client interface{}, market string) (float64, error) {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	ticker, err := crypto.Ticker(market)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}

	return ticker.Last, nil
}

func (self *CryptoDotCom) Get24h(client interface{}, market string) (*model.Stats, error) {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	ticker, err := crypto.Ticker(market)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return &model.Stats{
		Market:    market,
		High:      ticker.High,
		Low:       ticker.Low,
		BtcVolume: 0,
	}, nil
}

func (self *CryptoDotCom) GetPricePrec(client interface{}, market string) (int, error) {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	symbols, err := self.getSymbols(crypto, nil, true)
	if err != nil {
		return 0, err
	}
	for _, symbol := range symbols {
		if symbol.Symbol == market {
			return symbol.PricePrecision, nil
		}
	}
	return 8, nil
}

func (self *CryptoDotCom) GetSizePrec(client interface{}, market string) (int, error) {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	symbols, err := self.getSymbols(crypto, nil, true)
	if err != nil {
		return 0, err
	}
	for _, symbol := range symbols {
		if symbol.Symbol == market {
			return symbol.AmountPrecision, nil
		}
	}
	return 0, nil
}

func (self *CryptoDotCom) GetMaxSize(client interface{}, base, quote string, hold bool, def float64) float64 {
	fn := func() int {
		prec, err := self.GetSizePrec(client, self.FormatMarket(base, quote))
		if err != nil {
			return 0
		} else {
			return prec
		}
	}
	return model.GetSizeMax(hold, def, fn)
}

func (self *CryptoDotCom) Cancel(client interface{}, market string, side model.OrderSide) error {
	var err error

	crypto, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	var orders []exchange.Order
	if orders, err = crypto.OpenOrders(market); err != nil {
		return errors.Wrap(err, 1)
	}

	for _, order := range orders {
		if ((side == model.BUY) && (order.GetSide() == exchange.BUY)) || ((side == model.SELL) && (order.GetSide() == exchange.SELL)) {
			if err = crypto.CancelOrder(market, order.Id); err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func (self *CryptoDotCom) Buy(client interface{}, cancel bool, market string, calls model.Calls, size, deviation float64, kind model.OrderType) error {
	var err error

	crypto, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	// step #1: delete the buy order(s) that are open in your book
	if cancel {
		var orders []exchange.Order
		if orders, err = crypto.OpenOrders(market); err != nil {
			return errors.Wrap(err, 1)
		}
		for _, order := range orders {
			side := order.GetSide()
			if side == exchange.BUY {
				// do not cancel orders that we're about to re-place
				index := calls.IndexByPrice(order.Price)
				if index > -1 && order.Volume == size {
					calls[index].Skip = true
				} else {
					if err = crypto.CancelOrder(market, order.Id); err != nil {
						return errors.Wrap(err, 1)
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
			if kind == model.MARKET {
				_, err = crypto.CreateOrder(market, exchange.BUY, exchange.MARKET, size, 0)
			} else {
				_, err = crypto.CreateOrder(market, exchange.BUY, exchange.LIMIT, size, limit)
			}
			if err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func (self *CryptoDotCom) IsLeveragedToken(name string) bool {
	return false
}

func NewCryptoDotCom() model.Exchange {
	return &CryptoDotCom{
		ExchangeInfo: &model.ExchangeInfo{
			Code: "CRO",
			Name: "crypto.com",
			URL:  "https://crypto.com",
			REST: model.Endpoint{
				URI: "https://api.crypto.com",
			},
			Country: "Singapore",
		},
	}
}

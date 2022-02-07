//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package exchanges

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	filemutex "github.com/alexflint/go-filemutex"
	exchange "github.com/svanas/go-crypto-dot-com"
	"github.com/svanas/nefertiti/aggregation"
	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/logger"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/multiplier"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/precision"
	"github.com/svanas/nefertiti/pricing"
	"github.com/svanas/nefertiti/session"
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

func cryptoDotComRequestsPerSecond(def float64) (float64, error) {
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
	if def == 0 {
		return exchange.RequestsPerSecond[exchange.RATE_LIMIT_NORMAL], nil
	} else {
		return def, nil
	}
}

func init() {
	exchange.BeforeRequest = func(method, path string, params *url.Values, rps float64) error {
		var err error

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
			if rps, err = cryptoDotComRequestsPerSecond(rps); err != nil {
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
				log.Printf("[DEBUG] %s %s?%s\n", method, path, params.Encode())
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

func (self *CryptoDotCom) getSymbol(client *exchange.Client, name string) (*exchange.Symbol, error) {
	cached := true
	for {
		symbols, err := self.getSymbols(client, nil, cached)
		if err != nil {
			return nil, err
		}
		for _, symbol := range symbols {
			if symbol.Symbol == name {
				return &symbol, nil
			}
		}
		if !cached {
			return nil, errors.Errorf("symbol %s does not exist", name)
		}
		cached = false
	}
}

func (self *CryptoDotCom) getSymbols(client *exchange.Client, quotes []string, cached bool) ([]exchange.Symbol, error) {
	if len(self.symbols) == 0 || !cached {
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
			if strings.EqualFold(symbol.QuoteCurrency, quote) {
				filtered = append(filtered, symbol)
			}
		}
	}

	return filtered, nil
}

func (self *CryptoDotCom) parseSymbol(symbols []exchange.Symbol, symbol string) (base, quote string, err error) {
	for _, market := range symbols {
		if market.Symbol == symbol {
			return market.BaseCurrency, market.QuoteCurrency, nil
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

func (self *CryptoDotCom) GetMarkets(cached, sandbox bool, blacklist []string) ([]model.Market, error) {
	var out []model.Market

	symbols, err := self.getSymbols(exchange.New("", ""), nil, cached)
	if err != nil {
		return nil, err
	}

	for _, symbol := range symbols {
		if func() bool {
			for _, ignore := range blacklist {
				if strings.EqualFold(symbol.Symbol, ignore) {
					return false
				}
			}
			return true
		}() {
			out = append(out, model.Market{
				Name:  symbol.Symbol,
				Base:  symbol.BaseCurrency,
				Quote: symbol.QuoteCurrency,
			})
		}
	}

	return out, nil
}

func (self *CryptoDotCom) FormatMarket(base, quote string) string {
	return strings.ToLower(base + quote)
}

// listen to the opened orders, look for cancelled orders, send a notification.
func (self *CryptoDotCom) listen(
	client *exchange.Client,
	symbols []exchange.Symbol,
	service model.Notify,
	level int64,
	old []exchange.Order,
	filled []exchange.Trade,
) ([]exchange.Order, error) {
	var err error

	// get my opened orders
	var new []exchange.Order
	for _, market := range symbols {
		var orders []exchange.Order
		if orders, err = client.OpenOrders(market.Symbol); err != nil {
			return old, errors.Wrap(err, 1)
		}
		new = append(new, orders...)
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
						if err = service.SendMessage(order, fmt.Sprintf("crypto.com - Done %s (Reason: Cancelled)", order.SideMsg), model.ALWAYS); err != nil {
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
			// if this order has been filled, then it cannot possibly be new.
			if self.indexByTradeId(filled, order.Id) == -1 {
				var data []byte
				if data, err = json.Marshal(order); err != nil {
					return new, errors.Wrap(err, 1)
				}

				log.Println("[OPEN] " + string(data))

				if service != nil {
					if notify.CanSend(level, notify.OPENED) || (level == notify.LEVEL_DEFAULT && order.GetSide() == exchange.SELL) {
						if err = service.SendMessage(order, ("crypto.com - Open " + order.SideMsg), model.ALWAYS); err != nil {
							log.Printf("[ERROR] %v", err)
						}
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
	symbols []exchange.Symbol,
	mult multiplier.Mult,
	hold, earn model.Markets,
	service model.Notify,
	level int64,
	old []exchange.Trade,
) ([]exchange.Trade, error) {
	var err error

	// get my filled orders
	var filled []exchange.Trade
	for _, market := range symbols {
		var trades []exchange.Trade
		if trades, err = client.MyTrades(market.Symbol); err != nil {
			return old, errors.Wrap(err, 1)
		}
		filled = append(filled, trades...)
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
			logger.Error(self.Name, err, level, service)
		} else {
			log.Println("[FILLED] " + string(data))
			if notify.CanSend(level, notify.FILLED) && service != nil {
				if err = service.SendMessage(trade, fmt.Sprintf("crypto.com - Done %s (Reason: Filled)", trade.Type), model.ALWAYS); err != nil {
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
					qty = precision.Floor(qty, prec)
				}
			}

			// get base currency and desired size, calculate price, place sell order
			var (
				base  string
				quote string
			)
			base, quote, err = self.parseSymbol(symbols, new[i].Symbol)
			if err == nil {
				qty = self.GetMaxSize(client, base, quote, hold.HasMarket(new[i].Symbol), earn.HasMarket(new[i].Symbol), qty, mult)
				if qty > 0 {
					var prec int
					if prec, err = self.GetPricePrec(client, new[i].Symbol); err == nil {
						_, err = client.CreateOrder(
							new[i].Symbol,
							exchange.SELL,
							exchange.LIMIT,
							qty,
							pricing.Multiply(new[i].Price, mult, prec),
						)
					}
				}
			}

			if err != nil {
				logger.Error(self.Name, errors.Append(errors.Wrap(err, 1), new[i]), level, service)
			}
		}
	}

	return filled, nil
}

func (self *CryptoDotCom) Sell(
	strategy model.Strategy,
	hold, earn model.Markets,
	sandbox, tweet, debug bool,
	success model.OnSuccess,
) error {
	if strategy != model.STRATEGY_STANDARD {
		return errors.New("strategy not implemented")
	}

	var (
		err       error
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
		quotes = flg.Split()
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
		filled = append(filled, trades...)
	}

	// get my opened orders
	for _, market := range symbols {
		var orders []exchange.Order
		if orders, err = client.OpenOrders(market.Symbol); err != nil {
			return errors.Wrap(err, 1)
		}
		opened = append(opened, orders...)
	}

	if err = success(service); err != nil {
		return err
	}

	for {
		// read the dynamic settings
		var (
			level int64 = notify.LEVEL_DEFAULT
			mult  multiplier.Mult
		)
		if level, err = notify.Level(); err != nil {
			logger.Error(self.Name, err, level, service)
		} else if mult, err = multiplier.Get(multiplier.FIVE_PERCENT); err != nil {
			logger.Error(self.Name, err, level, service)
		} else if symbols, err = self.getSymbols(client, quotes, false); err != nil {
			logger.Error(self.Name, err, level, service)
		}
		// listen to the filled orders, look for newly filled orders, automatically place new LIMIT SELL orders.
		if filled, err = self.sell(client, symbols, mult, hold, earn, service, level, filled); err != nil {
			logger.Error(self.Name, err, level, service)
		} else
		// listen to the opened orders, look for cancelled orders, send a notification.
		if opened, err = self.listen(client, symbols, service, level, opened, filled); err != nil {
			logger.Error(self.Name, err, level, service)
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
	metadata string,
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

func (self *CryptoDotCom) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, metadata string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (self *CryptoDotCom) OCO(client interface{}, market string, size float64, price, stop float64, metadata string) ([]byte, error) {
	return nil, errors.New("not implemented")
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
		price := precision.Round(aggregation.Round(e.Price(), agg), prec)
		entry := out.EntryByPrice(price)
		if entry != nil {
			entry.Size = entry.Size + e.Size()
		} else {
			entry = &model.Buy{
				Market: market,
				Price:  price,
				Size:   e.Size(),
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
		return 8, errors.New("invalid argument: client")
	}
	symbol, err := self.getSymbol(crypto, market)
	if err != nil {
		return 8, err
	}
	return symbol.PriceDecimals, nil
}

func (self *CryptoDotCom) GetSizePrec(client interface{}, market string) (int, error) {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}
	symbol, err := self.getSymbol(crypto, market)
	if err != nil {
		return 0, err
	}
	return symbol.QuantityDecimals, nil
}

func (self *CryptoDotCom) GetMaxSize(client interface{}, base, quote string, hold, earn bool, def float64, mult multiplier.Mult) float64 {
	if hold {
		if base == "CRO" {
			return 0
		}
	}
	return model.GetSizeMax(hold, earn, def, mult, func() int {
		prec, err := self.GetSizePrec(client, self.FormatMarket(base, quote))
		if err != nil {
			return 0
		}
		return prec
	})
}

func (self *CryptoDotCom) Cancel(client interface{}, market string, side model.OrderSide) error {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	orders, err := crypto.OpenOrders(market)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	for _, order := range orders {
		if ((side == model.BUY) && (order.GetSide() == exchange.BUY)) || ((side == model.SELL) && (order.GetSide() == exchange.SELL)) {
			if err := crypto.CancelOrder(market, order.Id); err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func (self *CryptoDotCom) Coalesce(client interface{}, market string, side model.OrderSide) error {
	crypto, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	for {
		orders, err := crypto.OpenOrders(market)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		// do we have any duplicates?
		price := func() float64 {
			for i1 := 0; i1 < len(orders); i1++ {
				if self.getOrderSide(orders[i1].GetSide()) == side && orders[i1].Price > 0 {
					for i2 := 0; i2 < len(orders); i2++ {
						if i2 != i1 && self.getOrderSide(orders[i2].GetSide()) == side && orders[i2].Price == orders[i1].Price {
							return orders[i1].Price
						}
					}
				}
			}
			return 0
		}()

		if price == 0 { // no duplicates, nothing to coalesce
			return nil
		}

		prec, err := self.GetSizePrec(client, market)
		if err != nil {
			return err
		}

		// get the cummulative order size
		qty := 0.0
		for _, order := range orders {
			if self.getOrderSide(order.GetSide()) == side && order.Price == price {
				qty = precision.Round((qty + order.Volume), prec)
			}
		}

		// cancel every duplicate order
		for _, order := range orders {
			if self.getOrderSide(order.GetSide()) == side && order.Price == price {
				if err := crypto.CancelOrder(market, order.Id); err != nil {
					return errors.Wrap(err, 1)
				}
			}
		}

		// wait for the above orders to get cancelled
		for {
			orders, err := crypto.OpenOrders(market)
			if err != nil {
				return errors.Wrap(err, 1)
			}
			if !func() bool {
				for _, order := range orders {
					if self.getOrderSide(order.GetSide()) == side && order.Price == price {
						return true
					}
				}
				return false
			}() {
				break
			}
		}

		// place a new order
		if _, _, err := self.Order(client, side, market, qty, price, model.LIMIT, ""); err != nil {
			return err
		}
	}
}

func (self *CryptoDotCom) Buy(client interface{}, cancel bool, market string, calls model.Calls, deviation float64, kind model.OrderType) error {
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
				if index > -1 && order.Volume == calls[index].Size {
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
				_, err = crypto.CreateOrder(market, exchange.BUY, exchange.MARKET, call.Size, 0)
			} else {
				_, err = crypto.CreateOrder(market, exchange.BUY, exchange.LIMIT, call.Size, limit)
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

func (self *CryptoDotCom) HasAlgoOrder(client interface{}, market string) (bool, error) {
	return false, nil
}

func newCryptoDotCom() model.Exchange {
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

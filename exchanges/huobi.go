//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package exchanges

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	filemutex "github.com/alexflint/go-filemutex"
	"github.com/svanas/nefertiti/aggregation"
	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/flag"
	exchange "github.com/svanas/nefertiti/huobi"
	"github.com/svanas/nefertiti/logger"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/multiplier"
	"github.com/svanas/nefertiti/notify"
	"github.com/svanas/nefertiti/precision"
	"github.com/svanas/nefertiti/pricing"
	"github.com/svanas/nefertiti/session"
)

var (
	huobiMutex *filemutex.FileMutex
)

const (
	huobiSessionFile = "huobi.time"
	huobiSessionLock = "huobi.lock"
)

func init() {
	exchange.BeforeRequest = func(method, path string) error {
		var err error

		if huobiMutex == nil {
			if huobiMutex, err = filemutex.New(session.GetSessionFile(huobiSessionLock)); err != nil {
				return err
			}
		}

		if err = huobiMutex.Lock(); err != nil {
			return err
		}

		var lastRequest *time.Time
		if lastRequest, err = session.GetLastRequest(huobiSessionFile); err != nil {
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
			huobiMutex.Unlock()
		}()
		session.SetLastRequest(huobiSessionFile, time.Now())
	}
}

type Huobi struct {
	*model.ExchangeInfo
	symbols []exchange.Symbol
}

func (self *Huobi) getBrokerId() string {
	const (
		MAX_LEN = 54
		BROKER  = "AAf68ef084"
	)
	out := BROKER
	for len(out) < MAX_LEN {
		out += strconv.Itoa(rand.Intn(10))
	}
	return out
}

func (self *Huobi) getBaseURL(sandbox bool) string {
	return self.ExchangeInfo.REST.URI
}

func (self *Huobi) indexByOrderID(orders []exchange.Order, id int64) int {
	for i, o := range orders {
		if o.Id == id {
			return i
		}
	}
	return -1
}

func (self *Huobi) getSymbols(client *exchange.Client, cached bool) ([]exchange.Symbol, error) {
	if self.symbols == nil || !cached {
		symbols, err := client.Symbols()
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		self.symbols = nil
		partition := flag.Get("partition")

		for _, symbol := range symbols {
			if symbol.Online() && symbol.Enabled() {
				if !partition.Exists || partition.Contains(symbol.Partition) {
					self.symbols = append(self.symbols, symbol)
				}
			}
		}
	}
	return self.symbols, nil
}

func (self *Huobi) getSymbolsEx(client *exchange.Client, quotes []string, cached bool) ([]exchange.Symbol, error) {
	symbols, err := self.getSymbols(client, cached)

	if err != nil {
		return nil, err
	}

	if len(quotes) == 0 {
		return symbols, err
	}

	var out []exchange.Symbol
	for _, symbol := range symbols {
		for _, quote := range quotes {
			if strings.EqualFold(symbol.QuoteCurrency, quote) {
				out = append(out, symbol)
			}
		}
	}

	return out, nil
}

func (self *Huobi) getSymbol(client *exchange.Client, market string) (*exchange.Symbol, error) {
	cached := true
	for {
		symbols, err := self.getSymbols(client, cached)
		if err != nil {
			return nil, err
		}
		for _, symbol := range symbols {
			if symbol.Symbol == market {
				return &symbol, nil
			}
		}
		if !cached {
			return nil, errors.Errorf("symbol %s does not exist", market)
		}
		cached = false
	}
}

func (self *Huobi) parseSymbol(symbols []exchange.Symbol, symbol string) (base, quote string, err error) {
	for _, market := range symbols {
		if market.Symbol == symbol {
			return market.BaseCurrency, market.QuoteCurrency, nil
		}
	}
	return "", "", errors.Errorf("symbol %s does not exist", symbol)
}

func (self *Huobi) GetInfo() *model.ExchangeInfo {
	return self.ExchangeInfo
}

func (self *Huobi) GetClient(permission model.Permission, sandbox bool) (interface{}, error) {
	if permission != model.PRIVATE {
		return exchange.New(self.getBaseURL(sandbox), "", ""), nil
	}

	var (
		err       error
		apiKey    string
		apiSecret string
	)
	if apiKey, apiSecret, err = promptForApiKeys("Huobi"); err != nil {
		return nil, err
	}

	return exchange.New(self.getBaseURL(sandbox), apiKey, apiSecret), nil
}

func (self *Huobi) GetMarkets(cached, sandbox bool, blacklist []string) ([]model.Market, error) {
	var out []model.Market

	symbols, err := self.getSymbols(exchange.New(self.getBaseURL(sandbox), "", ""), cached)
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

func (self *Huobi) FormatMarket(base, quote string) string {
	return strings.ToLower(base + quote)
}

// listen to the opened orders, look for cancelled orders, send a notification.
func (self *Huobi) listen(
	client *exchange.Client,
	quotes []string,
	service model.Notify,
	level int64,
	old []exchange.Order,
	filled []exchange.Order,
) ([]exchange.Order, error) {
	symbols, err := self.getSymbolsEx(client, quotes, true)
	if err != nil {
		return old, err
	}

	// get my opened orders
	var new []exchange.Order
	for _, market := range symbols {
		orders, err := client.OpenOrders(market.Symbol)
		if err != nil {
			return old, errors.Wrap(err, 1)
		}
		new = append(new, orders...)
	}

	// look for cancelled orders
	for _, order := range old {
		if self.indexByOrderID(new, order.Id) == -1 {
			// if this order has NOT been FILLED, then it has been cancelled.
			if self.indexByOrderID(filled, order.Id) == -1 {
				data, err := json.Marshal(order)
				if err != nil {
					return new, errors.Wrap(err, 1)
				}

				log.Println("[CANCELLED] " + string(data))

				if service != nil {
					if notify.CanSend(level, notify.CANCELLED) {
						err := service.SendMessage(order, fmt.Sprintf("Huobi - Done %v (Reason: Cancelled)", order.Side()), model.ALWAYS)
						if err != nil {
							log.Printf("[ERROR] %v", err)
						}
					}
				}
			}
		}
	}

	// look for newly opened orders
	for _, order := range new {
		if self.indexByOrderID(old, order.Id) == -1 {
			data, err := json.Marshal(order)
			if err != nil {
				return new, errors.Wrap(err, 1)
			}

			log.Println("[OPEN] " + string(data))

			if service != nil {
				if notify.CanSend(level, notify.OPENED) || (level == notify.LEVEL_DEFAULT && order.IsSell()) {
					err := service.SendMessage(order, fmt.Sprintf("Huobi - Open %v", order.Side()), model.ALWAYS)
					if err != nil {
						log.Printf("[ERROR] %v", err)
					}
				}
			}
		}
	}

	return new, nil
}

// listen to the filled orders, look for newly filled orders, automatically place new LIMIT SELL orders.
func (self *Huobi) sell(
	client *exchange.Client,
	quotes []string,
	mult multiplier.Mult,
	hold, earn model.Markets,
	service model.Notify,
	level int64,
	old []exchange.Order,
) ([]exchange.Order, error) {
	symbols, err := self.getSymbolsEx(client, quotes, true)
	if err != nil {
		return old, err
	}

	// get my filled orders
	var filled []exchange.Order
	for _, market := range symbols {
		orders, err := client.PastOrders(market.Symbol, 1, exchange.OrderStateFilled)
		if err != nil {
			return old, errors.Wrap(err, 1)
		}
		filled = append(filled, orders...)
	}

	// make a list of newly filled orders
	var new []exchange.Order
	for _, order := range filled {
		if self.indexByOrderID(old, order.Id) == -1 {
			new = append(new, order)
		}
	}

	// send notification(s)
	for _, order := range new {
		data, err := json.Marshal(order)
		if err != nil {
			logger.Error(self.Name, err, level, service)
		} else {
			log.Println("[FILLED] " + string(data))
			if notify.CanSend(level, notify.FILLED) && service != nil {
				err := service.SendMessage(order, fmt.Sprintf("Huobi - Done %v (Reason: Filled)", order.Side()), model.ALWAYS)
				if err != nil {
					log.Printf("[ERROR] %v", err)
				}
			}
		}
	}

	// has a buy order been filled? then place a sell order
	for i := 0; i < len(new); i++ {
		if new[i].IsBuy() {
			qty := func() float64 {
				if new[i].FilledAmount == 0 {
					return new[i].Amount
				} else {
					return new[i].FilledAmount - new[i].FilledFees
				}
			}()

			// add up amount(s), hereby preventing a problem with partial matches
			n := i + 1
			for n < len(new) {
				if new[n].Symbol == new[i].Symbol && new[n].Side() == new[i].Side() && new[n].Price == new[i].Price {
					qty = qty + func() float64 {
						if new[n].FilledAmount == 0 {
							return new[n].Amount
						} else {
							return new[n].FilledAmount - new[n].FilledFees
						}
					}()
					new = append(new[:n], new[n+1:]...)
				} else {
					n++
				}
			}

			prec, err := self.GetSizePrec(client, new[i].Symbol)
			if err == nil {
				qty = precision.Floor(qty, prec)
				// get base currency and desired size, calculate price, place sell order
				var (
					base  string
					quote string
				)
				base, quote, err = self.parseSymbol(symbols, new[i].Symbol)
				if err == nil {
					qty = self.GetMaxSize(client, base, quote, hold.HasMarket(new[i].Symbol), earn.HasMarket(new[i].Symbol), qty, mult)
					if qty > 0 {
						prec, err = self.GetPricePrec(client, new[i].Symbol)
						if err == nil {
							_, err = client.PlaceOrder(
								new[i].Symbol,
								exchange.OrderTypeSellLimit,
								qty,
								pricing.Multiply(new[i].Price, mult, prec),
								self.getBrokerId(),
							)
						}
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

func (self *Huobi) Sell(
	strategy model.Strategy,
	hold, earn model.Markets,
	sandbox, tweet, debug bool,
	success model.OnSuccess,
) error {
	if strategy != model.STRATEGY_STANDARD {
		return errors.New("strategy not implemented")
	}

	apiKey, apiSecret, err := promptForApiKeys("Huobi")
	if err != nil {
		return err
	}

	service, err := notify.New().Init(flag.Interactive(), true)
	if err != nil {
		return err
	}

	client := exchange.New(self.getBaseURL(sandbox), apiKey, apiSecret)

	var (
		quotes []string = []string{model.BTC}
		filled []exchange.Order
		opened []exchange.Order
	)

	flg := flag.Get("quote")
	if flg.Exists {
		quotes = flg.Split()
	} else {
		flag.Set("quote", strings.Join(quotes, ","))
	}

	symbols, err := self.getSymbolsEx(client, quotes, true)
	if err != nil {
		return err
	}

	// get my filled orders
	for _, market := range symbols {
		orders, err := client.PastOrders(market.Symbol, 1, exchange.OrderStateFilled)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		filled = append(filled, orders...)
	}

	// get my opened orders
	for _, market := range symbols {
		orders, err := client.OpenOrders(market.Symbol)
		if err != nil {
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
			level  int64 = notify.LEVEL_DEFAULT
			mult   multiplier.Mult
			quotes []string = flag.Get("quote").Split()
		)
		if level, err = notify.Level(); err != nil {
			logger.Error(self.Name, err, level, service)
		} else if mult, err = multiplier.Get(multiplier.FIVE_PERCENT); err != nil {
			logger.Error(self.Name, err, level, service)
		} else
		// listen to the filled orders, look for newly filled orders, automatically place new LIMIT SELL orders.
		if filled, err = self.sell(client, quotes, mult, hold, earn, service, level, filled); err != nil {
			logger.Error(self.Name, err, level, service)
		} else
		// listen to the opened orders, look for cancelled orders, send a notification.
		if opened, err = self.listen(client, quotes, service, level, opened, filled); err != nil {
			logger.Error(self.Name, err, level, service)
		}
	}
}

func (self *Huobi) Order(
	client interface{},
	side model.OrderSide,
	market string,
	size float64,
	price float64,
	kind model.OrderType,
	metadata string,
) (oid []byte, raw []byte, err error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, nil, errors.New("invalid argument: client")
	}

	if oid, err = huobiClient.PlaceOrder(market, func() exchange.OrderType {
		if side == model.BUY {
			if kind == model.LIMIT {
				return exchange.OrderTypeBuyLimit
			} else {
				return exchange.OrderTypeBuyMarket
			}
		} else {
			if kind == model.LIMIT {
				return exchange.OrderTypeSellLimit
			} else {
				return exchange.OrderTypeSellMarket
			}
		}
	}(), size, price, func() string {
		if metadata != "" {
			return metadata
		} else {
			return self.getBrokerId()
		}
	}()); err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	return oid, nil, nil
}

func (self *Huobi) StopLoss(client interface{}, market string, size float64, price float64, kind model.OrderType, metadata string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (self *Huobi) OCO(client interface{}, market string, size float64, price, stop float64, metadata string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (self *Huobi) GetClosed(client interface{}, market string) (model.Orders, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var (
		err    error
		orders []exchange.Order
		output model.Orders
	)

	if orders, err = huobiClient.PastOrders(market, 14, exchange.OrderStateFilled); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	for _, order := range orders {
		output = append(output, model.Order{
			Side: func() model.OrderSide {
				if order.IsSell() {
					return model.SELL
				}
				return model.BUY
			}(),
			Market:    market,
			Size:      order.Amount,
			Price:     order.Price,
			CreatedAt: order.GetCreatedAt(),
		})
	}

	return output, nil
}

func (self *Huobi) GetOpened(client interface{}, market string) (model.Orders, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var (
		err    error
		orders []exchange.Order
		output model.Orders
	)

	if orders, err = huobiClient.OpenOrders(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	for _, order := range orders {
		output = append(output, model.Order{
			Side: func() model.OrderSide {
				if order.IsSell() {
					return model.SELL
				}
				return model.BUY
			}(),
			Market:    market,
			Size:      order.Amount,
			Price:     order.Price,
			CreatedAt: order.GetCreatedAt(),
		})
	}

	return output, nil
}

func (self *Huobi) GetBook(client interface{}, market string, side model.BookSide) (interface{}, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	book, err := huobiClient.OrderBook(market)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return func() []exchange.BookEntry {
		if side == model.BOOK_SIDE_ASKS {
			return book.Asks
		} else {
			return book.Bids
		}
	}(), nil
}

func (self *Huobi) Aggregate(client, book interface{}, market string, agg float64) (model.Book, error) {
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

func (self *Huobi) GetTicker(client interface{}, market string) (float64, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	ticker, err := huobiClient.Ticker(market)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}

	return ticker.Price, nil
}

func (self *Huobi) Get24h(client interface{}, market string) (*model.Stats, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return nil, errors.New("invalid argument: client")
	}

	var (
		err error
		sum *exchange.Summary
	)
	if sum, err = huobiClient.Summary(market); err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return &model.Stats{
		Market: market,
		High:   sum.High,
		Low:    sum.Low,
		BtcVolume: func(sum *exchange.Summary) float64 {
			symbols, err := self.getSymbols(huobiClient, true)
			if err == nil {
				for _, symbol := range symbols {
					if symbol.Symbol == market {
						if strings.EqualFold(symbol.BaseCurrency, model.BTC) {
							return sum.Volume
						}
						tick, err := huobiClient.Ticker(self.FormatMarket(symbol.BaseCurrency, model.BTC))
						if err == nil {
							return sum.Volume * tick.Price
						}
					}
				}
			}
			return 0
		}(sum),
	}, nil
}

func (self *Huobi) GetPricePrec(client interface{}, market string) (int, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return 8, errors.New("invalid argument: client")
	}

	symbol, err := self.getSymbol(huobiClient, market)
	if err != nil {
		return 8, err
	}

	return symbol.PricePrecision, nil
}

func (self *Huobi) GetSizePrec(client interface{}, market string) (int, error) {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return 0, errors.New("invalid argument: client")
	}

	symbol, err := self.getSymbol(huobiClient, market)
	if err != nil {
		return 0, err
	}

	return symbol.AmountPrecision, nil
}

func (self *Huobi) GetMaxSize(client interface{}, base, quote string, hold, earn bool, def float64, mult multiplier.Mult) float64 {
	return model.GetSizeMax(hold, earn, def, mult, func() int {
		prec, err := self.GetSizePrec(client, self.FormatMarket(base, quote))
		if err != nil {
			return 0
		}
		return prec
	})
}

func (self *Huobi) Cancel(client interface{}, market string, side model.OrderSide) error {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	orders, err := huobiClient.OpenOrders(market)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	for _, order := range orders {
		if ((side == model.BUY) && order.IsBuy()) || ((side == model.SELL) && order.IsSell()) {
			if err := huobiClient.CancelOrder(order.Id); err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func (self *Huobi) Coalesce(client interface{}, market string, side model.OrderSide) error {
	return errors.New("not implemented")
}

func (self *Huobi) Buy(client interface{}, cancel bool, market string, calls model.Calls, deviation float64, kind model.OrderType) error {
	huobiClient, ok := client.(*exchange.Client)
	if !ok {
		return errors.New("invalid argument: client")
	}

	// step #1: delete the buy order(s) that are open in your book
	if cancel {
		orders, err := huobiClient.OpenOrders(market)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		for _, order := range orders {
			if order.IsBuy() {
				// do not cancel orders that we're about to re-place
				index := calls.IndexByPrice(order.Price)
				if index > -1 && order.Amount == calls[index].Size {
					calls[index].Skip = true
				} else {
					if err := huobiClient.CancelOrder(order.Id); err != nil {
						return errors.Wrap(err, 1)
					}
				}
			}
		}
	}

	// step 2: open the top X buy orders
	for _, call := range calls {
		if !call.Skip {
			var (
				qty   float64 = call.Size
				limit float64 = call.Price
			)
			if deviation != 1.0 {
				kind, limit = call.Deviate(self, client, kind, deviation)
			}
			// --- BEGIN --- order value should be greater or equal to X ------
			symbol, err := self.getSymbol(huobiClient, market)
			if err != nil {
				return err
			}
			if symbol.MinOrderValue > 0 {
				if limit == 0 {
					if limit, err = self.GetTicker(client, market); err != nil {
						return err
					}
				}
				if (qty * limit) < symbol.MinOrderValue {
					prec, err := self.GetSizePrec(client, market)
					if err != nil {
						return err
					}
					qty = precision.Ceil((symbol.MinOrderValue / limit), prec)
				}
			}
			// ---- END ---- order value should be greater or equal to X ------
			if _, err := huobiClient.PlaceOrder(market, func() exchange.OrderType {
				if kind == model.MARKET {
					return exchange.OrderTypeBuyMarket
				} else {
					return exchange.OrderTypeBuyLimit
				}
			}(), qty, limit, self.getBrokerId()); err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	return nil
}

func (self *Huobi) IsLeveragedToken(name string) bool {
	return false
}

func (self *Huobi) HasAlgoOrder(client interface{}, market string) (bool, error) {
	return false, nil
}

func newHuobi() model.Exchange {
	return &Huobi{
		ExchangeInfo: &model.ExchangeInfo{
			Code: "HUBI",
			Name: "Huobi",
			URL:  "https://www.huobi.com",
			REST: model.Endpoint{
				URI: "https://api.huobi.pro",
			},
			WebSocket: model.Endpoint{
				URI: "wss://api.huobi.pro/ws",
			},
			Country: "Seychelles",
		},
	}
}

package hitbtc

import (
	"encoding/json"
	"strconv"
	"time"
)

const (
	ORDER_TYPE_LIMIT       = "limit"
	ORDER_TYPE_MARKET      = "market"
	ORDER_TYPE_STOP_LIMIT  = "stopLimit"
	ORDER_TYPE_STOP_MARKET = "stopMarket"
)

const (
	GTC = "GTC"
	IOC = "IOC"
	FOK = "FOK"
	GTD = "GTD"
)

type Order struct {
	Id            uint64    `json:"id"`
	ClientOrderId string    `json:"clientOrderId"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Status        string    `json:"status"`
	Type          string    `json:"type"`
	TimeInForce   string    `json:"timeInForce"`
	Quantity      float64   `json:"quantity,string"`
	Price         string    `json:"price"`
	CumQuantity   float64   `json:"cumQuantity,string"`
	Created       time.Time `json:"createdAt"`
	Updated       time.Time `json:"updatedAt"`
	StopPrice     string    `json:"stopPrice"`
	Expire        time.Time `json:"expireTime"`
}

func (o *Order) ParsePrice() float64 {
	out, err := strconv.ParseFloat(o.Price, 64)
	if err == nil {
		return out
	}
	return 0
}

func (o *Order) ParseStopPrice() float64 {
	out, err := strconv.ParseFloat(o.StopPrice, 64)
	if err == nil {
		return out
	}
	return 0
}

func (t *Order) UnmarshalJSON(data []byte) error {
	var err error
	type Alias Order
	aux := &struct {
		Created string `json:"createdAt"`
		Updated string `json:"updatedAt"`
		Expire  string `json:"expireTime"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Created != "" {
		t.Created, err = time.Parse("2006-01-02T15:04:05.999Z", aux.Created)
		if err != nil {
			return err
		}
	}
	if aux.Updated != "" {
		t.Updated, err = time.Parse("2006-01-02T15:04:05.999Z", aux.Updated)
		if err != nil {
			return err
		}
	}
	if aux.Expire != "" {
		t.Expire, err = time.Parse("2006-01-02T15:04:05.999Z", aux.Expire)
		if err != nil {
			return err
		}
	}
	return nil
}

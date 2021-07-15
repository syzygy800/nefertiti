package hitbtc

import (
	"encoding/json"
	"strconv"
	"time"
)

type Ticker struct {
	Ask         float64   `json:"ask,string"`
	Bid         float64   `json:"bid,string"`
	Last        float64   `json:"last,string"`
	Open        float64   `json:"open,string"`
	Low         float64   `json:"low,string"`
	High        float64   `json:"high,string"`
	Volume      float64   `json:"volume,string"`
	VolumeQuote float64   `json:"volumeQuote,string"`
	Timestamp   time.Time `json:"timestamp"`
	Symbol      string    `json:"symbol"`
}

func (t *Ticker) UnmarshalJSON(data []byte) error {
	var err error
	type Alias Ticker
	aux := &struct {
		Ask         string `json:"ask"`
		Bid         string `json:"bid"`
		Last        string `json:"last"`
		Open        string `json:"open"`
		Low         string `json:"low"`
		High        string `json:"high"`
		Volume      string `json:"volume"`
		VolumeQuote string `json:"volumeQuote"`
		Timestamp   string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}
	//--- BEGIN --- svanas --- 2018-04-04 -------------------------------------
	if t.Ask, err = strconv.ParseFloat(aux.Ask, 64); err != nil {
		return err
	}
	if t.Bid, err = strconv.ParseFloat(aux.Bid, 64); err != nil {
		return err
	}
	if t.Last, err = strconv.ParseFloat(aux.Last, 64); err != nil {
		return err
	}
	if t.Open, err = strconv.ParseFloat(aux.Open, 64); err != nil {
		return err
	}
	if t.Low, err = strconv.ParseFloat(aux.Low, 64); err != nil {
		return err
	}
	if t.High, err = strconv.ParseFloat(aux.High, 64); err != nil {
		return err
	}
	if t.Volume, err = strconv.ParseFloat(aux.Volume, 64); err != nil {
		return err
	}
	if t.VolumeQuote, err = strconv.ParseFloat(aux.VolumeQuote, 64); err != nil {
		return err
	}
	//---- END ---- svanas --- 2018-04-04 -------------------------------------
	if t.Timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", aux.Timestamp); err != nil {
		return err
	}
	return nil
}

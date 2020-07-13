package cexio

type Ticker struct {
	Timestamp int64   `json:"timestamp,string"`
	Low       float64 `json:"low,string"`
	High      float64 `json:"high,string"`
	Last      float64 `json:"last,string"`
	Volume    float64 `json:"volume,string"`
	Volume30d float64 `json:"volume30d,string"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
}

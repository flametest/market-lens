package model

import "github.com/shopspring/decimal"

type MarketType string

const (
	MarketUSStock MarketType = "stock_us"
	MarketCNStock MarketType = "stock_cn"
	MarketForex   MarketType = "forex"
)

type Interval string

const (
	Interval1Min   Interval = "1m"
	Interval5Min   Interval = "5m"
	Interval15Min  Interval = "15m"
	Interval30Min  Interval = "30m"
	Interval1Hour  Interval = "1h"
	Interval4Hour  Interval = "4h"
	Interval1Day   Interval = "1d"
	Interval1Week  Interval = "1w"
	Interval1Month Interval = "1M"
)

type SignalType string

const (
	SignalBuy     SignalType = "BUY"
	SignalSell    SignalType = "SELL"
	SignalNeutral SignalType = "NEUTRAL"
)

type SymbolInfo struct {
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Market   MarketType      `json:"market"`
	Exchange string          `json:"exchange"`
	Type     string          `json:"type"`
	LotSize  decimal.Decimal `json:"lotSize"`
	TickSize decimal.Decimal `json:"tickSize"`
	Currency string          `json:"currency"`
	Enabled  bool            `json:"enabled"`
}

type Tick struct {
	Symbol        string          `json:"symbol"`
	Price         decimal.Decimal `json:"price"`
	Volume        decimal.Decimal `json:"volume"`
	High          decimal.Decimal `json:"high"`
	Low           decimal.Decimal `json:"low"`
	Open          decimal.Decimal `json:"open"`
	Change        decimal.Decimal `json:"change"`
	ChangePercent decimal.Decimal `json:"changePercent"`
	Timestamp     int64           `json:"timestamp"`
	Source        string          `json:"source"`
}

type Candle struct {
	Symbol    string          `json:"symbol"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`
	Timestamp int64           `json:"timestamp"`
	Interval  Interval        `json:"interval"`
	Source    string          `json:"source"`
}

type Signal struct {
	ID        string          `json:"id"`
	Symbol    string          `json:"symbol"`
	Source    string          `json:"source"`
	Type      SignalType      `json:"type"`
	Strength  float64         `json:"strength"`
	Price     decimal.Decimal `json:"price"`
	Timestamp int64           `json:"timestamp"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}

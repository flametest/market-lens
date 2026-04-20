package model

import "github.com/shopspring/decimal"

type OrderSide string

const (
	OrderBuy  OrderSide = "BUY"
	OrderSell OrderSide = "SELL"
)

type OrderType string

const (
	OrderMarket OrderType = "MARKET"
	OrderLimit  OrderType = "LIMIT"
)

type OrderStatus string

const (
	OrderNew       OrderStatus = "NEW"
	OrderFilled    OrderStatus = "FILLED"
	OrderCancelled OrderStatus = "CANCELLED"
	OrderRejected  OrderStatus = "REJECTED"
)

type Order struct {
	ID           string          `json:"id"`
	AccountID    string          `json:"accountId"`
	Symbol       string          `json:"symbol"`
	Side         OrderSide       `json:"side"`
	Type         OrderType       `json:"type"`
	Status       OrderStatus     `json:"status"`
	Quantity     decimal.Decimal `json:"quantity"`
	FilledQty    decimal.Decimal `json:"filledQty"`
	Price        decimal.Decimal `json:"price"`
	AvgFillPrice decimal.Decimal `json:"avgFillPrice"`
	Fee          decimal.Decimal `json:"fee"`
	Slippage     decimal.Decimal `json:"slippage"`
	Strategy     string          `json:"strategy"`
	Reason       string          `json:"reason"`
	CreatedAt    int64           `json:"createdAt"`
	UpdatedAt    int64           `json:"updatedAt"`
}

type Position struct {
	ID               string          `json:"id"`
	AccountID        string          `json:"accountId"`
	Symbol           string          `json:"symbol"`
	Side             OrderSide       `json:"side"`
	Quantity         decimal.Decimal `json:"quantity"`
	AvgPrice         decimal.Decimal `json:"avgPrice"`
	UnrealizedPnL    decimal.Decimal `json:"unrealizedPnl"`
	RealizedPnL      decimal.Decimal `json:"realizedPnl"`
	OpenedAt         int64           `json:"openedAt"`
	UpdatedAt        int64           `json:"updatedAt"`
}

type Account struct {
	ID         string          `json:"id"`
	Cash       decimal.Decimal `json:"cash"`
	TotalValue decimal.Decimal `json:"totalValue"`
	Currency   string          `json:"currency"`
	CreatedAt  int64           `json:"createdAt"`
	UpdatedAt  int64           `json:"updatedAt"`
}

type Fill struct {
	OrderID   string          `json:"orderId"`
	Symbol    string          `json:"symbol"`
	Side      OrderSide       `json:"side"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"quantity"`
	Fee       decimal.Decimal `json:"fee"`
	Slippage  decimal.Decimal `json:"slippage"`
	Strategy  string          `json:"strategy"`
	Timestamp int64           `json:"timestamp"`
}

type OrderDecision struct {
	Symbol   string          `json:"symbol"`
	Side     OrderSide       `json:"side"`
	Type     OrderType       `json:"type"`
	Quantity decimal.Decimal `json:"quantity"`
	Price    decimal.Decimal `json:"price"`
	Reason   string          `json:"reason"`
	Strategy string          `json:"strategy"`
}

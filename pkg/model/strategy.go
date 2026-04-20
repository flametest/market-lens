package model

import "github.com/shopspring/decimal"

type AIMode string

const (
	AIModeOff     AIMode = "off"
	AIModeOn      AIMode = "on"
	AIModeCompare AIMode = "compare"
)

type RiskConfig struct {
	MaxPosition      decimal.Decimal `json:"maxPosition"`
	MaxDrawdown      float64         `json:"maxDrawdown"`
	MaxDailyTrades   int             `json:"maxDailyTrades"`
	MaxOrderAmount   decimal.Decimal `json:"maxOrderAmount"`
	MinTradeInterval int64           `json:"minTradeInterval"`
}

type AdjustmentFactors struct {
	StrongPositiveBuy  float64 `json:"strongPositiveBuy"`
	StrongPositiveSell float64 `json:"strongPositiveSell"`
	WeakPositiveBuy    float64 `json:"weakPositiveBuy"`
	StrongNegativeBuy  float64 `json:"strongNegativeBuy"`
	StrongNegativeSell float64 `json:"strongNegativeSell"`
	WeakNegativeSell   float64 `json:"weakNegativeSell"`
}

type AIRiskThresholds struct {
	PanicThreshold      float64 `json:"panicThreshold"`
	OverheatThreshold   float64 `json:"overheatThreshold"`
	EventPauseThreshold float64 `json:"eventPauseThreshold"`
	PositionReduction   float64 `json:"positionReduction"`
}

type AIConfig struct {
	Enabled              bool              `json:"enabled"`
	SignalParticipation  bool              `json:"signalParticipation"`
	ConfidenceAdjustment bool              `json:"confidenceAdjustment"`
	RiskEnhancement      bool              `json:"riskEnhancement"`
	SignalWeight         float64           `json:"signalWeight"`
	AdjustmentFactors    AdjustmentFactors `json:"adjustmentFactors"`
	RiskThresholds       AIRiskThresholds  `json:"riskThresholds"`
}

type StrategyConfig struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Enabled     bool                   `json:"enabled"`
	Symbols     []string               `json:"symbols"`
	Params      map[string]any         `json:"params"`
	Risk        RiskConfig             `json:"risk"`
	AI          AIConfig               `json:"ai"`
	CreatedAt   int64                  `json:"createdAt"`
	UpdatedAt   int64                  `json:"updatedAt"`
}

type BacktestConfig struct {
	Symbol      string          `json:"symbol"`
	StartDate   int64           `json:"startDate"`
	EndDate     int64           `json:"endDate"`
	Interval    Interval        `json:"interval"`
	InitialCash decimal.Decimal `json:"initialCash"`
	Commission  decimal.Decimal `json:"commission"`
	Slippage    decimal.Decimal `json:"slippage"`
	Strategy    StrategyConfig  `json:"strategy"`
	Risk        RiskConfig      `json:"risk"`
	AIMode      AIMode          `json:"aiMode"`
}

type BacktestResult struct {
	TotalReturn  decimal.Decimal `json:"totalReturn"`
	AnnualReturn decimal.Decimal `json:"annualReturn"`
	MaxDrawdown  decimal.Decimal `json:"maxDrawdown"`
	SharpeRatio  decimal.Decimal `json:"sharpeRatio"`
	WinRate      decimal.Decimal `json:"winRate"`
	TotalTrades  int             `json:"totalTrades"`
	ProfitTrades int             `json:"profitTrades"`
	LossTrades   int             `json:"lossTrades"`
	Trades       []SimulatedTrade `json:"trades"`
	EquityCurve  []EquityPoint   `json:"equityCurve"`
}

type SimulatedTrade struct {
	Symbol    string          `json:"symbol"`
	Side      OrderSide       `json:"side"`
	EntryTime int64           `json:"entryTime"`
	ExitTime  int64           `json:"exitTime"`
	EntryPrice decimal.Decimal `json:"entryPrice"`
	ExitPrice  decimal.Decimal `json:"exitPrice"`
	Quantity   decimal.Decimal `json:"quantity"`
	PnL        decimal.Decimal `json:"pnl"`
	Fee        decimal.Decimal `json:"fee"`
}

type EquityPoint struct {
	Timestamp int64           `json:"timestamp"`
	Value     decimal.Decimal `json:"value"`
}

type BacktestRun struct {
	ID            string          `json:"id"`
	Config        BacktestConfig  `json:"config"`
	Status        string          `json:"status"`
	AIMode        AIMode          `json:"aiMode"`
	Result        *BacktestResult `json:"result,omitempty"`
	CompareResult *BacktestResult `json:"compareResult,omitempty"`
	Error         string          `json:"error,omitempty"`
	CreatedAt     int64           `json:"createdAt"`
	CompletedAt   int64           `json:"completedAt,omitempty"`
}

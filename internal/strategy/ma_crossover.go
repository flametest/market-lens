package strategy

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/pkg/indicator"
	"github.com/flametest/market-lens/pkg/model"
)

type MACrossover struct {
	shortPeriod int
	longPeriod  int
	allocRatio  float64
	cfg         model.StrategyConfig
}

func NewMACrossover(cfg model.StrategyConfig) *MACrossover {
	short := intParam(cfg.Params, "shortPeriod", 5)
	long := intParam(cfg.Params, "longPeriod", 20)
	return &MACrossover{
		shortPeriod: short,
		longPeriod:  long,
		allocRatio:  0.2,
		cfg:         cfg,
	}
}

func (m *MACrossover) Name() string { return "ma_crossover" }

func (m *MACrossover) OnSignal(ctx context.Context, sig model.Signal, candles []model.Candle) (*model.OrderDecision, error) {
	if len(candles) < m.longPeriod+1 {
		return nil, nil
	}

	shortMA := &indicator.MA{Period: m.shortPeriod}
	longMA := &indicator.MA{Period: m.longPeriod}

	shortResult := shortMA.Compute(candles)
	longResult := longMA.Compute(candles)

	if len(shortResult.Values) < 2 || len(longResult.Values) < 2 {
		return nil, nil
	}

	curShort := shortResult.Values[len(shortResult.Values)-1].Value
	prevShort := shortResult.Values[len(shortResult.Values)-2].Value
	curLong := longResult.Values[len(longResult.Values)-1].Value
	prevLong := longResult.Values[len(longResult.Values)-1].Value

	if len(longResult.Values) >= 2 {
		prevLong = longResult.Values[len(longResult.Values)-2].Value
	}

	var side model.OrderSide
	if prevShort <= prevLong && curShort > curLong {
		side = model.OrderBuy
	} else if prevShort >= prevLong && curShort < curLong {
		side = model.OrderSell
	} else {
		return nil, nil
	}

	price := candles[len(candles)-1].Close
	qty := positionSize("100000", price, m.allocRatio)
	if qty.IsZero() {
		return nil, nil
	}

	return &model.OrderDecision{
		Symbol:   sig.Symbol,
		Side:     side,
		Type:     model.OrderMarket,
		Quantity: qty,
		Price:    price,
		Reason:   "MA crossover signal",
		Strategy: m.Name(),
	}, nil
}

func intParam(params map[string]any, key string, def int) int {
	v, ok := params[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return def
	}
}

func decParam(params map[string]any, key string, def float64) decimal.Decimal {
	v, ok := params[key]
	if !ok {
		return decimal.NewFromFloat(def)
	}
	switch n := v.(type) {
	case float64:
		return decimal.NewFromFloat(n)
	default:
		return decimal.NewFromFloat(def)
	}
}

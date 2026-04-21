package strategy

import (
	"context"

	"github.com/flametest/market-lens/pkg/model"
)

type RSIReversion struct {
	period     int
	oversold   float64
	overbought float64
	allocRatio float64
	cfg        model.StrategyConfig
}

func NewRSIReversion(cfg model.StrategyConfig) *RSIReversion {
	return &RSIReversion{
		period:     intParam(cfg.Params, "period", 14),
		oversold:   decParam(cfg.Params, "oversold", 30).InexactFloat64(),
		overbought: decParam(cfg.Params, "overbought", 70).InexactFloat64(),
		allocRatio: 0.15,
		cfg:        cfg,
	}
}

func (r *RSIReversion) Name() string { return "rsi_reversion" }

func (r *RSIReversion) OnSignal(ctx context.Context, sig model.Signal, candles []model.Candle) (*model.OrderDecision, error) {
	if sig.Source != "RSI" || len(candles) == 0 {
		return nil, nil
	}

	var side model.OrderSide
	if sig.Type == model.SignalBuy && sig.Strength > 0 {
		side = model.OrderBuy
	} else if sig.Type == model.SignalSell && sig.Strength > 0 {
		side = model.OrderSell
	} else {
		return nil, nil
	}

	price := candles[len(candles)-1].Close
	qty := positionSize("100000", price, r.allocRatio)
	if qty.IsZero() {
		return nil, nil
	}

	return &model.OrderDecision{
		Symbol:   sig.Symbol,
		Side:     side,
		Type:     model.OrderMarket,
		Quantity: qty,
		Price:    price,
		Reason:   "RSI mean reversion signal",
		Strategy: r.Name(),
	}, nil
}

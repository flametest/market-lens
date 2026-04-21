package analysis

import (
	"context"
	"log/slog"
	"sync"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/indicator"
	"github.com/flametest/market-lens/pkg/model"
)

type Engine struct {
	indicators map[string]indicator.Indicator
	repo       *data.Repository
	bus        *eventbus.InMemoryBus
	logger     *slog.Logger
	mu         sync.RWMutex
}

func NewEngine(repo *data.Repository, bus *eventbus.InMemoryBus, logger *slog.Logger) *Engine {
	return &Engine{
		indicators: make(map[string]indicator.Indicator),
		repo:       repo,
		bus:        bus,
		logger:     logger,
	}
}

func (e *Engine) Register(ind indicator.Indicator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.indicators[ind.Name()] = ind
}

func (e *Engine) Start(ctx context.Context) error {
	e.RegisterDefaults()

	e.bus.SubscribeFunc(model.EventMarketCandle, func(ev any) {
		candle, ok := ev.(model.Candle)
		if !ok {
			return
		}
		e.onCandle(ctx, candle)
	})

	e.bus.SubscribeFunc(model.EventMarketTick, func(ev any) {
		tick, ok := ev.(model.Tick)
		if !ok {
			return
		}
		e.bus.Publish(model.EventMarketCandle, model.Candle{
			Symbol:    tick.Symbol,
			Open:      tick.Open,
			High:      tick.High,
			Low:       tick.Low,
			Close:     tick.Price,
			Volume:    tick.Volume,
			Timestamp: tick.Timestamp,
			Interval:  model.Interval1Day,
			Source:    tick.Source,
		})
	})

	e.logger.Info("analysis engine started")
	return nil
}

func (e *Engine) RegisterDefaults() {
	e.Register(&indicator.MA{Period: 5})
	e.Register(&indicator.MA{Period: 20})
	e.Register(&indicator.EMA{Period: 12})
	e.Register(&indicator.RSI{Period: 14, Oversold: 30, Overbought: 70})
	e.Register(&indicator.MACD{FastPeriod: 12, SlowPeriod: 26, SignalPeriod: 9})
	e.Register(&indicator.Bollinger{Period: 20, StdDev: 2})
	e.Register(&indicator.VWAP{})
}

func (e *Engine) onCandle(ctx context.Context, candle model.Candle) {
	candles, err := e.repo.GetCandles(ctx, candle.Symbol, candle.Interval, 0, candle.Timestamp)
	if err != nil {
		e.logger.Error("failed to get candles for analysis",
			slog.String("symbol", candle.Symbol),
			slog.String("error", err.Error()))
		return
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, ind := range e.indicators {
		result := ind.Compute(candles)
		if result.Signal == model.SignalNeutral && result.Strength == 0 {
			continue
		}

		sig := model.Signal{
			ID:        data.GenerateID(),
			Symbol:    candle.Symbol,
			Source:    ind.Name(),
			Type:      result.Signal,
			Strength:  result.Strength,
			Price:     candle.Close,
			Timestamp: candle.Timestamp,
		}

		if err := e.repo.SaveSignal(ctx, sig); err != nil {
			e.logger.Error("failed to save signal", slog.String("error", err.Error()))
		}
		e.bus.Publish(model.EventAnalysisSignal, sig)
	}
}

func (e *Engine) ComputeIndicators(candles []model.Candle) map[string]indicator.Result {
	e.mu.RLock()
	defer e.mu.RUnlock()

	results := make(map[string]indicator.Result)
	for name, ind := range e.indicators {
		results[name] = ind.Compute(candles)
	}
	return results
}

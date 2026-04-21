package strategy

import (
	"context"
	"log/slog"
	"sync"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/model"
)

type Strategy interface {
	Name() string
	OnSignal(ctx context.Context, signal model.Signal, candles []model.Candle) (*model.OrderDecision, error)
}

type Engine struct {
	strategies map[string]Strategy
	repo       *data.Repository
	bus        *eventbus.InMemoryBus
	logger     *slog.Logger
	mu         sync.RWMutex
	configs    map[string]model.StrategyConfig
}

func NewEngine(repo *data.Repository, bus *eventbus.InMemoryBus, logger *slog.Logger) *Engine {
	return &Engine{
		strategies: make(map[string]Strategy),
		repo:       repo,
		bus:        bus,
		logger:     logger,
		configs:    make(map[string]model.StrategyConfig),
	}
}

func (e *Engine) Register(s Strategy, cfg model.StrategyConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategies[s.Name()] = s
	e.configs[s.Name()] = cfg
}

func (e *Engine) Start(ctx context.Context) error {
	configs, err := e.repo.GetStrategyConfigs(ctx)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if cfg.Enabled {
			switch cfg.Name {
			case "ma_crossover":
				ma := NewMACrossover(cfg)
				e.Register(ma, cfg)
			case "rsi_reversion":
				rsi := NewRSIReversion(cfg)
				e.Register(rsi, cfg)
			}
		}
	}

	e.bus.SubscribeFunc(model.EventAnalysisSignal, func(ev any) {
		sig, ok := ev.(model.Signal)
		if !ok {
			return
		}
		e.onSignal(ctx, sig)
	})

	e.logger.Info("strategy engine started", slog.Int("strategies", len(e.strategies)))
	return nil
}

func (e *Engine) onSignal(ctx context.Context, sig model.Signal) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for name, s := range e.strategies {
		cfg, ok := e.configs[name]
		if !ok || !cfg.Enabled {
			continue
		}

		matched := false
		for _, sym := range cfg.Symbols {
			if sym == sig.Symbol {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		candles, err := e.repo.GetCandles(ctx, sig.Symbol, model.Interval1Day, 0, sig.Timestamp)
		if err != nil {
			e.logger.Error("failed to get candles", slog.String("error", err.Error()))
			continue
		}

		decision, err := s.OnSignal(ctx, sig, candles)
		if err != nil {
			e.logger.Error("strategy error", slog.String("strategy", name), slog.String("error", err.Error()))
			continue
		}
		if decision == nil {
			continue
		}

		e.logger.Info("strategy decision",
			slog.String("strategy", name),
			slog.String("symbol", decision.Symbol),
			slog.String("side", string(decision.Side)),
		)

		e.bus.Publish(model.EventStrategyDecision, *decision)
	}
}

func (e *Engine) GetStrategies() map[string]model.StrategyConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]model.StrategyConfig, len(e.configs))
	for k, v := range e.configs {
		result[k] = v
	}
	return result
}

func (e *Engine) UpdateConfig(name string, cfg model.StrategyConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.configs[name]; ok {
		e.configs[name] = cfg
	}
}

func positionSize(cash string, price decimal.Decimal, maxRatio float64) decimal.Decimal {
	cashDec := decimal.RequireFromString(cash)
	alloc := cashDec.Mul(decimal.NewFromFloat(maxRatio))
	if price.IsZero() {
		return decimal.Zero
	}
	qty := alloc.Div(price)
	return qty.Floor()
}

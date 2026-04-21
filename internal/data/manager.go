package data

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/exchange"
	"github.com/flametest/market-lens/pkg/model"
)

type Manager struct {
	mu         sync.RWMutex
	providers  map[string]exchange.DataProvider
	activeName string
	db         *DB
	bus        *eventbus.InMemoryBus
	logger     *slog.Logger
	symbols    []string
}

func NewManager(providers map[string]exchange.DataProvider, activeName string, db *DB, bus *eventbus.InMemoryBus, logger *slog.Logger) *Manager {
	return &Manager{
		providers:  providers,
		activeName: activeName,
		db:         db,
		bus:        bus,
		logger:     logger,
	}
}

func (m *Manager) active() exchange.DataProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[m.activeName]
}

func (m *Manager) ActiveProviderName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeName
}

func (m *Manager) ProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

func (m *Manager) SwitchProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.providers[name]
	if !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	old := m.providers[m.activeName]
	if old != nil {
		old.Close()
	}

	m.activeName = name

	if len(m.symbols) > 0 {
		if err := p.Connect(context.Background()); err != nil {
			m.logger.Error("failed to connect provider", slog.String("provider", name), slog.String("error", err.Error()))
			return err
		}
		p.SetOnTick(func(tick model.Tick) {
			if err := m.saveAndPublishTick(context.Background(), tick); err != nil {
				m.logger.Error("failed to save tick",
					slog.String("symbol", tick.Symbol),
					slog.String("error", err.Error()),
				)
			}
		})
		if err := p.Subscribe(m.symbols); err != nil {
			m.logger.Error("failed to subscribe", slog.String("provider", name), slog.String("error", err.Error()))
			return err
		}
	}

	m.logger.Info("switched data provider", slog.String("from", m.activeName), slog.String("to", name))
	return nil
}

func (m *Manager) Start(ctx context.Context, symbols []string) error {
	m.mu.Lock()
	m.symbols = symbols
	p := m.providers[m.activeName]
	m.mu.Unlock()

	p.SetOnTick(func(tick model.Tick) {
		if err := m.saveAndPublishTick(ctx, tick); err != nil {
			m.logger.Error("failed to save tick",
				slog.String("symbol", tick.Symbol),
				slog.String("error", err.Error()),
			)
		}
	})

	if err := p.Connect(ctx); err != nil {
		return err
	}

	if err := p.Subscribe(symbols); err != nil {
		return err
	}

	m.logger.Info("data manager started", slog.Any("symbols", symbols))
	return nil
}

func (m *Manager) Stop() error {
	return m.active().Close()
}

func (m *Manager) saveAndPublishTick(ctx context.Context, tick model.Tick) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ticks (symbol, price, volume, high, low, open_price, source, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tick.Symbol, tick.Price.String(), tick.Volume.String(),
		tick.High.String(), tick.Low.String(), tick.Open.String(),
		tick.Source, tick.Timestamp,
	)
	if err != nil {
		return err
	}

	m.bus.Publish(model.EventMarketTick, tick)
	return nil
}

func (m *Manager) SaveCandles(ctx context.Context, candles []model.Candle) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO candles (symbol, interval, timestamp, open, high, low, close, volume, source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range candles {
		_, err := stmt.ExecContext(ctx,
			c.Symbol, c.Interval, c.Timestamp,
			c.Open.String(), c.High.String(), c.Low.String(),
			c.Close.String(), c.Volume.String(), c.Source,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (m *Manager) GetCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT symbol, interval, timestamp, open, high, low, close, volume, source
		 FROM candles WHERE symbol = ? AND interval = ? AND timestamp >= ? AND timestamp <= ?
		 ORDER BY timestamp ASC`,
		symbol, interval, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCandles(rows)
}

func (m *Manager) GetLatestTick(ctx context.Context, symbol string) (*model.Tick, error) {
	row := m.db.QueryRowContext(ctx,
		`SELECT symbol, price, volume, high, low, open_price, source, timestamp
		 FROM ticks WHERE symbol = ? ORDER BY timestamp DESC LIMIT 1`, symbol)

	var tick model.Tick
	var price, vol, high, low, openP string
	if err := row.Scan(&tick.Symbol, &price, &vol, &high, &low, &openP, &tick.Source, &tick.Timestamp); err != nil {
		return nil, err
	}
	tick.Price = dec(price)
	tick.Volume = dec(vol)
	tick.High = dec(high)
	tick.Low = dec(low)
	tick.Open = dec(openP)
	return &tick, nil
}

func (m *Manager) FetchAndStoreCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error) {
	candles, err := m.active().FetchCandles(ctx, symbol, interval, start, end)
	if err != nil {
		return nil, err
	}
	if len(candles) > 0 {
		if err := m.SaveCandles(ctx, candles); err != nil {
			return nil, err
		}
	}
	return candles, nil
}

func (m *Manager) GetQuote(ctx context.Context, symbol string) (*model.Tick, error) {
	return m.active().FetchQuote(ctx, symbol)
}

func (m *Manager) SearchSymbols(ctx context.Context, query string) ([]model.SymbolInfo, error) {
	return m.active().SearchSymbols(ctx, query)
}

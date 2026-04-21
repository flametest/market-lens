package data

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/pkg/model"
)

func TestRepository_Signals(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	sig1 := model.Signal{
		ID: "sig-1", Symbol: "AAPL", Source: "MA",
		Type: model.SignalBuy, Strength: 0.8,
		Price: decimal.NewFromFloat(150.0), Timestamp: 1000,
	}
	sig2 := model.Signal{
		ID: "sig-2", Symbol: "AAPL", Source: "RSI",
		Type: model.SignalSell, Strength: 0.6,
		Price: decimal.NewFromFloat(152.0), Timestamp: 2000,
	}
	sig3 := model.Signal{
		ID: "sig-3", Symbol: "NVDA", Source: "MACD",
		Type: model.SignalBuy, Strength: 0.9,
		Price: decimal.NewFromFloat(800.0), Timestamp: 1500,
	}

	require.NoError(t, repo.SaveSignal(ctx, sig1))
	require.NoError(t, repo.SaveSignal(ctx, sig2))
	require.NoError(t, repo.SaveSignal(ctx, sig3))

	t.Run("list all signals", func(t *testing.T) {
		signals, err := repo.GetSignals(ctx, "", 10)
		require.NoError(t, err)
		assert.Len(t, signals, 3)
	})

	t.Run("filter by symbol", func(t *testing.T) {
		signals, err := repo.GetSignals(ctx, "AAPL", 10)
		require.NoError(t, err)
		assert.Len(t, signals, 2)
		for _, s := range signals {
			assert.Equal(t, "AAPL", s.Symbol)
		}
	})

	t.Run("limit works", func(t *testing.T) {
		signals, err := repo.GetSignals(ctx, "", 2)
		require.NoError(t, err)
		assert.Len(t, signals, 2)
	})
}

func TestRepository_Candles(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	candle := model.Candle{
		Symbol: "AAPL", Interval: model.Interval1Day,
		Timestamp: 1000,
		Open:  dec("150.0"), High: dec("155.0"),
		Low: dec("149.0"), Close: dec("153.0"),
		Volume: dec("1000000"), Source: "finnhub",
	}
	require.NoError(t, repo.SaveCandle(ctx, candle))

	t.Run("get candles", func(t *testing.T) {
		candles, err := repo.GetCandles(ctx, "AAPL", model.Interval1Day, 0, 2000)
		require.NoError(t, err)
		require.Len(t, candles, 1)
		assert.Equal(t, "AAPL", candles[0].Symbol)
		assert.True(t, candles[0].Close.Equal(dec("153.0")))
	})

	t.Run("no candles for wrong symbol", func(t *testing.T) {
		candles, err := repo.GetCandles(ctx, "GOOGL", model.Interval1Day, 0, 2000)
		require.NoError(t, err)
		assert.Len(t, candles, 0)
	})
}

func TestRepository_Account(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	account := model.Account{
		ID: "acc-1", Cash: dec("100000.00"),
		TotalValue: dec("100000.00"), Currency: "USD",
		CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
	}
	require.NoError(t, repo.CreateAccount(ctx, account))

	t.Run("get account", func(t *testing.T) {
		a, err := repo.GetAccount(ctx, "acc-1")
		require.NoError(t, err)
		assert.True(t, a.Cash.Equal(dec("100000.00")))
	})

	t.Run("update account", func(t *testing.T) {
		a, _ := repo.GetAccount(ctx, "acc-1")
		a.Cash = dec("90000.00")
		a.TotalValue = dec("120000.00")
		require.NoError(t, repo.UpdateAccount(ctx, *a))

		updated, err := repo.GetAccount(ctx, "acc-1")
		require.NoError(t, err)
		assert.True(t, updated.Cash.Equal(dec("90000.00")))
		assert.True(t, updated.TotalValue.Equal(dec("120000.00")))
	})
}

func TestRepository_Orders(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	order := model.Order{
		ID: "ord-1", AccountID: "acc-1", Symbol: "AAPL",
		Side: model.OrderBuy, Type: model.OrderMarket, Status: model.OrderNew,
		Quantity: dec("10"), FilledQty: dec("0"),
		Price: dec("150.00"), CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
	}
	require.NoError(t, repo.SaveOrder(ctx, order))

	orders, err := repo.GetOrders(ctx, "acc-1", 10)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, "AAPL", orders[0].Symbol)
	assert.True(t, orders[0].Quantity.Equal(dec("10")))
}

func TestRepository_Positions(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	pos := model.Position{
		ID: "pos-1", AccountID: "acc-1", Symbol: "AAPL",
		Side: model.OrderBuy, Quantity: dec("50"),
		AvgPrice: dec("148.50"), UnrealizedPnL: dec("245.00"),
		OpenedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
	}
	require.NoError(t, repo.UpsertPosition(ctx, pos))

	t.Run("get position", func(t *testing.T) {
		p, err := repo.GetPosition(ctx, "acc-1", "AAPL")
		require.NoError(t, err)
		assert.True(t, p.Quantity.Equal(dec("50")))
	})

	t.Run("list positions", func(t *testing.T) {
		positions, err := repo.ListPositions(ctx, "acc-1")
		require.NoError(t, err)
		assert.Len(t, positions, 1)
	})

	t.Run("upsert updates existing", func(t *testing.T) {
		updated := model.Position{
			ID: "pos-1", AccountID: "acc-1", Symbol: "AAPL",
			Side: model.OrderBuy, Quantity: dec("100"),
			AvgPrice: dec("149.25"), UnrealizedPnL: dec("500.00"),
			OpenedAt: pos.OpenedAt, UpdatedAt: time.Now().Unix(),
		}
		require.NoError(t, repo.UpsertPosition(ctx, updated))

		p, err := repo.GetPosition(ctx, "acc-1", "AAPL")
		require.NoError(t, err)
		assert.True(t, p.Quantity.Equal(dec("100")))
	})
}

func TestRepository_StrategyConfig(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	sc := model.StrategyConfig{
		ID: "strat-1", Name: "ma_crossover", DisplayName: "MA Crossover",
		Enabled: true, Symbols: []string{"AAPL", "MSFT"},
		Params: map[string]any{"shortPeriod": float64(5), "longPeriod": float64(20)},
		CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(),
	}
	require.NoError(t, repo.SaveStrategyConfig(ctx, sc))

	configs, err := repo.GetStrategyConfigs(ctx)
	require.NoError(t, err)
	require.Len(t, configs, 1)
	assert.Equal(t, "ma_crossover", configs[0].Name)
	assert.Equal(t, []string{"AAPL", "MSFT"}, configs[0].Symbols)
}

func TestRepository_Backtest(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	run := model.BacktestRun{
		ID:     "bt-1",
		Status: "RUNNING",
		AIMode: model.AIModeOff,
		Config: model.BacktestConfig{
			Symbol:      "AAPL",
			InitialCash: dec("100000"),
		},
		CreatedAt: time.Now().Unix(),
	}
	require.NoError(t, repo.CreateBacktestRun(ctx, run))

	t.Run("get backtest run", func(t *testing.T) {
		r, err := repo.GetBacktestRun(ctx, "bt-1")
		require.NoError(t, err)
		assert.Equal(t, "RUNNING", r.Status)
	})

	t.Run("update result", func(t *testing.T) {
		result := &model.BacktestResult{
			TotalReturn: dec("24.5"), WinRate: dec("0.62"),
			TotalTrades: 48, ProfitTrades: 30, LossTrades: 18,
		}
		require.NoError(t, repo.UpdateBacktestResult(ctx, "bt-1", result, "COMPLETED", ""))

		r, err := repo.GetBacktestRun(ctx, "bt-1")
		require.NoError(t, err)
		assert.Equal(t, "COMPLETED", r.Status)
		assert.True(t, r.Result.TotalReturn.Equal(dec("24.5")))
	})

	t.Run("list runs", func(t *testing.T) {
		runs, err := repo.ListBacktestRuns(ctx)
		require.NoError(t, err)
		assert.Len(t, runs, 1)
	})
}

func TestRepository_SentimentResults(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	sr := model.SentimentResult{
		ID: "sent-1", Text: "Apple reports record earnings",
		Score: 0.72, Label: model.SentimentPositive,
		Keywords: []string{"earnings", "record"},
		Summary:  "利好",
		Symbols:  []string{"AAPL"},
		Timestamp: time.Now().UnixMilli(),
	}
	require.NoError(t, repo.SaveSentimentResult(ctx, sr))

	results, err := repo.GetSentimentResults(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 0.72, results[0].Score)
	assert.Equal(t, []string{"AAPL"}, results[0].Symbols)
}

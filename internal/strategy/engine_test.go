package strategy

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/pkg/model"
)

func TestPositionSize(t *testing.T) {
	t.Run("normal calculation", func(t *testing.T) {
		qty := positionSize("100000", decimal.NewFromFloat(150.0), 0.2)
		expected, _ := decimal.NewFromString("133")
		assert.True(t, qty.Equal(expected), "expected %s, got %s", expected, qty)
	})

	t.Run("zero price returns zero", func(t *testing.T) {
		qty := positionSize("100000", decimal.Zero, 0.2)
		assert.True(t, qty.IsZero())
	})

	t.Run("small allocation floors to zero", func(t *testing.T) {
		qty := positionSize("100", decimal.NewFromFloat(50000.0), 0.1)
		assert.True(t, qty.IsZero())
	})
}

func TestMACrossover(t *testing.T) {
	cfg := model.StrategyConfig{
		Name:    "ma_crossover",
		Params:  map[string]any{"shortPeriod": 3, "longPeriod": 5},
		Symbols: []string{"AAPL"},
	}
	ma := NewMACrossover(cfg)

	assert.Equal(t, "ma_crossover", ma.Name())

	t.Run("golden cross produces buy", func(t *testing.T) {
		// Short MA(3) crosses above Long MA(5) at the last bar
		// prices: 10,10,10,10,10,20,20,30
		// Short(3) at idx5: (10+10+20)/3=13.3, idx6: (10+20+20)/3=16.7, idx7: (20+20+30)/3=23.3
		// Long(5) at idx5: (10+10+10+10+20)/5=12, idx6: (10+10+10+20+20)/5=14, idx7: (10+10+20+20+30)/5=18
		// Short(3) crosses above Long(5) at last bar
		// prev: short=10 <= long=10, cur: short=16.67 > long=14
		prices := []float64{10, 10, 10, 10, 10, 10, 10, 30}
		candles := makeCandles(prices)
		sig := model.Signal{Symbol: "AAPL", Source: "MA", Type: model.SignalBuy, Strength: 0.8}

		decision, err := ma.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		require.NotNil(t, decision)
		assert.Equal(t, model.OrderBuy, decision.Side)
		assert.Equal(t, "AAPL", decision.Symbol)
		assert.False(t, decision.Quantity.IsZero())
	})

	t.Run("death cross produces sell", func(t *testing.T) {
		// Short(3) crosses below Long(5) at last bar
		// prev: short=30 >= long=30, cur: short=23.33 < long=26
		prices := []float64{30, 30, 30, 30, 30, 30, 30, 10}
		candles := makeCandles(prices)
		sig := model.Signal{Symbol: "AAPL", Source: "MA", Type: model.SignalSell, Strength: 0.8}

		decision, err := ma.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		require.NotNil(t, decision)
		assert.Equal(t, model.OrderSell, decision.Side)
	})

	t.Run("insufficient data returns nil", func(t *testing.T) {
		prices := []float64{10, 11, 12}
		candles := makeCandles(prices)
		sig := model.Signal{Symbol: "AAPL", Source: "MA", Type: model.SignalBuy, Strength: 0.8}

		decision, err := ma.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		assert.Nil(t, decision)
	})
}

func TestRSIReversion(t *testing.T) {
	cfg := model.StrategyConfig{
		Name:    "rsi_reversion",
		Params:  map[string]any{"period": 14, "oversold": 30, "overbought": 70},
		Symbols: []string{"AAPL"},
	}
	rsi := NewRSIReversion(cfg)

	assert.Equal(t, "rsi_reversion", rsi.Name())

	t.Run("buy signal on oversold RSI", func(t *testing.T) {
		sig := model.Signal{Symbol: "AAPL", Source: "RSI", Type: model.SignalBuy, Strength: 0.9}
		candles := makeCandles([]float64{150, 151, 152})

		decision, err := rsi.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		require.NotNil(t, decision)
		assert.Equal(t, model.OrderBuy, decision.Side)
	})

	t.Run("sell signal on overbought RSI", func(t *testing.T) {
		sig := model.Signal{Symbol: "AAPL", Source: "RSI", Type: model.SignalSell, Strength: 0.9}
		candles := makeCandles([]float64{150, 151, 152})

		decision, err := rsi.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		require.NotNil(t, decision)
		assert.Equal(t, model.OrderSell, decision.Side)
	})

	t.Run("ignores non-RSI signals", func(t *testing.T) {
		sig := model.Signal{Symbol: "AAPL", Source: "MA", Type: model.SignalBuy, Strength: 0.9}
		candles := makeCandles([]float64{150, 151, 152})

		decision, err := rsi.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		assert.Nil(t, decision)
	})

	t.Run("ignores zero strength", func(t *testing.T) {
		sig := model.Signal{Symbol: "AAPL", Source: "RSI", Type: model.SignalBuy, Strength: 0}
		candles := makeCandles([]float64{150})

		decision, err := rsi.OnSignal(t.Context(), sig, candles)
		require.NoError(t, err)
		assert.Nil(t, decision)
	})

	t.Run("empty candles returns nil", func(t *testing.T) {
		sig := model.Signal{Symbol: "AAPL", Source: "RSI", Type: model.SignalBuy, Strength: 0.9}

		decision, err := rsi.OnSignal(t.Context(), sig, []model.Candle{})
		require.NoError(t, err)
		assert.Nil(t, decision)
	})
}

func TestIntParam(t *testing.T) {
	assert.Equal(t, 10, intParam(map[string]any{"x": float64(10)}, "x", 5))
	assert.Equal(t, 10, intParam(map[string]any{"x": 10}, "x", 5))
	assert.Equal(t, 10, intParam(map[string]any{"x": int64(10)}, "x", 5))
	assert.Equal(t, 5, intParam(map[string]any{}, "x", 5))
	assert.Equal(t, 5, intParam(map[string]any{"x": "bad"}, "x", 5))
}

func TestDecParam(t *testing.T) {
	assert.True(t, decParam(map[string]any{"x": 3.14}, "x", 1.0).Equal(decimal.NewFromFloat(3.14)))
	assert.True(t, decParam(map[string]any{}, "x", 1.0).Equal(decimal.NewFromFloat(1.0)))
}

func makeCandles(prices []float64) []model.Candle {
	candles := make([]model.Candle, len(prices))
	for i, p := range prices {
		d := decimal.NewFromFloat(p)
		candles[i] = model.Candle{
			Symbol: "AAPL", Open: d, High: d, Low: d, Close: d,
			Volume: decimal.NewFromInt(1000), Timestamp: int64(i) * 86400000,
		}
	}
	return candles
}

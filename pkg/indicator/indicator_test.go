package indicator

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/pkg/model"
)

func makeCandles(prices []float64) []model.Candle {
	candles := make([]model.Candle, len(prices))
	for i, p := range prices {
		d := decimal.NewFromFloat(p)
		candles[i] = model.Candle{
			Symbol: "TEST", Open: d, High: d, Low: d, Close: d,
			Volume: decimal.NewFromInt(1000), Timestamp: int64(i) * 86400000,
		}
	}
	return candles
}

func TestMA(t *testing.T) {
	t.Run("computes correct values", func(t *testing.T) {
		prices := []float64{10, 11, 12, 13, 14}
		ma := &MA{Period: 3}
		result := ma.Compute(makeCandles(prices))

		assert.Equal(t, "MA", result.Name)
		require.Len(t, result.Values, 3)
		assert.InDelta(t, 11.0, result.Values[0].Value, 0.001)
		assert.InDelta(t, 12.0, result.Values[1].Value, 0.001)
		assert.InDelta(t, 13.0, result.Values[2].Value, 0.001)
	})

	t.Run("insufficient data returns neutral", func(t *testing.T) {
		ma := &MA{Period: 10}
		result := ma.Compute(makeCandles([]float64{1, 2, 3}))
		assert.Equal(t, model.SignalNeutral, result.Signal)
		assert.Len(t, result.Values, 0)
	})

	t.Run("rising MA gives buy signal", func(t *testing.T) {
		prices := []float64{10, 10, 10, 11, 12, 13, 14, 15}
		ma := &MA{Period: 3}
		result := ma.Compute(makeCandles(prices))
		assert.Equal(t, model.SignalBuy, result.Signal)
	})

	t.Run("falling MA gives sell signal", func(t *testing.T) {
		prices := []float64{15, 15, 15, 14, 13, 12, 11, 10}
		ma := &MA{Period: 3}
		result := ma.Compute(makeCandles(prices))
		assert.Equal(t, model.SignalSell, result.Signal)
	})
}

func TestEMA(t *testing.T) {
	t.Run("computes correct values", func(t *testing.T) {
		prices := []float64{22.27, 22.19, 22.08, 22.17, 22.18, 22.13, 22.23, 22.43, 22.24, 22.29}
		ema := &EMA{Period: 5}
		result := ema.Compute(makeCandles(prices))

		assert.Equal(t, "EMA", result.Name)
		require.Len(t, result.Values, 6)

		firstEMA := (22.27 + 22.19 + 22.08 + 22.17 + 22.18) / 5
		assert.InDelta(t, firstEMA, result.Values[0].Value, 0.01)
	})

	t.Run("insufficient data returns neutral", func(t *testing.T) {
		ema := &EMA{Period: 20}
		result := ema.Compute(makeCandles([]float64{1, 2}))
		assert.Equal(t, model.SignalNeutral, result.Signal)
	})
}

func TestRSI(t *testing.T) {
	t.Run("computes RSI correctly", func(t *testing.T) {
		prices := []float64{
			44, 44.34, 44.09, 43.61, 44.33, 44.83, 45.10, 45.42,
			45.84, 46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.00,
		}
		rsi := &RSI{Period: 14, Oversold: 30, Overbought: 70}
		result := rsi.Compute(makeCandles(prices))

		assert.Equal(t, "RSI", result.Name)
		assert.NotEmpty(t, result.Values)
		lastRSI := result.Values[len(result.Values)-1].Value
		assert.InDelta(t, 70.46, lastRSI, 2.0)
	})

	t.Run("oversold gives buy signal", func(t *testing.T) {
		prices := []float64{100}
		for i := 0; i < 14; i++ {
			prices = append(prices, prices[len(prices)-1]*0.95)
		}
		rsi := &RSI{Period: 14, Oversold: 30, Overbought: 70}
		result := rsi.Compute(makeCandles(prices))
		assert.Equal(t, model.SignalBuy, result.Signal)
	})

	t.Run("insufficient data", func(t *testing.T) {
		rsi := &RSI{Period: 14}
		result := rsi.Compute(makeCandles([]float64{1, 2, 3}))
		assert.Equal(t, model.SignalNeutral, result.Signal)
	})
}

func TestMACD(t *testing.T) {
	t.Run("produces result with enough data", func(t *testing.T) {
		prices := make([]float64, 50)
		for i := range prices {
			prices[i] = 100 + float64(i)*0.5
		}
		macd := &MACD{FastPeriod: 12, SlowPeriod: 26, SignalPeriod: 9}
		result := macd.Compute(makeCandles(prices))

		assert.Equal(t, "MACD", result.Name)
		assert.NotEmpty(t, result.Values)
		// Steady uptrend: MACD stays above signal, no crossover
		assert.NotNil(t, result.Signal)
	})

	t.Run("insufficient data", func(t *testing.T) {
		macd := &MACD{FastPeriod: 12, SlowPeriod: 26, SignalPeriod: 9}
		result := macd.Compute(makeCandles([]float64{1, 2, 3}))
		assert.Equal(t, model.SignalNeutral, result.Signal)
	})
}

func TestBollinger(t *testing.T) {
	t.Run("produces result with enough data", func(t *testing.T) {
		prices := make([]float64, 30)
		for i := range prices {
			prices[i] = 100 + float64(i%10-5)
		}
		bb := &Bollinger{Period: 20, StdDev: 2}
		result := bb.Compute(makeCandles(prices))

		assert.Equal(t, "Bollinger", result.Name)
		assert.NotEmpty(t, result.Values)
	})

	t.Run("price above upper band gives sell", func(t *testing.T) {
		prices := make([]float64, 25)
		for i := 0; i < 24; i++ {
			prices[i] = 100
		}
		prices[24] = 200
		bb := &Bollinger{Period: 20, StdDev: 2}
		result := bb.Compute(makeCandles(prices))
		assert.Equal(t, model.SignalSell, result.Signal)
	})

	t.Run("price below lower band gives buy", func(t *testing.T) {
		prices := make([]float64, 25)
		for i := 0; i < 24; i++ {
			prices[i] = 100
		}
		prices[24] = 50
		bb := &Bollinger{Period: 20, StdDev: 2}
		result := bb.Compute(makeCandles(prices))
		assert.Equal(t, model.SignalBuy, result.Signal)
	})

	t.Run("insufficient data", func(t *testing.T) {
		bb := &Bollinger{Period: 20}
		result := bb.Compute(makeCandles([]float64{1, 2}))
		assert.Equal(t, model.SignalNeutral, result.Signal)
	})
}

func TestVWAP(t *testing.T) {
	t.Run("computes VWAP correctly", func(t *testing.T) {
		candles := []model.Candle{
			{Close: dec("100"), Volume: dec("1000"), High: dec("100"), Low: dec("100"), Open: dec("100")},
			{Close: dec("102"), Volume: dec("2000"), High: dec("102"), Low: dec("102"), Open: dec("102")},
			{Close: dec("104"), Volume: dec("1000"), High: dec("104"), Low: dec("104"), Open: dec("104")},
		}
		for i := range candles {
			candles[i].Timestamp = int64(i) * 86400000
		}

		vwap := &VWAP{}
		result := vwap.Compute(candles)

		assert.Equal(t, "VWAP", result.Name)
		require.NotEmpty(t, result.Values)
		assert.InDelta(t, 102.0, result.Values[2].Value, 0.01)
	})

	t.Run("empty input", func(t *testing.T) {
		vwap := &VWAP{}
		result := vwap.Compute([]model.Candle{})
		assert.Equal(t, model.SignalNeutral, result.Signal)
	})
}

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

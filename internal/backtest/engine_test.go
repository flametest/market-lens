package backtest

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/pkg/model"
)

func TestSimulator(t *testing.T) {
	t.Run("buy and sell produces trade", func(t *testing.T) {
		sim := newSimulator(dec("100000"), dec("0.001"), dec("0.01"))

		// Buy at open=100
		candle100 := candle(100, 1000)
		sim.execute(model.OrderBuy, dec("10"), candle100)
		assert.True(t, sim.hasPosition())
		assert.True(t, sim.positionQuantity().Equal(dec("10")))

		// Sell at open=110
		candle110 := candle(110, 2000)
		sim.execute(model.OrderSell, dec("10"), candle110)
		assert.False(t, sim.hasPosition())
		require.Len(t, sim.trades, 1)

		trade := sim.trades[0]
		assert.True(t, trade.PnL.GreaterThan(decimal.Zero), "should be profitable")
	})

	t.Run("buy reduces quantity when cash insufficient", func(t *testing.T) {
		sim := newSimulator(dec("100"), dec("0"), dec("0"))
		sim.execute(model.OrderBuy, dec("1000"), candle(50, 1000))
		assert.True(t, sim.positionQuantity().Equal(dec("2"))) // 100 / 50 = 2
	})

	t.Run("zero quantity buy does nothing", func(t *testing.T) {
		sim := newSimulator(dec("5"), dec("0"), dec("0"))
		sim.execute(model.OrderBuy, dec("1"), candle(100, 1000))
		assert.False(t, sim.hasPosition())
	})

	t.Run("equity curve tracks correctly", func(t *testing.T) {
		sim := newSimulator(dec("100000"), dec("0"), dec("0"))
		sim.recordEquity(candle(100, 1000))
		sim.recordEquity(candle(105, 2000))
		sim.recordEquity(candle(110, 3000))

		require.Len(t, sim.equityCurve, 3)
		assert.True(t, sim.equityCurve[0].Value.Equal(dec("100000")))
		assert.True(t, sim.equityCurve[1].Value.Equal(dec("100000")))
		assert.True(t, sim.equityCurve[2].Value.Equal(dec("100000")))
	})

	t.Run("max drawdown tracks peak decline", func(t *testing.T) {
		// Manually set equity curve points with different values
		curve := []model.EquityPoint{
			{Value: dec("100000"), Timestamp: 1},
			{Value: dec("120000"), Timestamp: 2},
			{Value: dec("90000"), Timestamp: 3},
		}
		dd := CalcMaxDrawdown(curve)
		assert.True(t, dd.GreaterThan(decimal.Zero))
		expected, _ := decimal.NewFromString("0.25") // (120000-90000)/120000
		assert.True(t, dd.Sub(expected).Abs().LessThan(dec("0.01")))
	})
}

func TestSimulatorBuildResult(t *testing.T) {
	sim := newSimulator(dec("100000"), dec("0.001"), dec("0"))
	sim.execute(model.OrderBuy, dec("10"), candle(100, 1000))
	sim.execute(model.OrderSell, dec("10"), candle(110, 2000))

	result := sim.buildResult(dec("100000"), 0, 365*24*3600*1000)

	assert.Equal(t, 1, result.TotalTrades)
	assert.Equal(t, 1, result.ProfitTrades)
	assert.Equal(t, 0, result.LossTrades)
	assert.True(t, result.TotalReturn.GreaterThan(decimal.Zero))
	assert.True(t, result.WinRate.Equal(decimal.NewFromInt(1)))
}

func TestCalcSharpe(t *testing.T) {
	t.Run("empty curve returns zero", func(t *testing.T) {
		sharpe := calcSharpe(nil)
		assert.True(t, sharpe.IsZero())
	})

	t.Run("single point returns zero", func(t *testing.T) {
		sharpe := calcSharpe([]model.EquityPoint{{Value: dec("100000")}})
		assert.True(t, sharpe.IsZero())
	})

	t.Run("steady growth gives positive sharpe", func(t *testing.T) {
		curve := make([]model.EquityPoint, 30)
		for i := range curve {
			curve[i] = model.EquityPoint{
				Value:     decimal.NewFromFloat(100000 + float64(i)*100),
				Timestamp: int64(i) * 86400000,
			}
		}
		sharpe := calcSharpe(curve)
		assert.True(t, sharpe.GreaterThan(decimal.Zero), "steady growth should have positive Sharpe")
	})
}

func TestCalcMaxDrawdown(t *testing.T) {
	t.Run("empty returns zero", func(t *testing.T) {
		dd := CalcMaxDrawdown(nil)
		assert.True(t, dd.IsZero())
	})

	t.Run("monotonic rise has zero drawdown", func(t *testing.T) {
		curve := []model.EquityPoint{
			{Value: dec("100")},
			{Value: dec("110")},
			{Value: dec("120")},
		}
		dd := CalcMaxDrawdown(curve)
		assert.True(t, dd.IsZero())
	})

	t.Run("calculates correct drawdown", func(t *testing.T) {
		curve := []model.EquityPoint{
			{Value: dec("100")},  // peak
			{Value: dec("120")},  // new peak
			{Value: dec("90")},   // drawdown: (120-90)/120 = 0.25
			{Value: dec("110")},
			{Value: dec("80")},   // drawdown: (120-80)/120 = 0.333
		}
		dd := CalcMaxDrawdown(curve)
		expected, _ := decimal.NewFromString("0.333")
		assert.True(t, dd.Sub(expected).Abs().LessThan(dec("0.01")), "expected ~0.333, got %s", dd)
	})
}

func TestGenerateSignal(t *testing.T) {
	t.Run("ma_crossover returns buy signal", func(t *testing.T) {
		prices := make([]float64, 30)
		for i := range prices {
			prices[i] = 100 + float64(i)
		}
		candles := makeCandles(prices)
		sig := generateSignal("ma_crossover", candles)
		assert.Equal(t, "MA", sig.Source)
		assert.Equal(t, model.SignalBuy, sig.Type)
	})

	t.Run("rsi_reversion returns neutral for normal data", func(t *testing.T) {
		prices := make([]float64, 30)
		for i := range prices {
			prices[i] = 100 + float64(i%3-1)*0.5
		}
		candles := makeCandles(prices)
		sig := generateSignal("rsi_reversion", candles)
		assert.Equal(t, "RSI", sig.Source)
	})

	t.Run("rsi_reversion returns buy for oversold", func(t *testing.T) {
		prices := []float64{100}
		for i := 0; i < 20; i++ {
			prices = append(prices, prices[len(prices)-1]*0.9)
		}
		candles := makeCandles(prices)
		sig := generateSignal("rsi_reversion", candles)
		assert.Equal(t, model.SignalBuy, sig.Type)
	})

	t.Run("unknown strategy returns default signal", func(t *testing.T) {
		candles := makeCandles([]float64{100, 101, 102})
		sig := generateSignal("custom", candles)
		assert.Equal(t, "custom", sig.Source)
	})
}

func TestCheckRisk(t *testing.T) {
	decision := &model.OrderDecision{
		Quantity: dec("100"),
		Price:    dec("100"), // $10000 order
	}

	t.Run("allows when no risk limits", func(t *testing.T) {
		sim := newSimulator(dec("100000"), dec("0"), dec("0"))
		assert.True(t, checkRisk(decision, model.RiskConfig{}, sim))
	})

	t.Run("rejects when order exceeds max amount", func(t *testing.T) {
		sim := newSimulator(dec("100000"), dec("0"), dec("0"))
		risk := model.RiskConfig{MaxOrderAmount: dec("5000")}
		assert.False(t, checkRisk(decision, risk, sim))
	})

	t.Run("allows when order within limit", func(t *testing.T) {
		sim := newSimulator(dec("100000"), dec("0"), dec("0"))
		risk := model.RiskConfig{MaxOrderAmount: dec("20000")}
		assert.True(t, checkRisk(decision, risk, sim))
	})
}

// helpers

func candle(price float64, ts int64) model.Candle {
	d := decimal.NewFromFloat(price)
	return model.Candle{
		Symbol: "TEST", Open: d, High: d, Low: d, Close: d,
		Volume: decimal.NewFromInt(1000), Timestamp: ts,
	}
}

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

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

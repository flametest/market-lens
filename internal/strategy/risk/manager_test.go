package risk

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/pkg/model"
)

func makeDecision(qty float64, price float64) model.OrderDecision {
	return model.OrderDecision{
		Symbol:   "AAPL",
		Side:     model.OrderBuy,
		Type:     model.OrderMarket,
		Quantity: decimal.NewFromFloat(qty),
		Price:    decimal.NewFromFloat(price),
		Strategy: "test",
	}
}

func defaultRisk() model.RiskConfig {
	return model.RiskConfig{
		MaxPosition:      decimal.NewFromFloat(50000),
		MaxOrderAmount:   decimal.NewFromFloat(20000),
		MaxDailyTrades:   5,
		MinTradeInterval: 60,
		MaxDrawdown:      0.2,
	}
}

func defaultAI() model.AIConfig {
	return model.AIConfig{
		Enabled:         true,
		RiskEnhancement: true,
		RiskThresholds: model.AIRiskThresholds{
			PanicThreshold:      -0.8,
			OverheatThreshold:   0.8,
			EventPauseThreshold: 0.9,
			PositionReduction:   0.5,
		},
	}
}

func TestMaxPositionRule(t *testing.T) {
	rule := &MaxPositionRule{}
	assert.Equal(t, "max_position", rule.Name())

	t.Run("allows order within limit", func(t *testing.T) {
		ec := EvalContext{
			Decision: makeDecision(10, 100), // $1000
			Risk:     defaultRisk(),
		}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("rejects order exceeding max position", func(t *testing.T) {
		ec := EvalContext{
			Decision: makeDecision(1000, 100), // $100000
			Risk:     defaultRisk(),
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reject", v.Action)
	})

	t.Run("skips when max position is zero", func(t *testing.T) {
		risk := defaultRisk()
		risk.MaxPosition = decimal.Zero
		ec := EvalContext{Decision: makeDecision(1000, 100), Risk: risk}
		assert.Nil(t, rule.Evaluate(ec))
	})
}

func TestMaxOrderAmountRule(t *testing.T) {
	rule := &MaxOrderAmountRule{}
	assert.Equal(t, "max_order_amount", rule.Name())

	t.Run("allows order within amount", func(t *testing.T) {
		ec := EvalContext{
			Decision: makeDecision(10, 100), // $1000
			Risk:     defaultRisk(),
		}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("rejects order exceeding max amount", func(t *testing.T) {
		ec := EvalContext{
			Decision: makeDecision(1000, 100), // $100000
			Risk:     defaultRisk(),
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reject", v.Action)
	})
}

func TestMaxDailyTradesRule(t *testing.T) {
	rule := &MaxDailyTradesRule{}
	assert.Equal(t, "max_daily_trades", rule.Name())

	t.Run("allows when under limit", func(t *testing.T) {
		ec := EvalContext{Risk: defaultRisk(), RecentOrders: []model.Order{}}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("rejects when limit reached", func(t *testing.T) {
		now := time.Now().Unix()
		orders := make([]model.Order, 5)
		for i := range orders {
			orders[i] = model.Order{CreatedAt: now - int64(i)*60}
		}
		ec := EvalContext{Risk: defaultRisk(), RecentOrders: orders}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reject", v.Action)
	})

	t.Run("skips when limit is zero", func(t *testing.T) {
		risk := defaultRisk()
		risk.MaxDailyTrades = 0
		ec := EvalContext{Risk: risk, RecentOrders: make([]model.Order, 10)}
		assert.Nil(t, rule.Evaluate(ec))
	})
}

func TestMinTradeIntervalRule(t *testing.T) {
	rule := &MinTradeIntervalRule{}
	assert.Equal(t, "min_trade_interval", rule.Name())

	t.Run("allows when no recent orders", func(t *testing.T) {
		ec := EvalContext{Risk: defaultRisk(), RecentOrders: []model.Order{}}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("allows when interval is zero", func(t *testing.T) {
		risk := defaultRisk()
		risk.MinTradeInterval = 0
		ec := EvalContext{Risk: risk}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("rejects when interval not met", func(t *testing.T) {
		ec := EvalContext{
			Risk: defaultRisk(), // 60s
			RecentOrders: []model.Order{{CreatedAt: time.Now().Unix() - 30}}, // 30s ago
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reject", v.Action)
	})
}

func TestMaxDrawdownRule(t *testing.T) {
	rule := &MaxDrawdownRule{}
	assert.Equal(t, "max_drawdown", rule.Name())

	t.Run("skips when no account", func(t *testing.T) {
		ec := EvalContext{Risk: defaultRisk()}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("skips when drawdown is zero", func(t *testing.T) {
		risk := defaultRisk()
		risk.MaxDrawdown = 0
		ec := EvalContext{Risk: risk}
		assert.Nil(t, rule.Evaluate(ec))
	})
}

func TestPanicProtectionRule(t *testing.T) {
	rule := &PanicProtectionRule{}
	assert.Equal(t, "panic_protection", rule.Name())

	t.Run("rejects during panic sentiment", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: -0.9,
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reject", v.Action)
	})

	t.Run("allows normal sentiment", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: 0.2,
		}
		assert.Nil(t, rule.Evaluate(ec))
	})

	t.Run("uses default threshold when not set", func(t *testing.T) {
		ai := model.AIConfig{RiskThresholds: model.AIRiskThresholds{}}
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        ai,
			Sentiment: -0.85, // below default -0.8
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reject", v.Action)
	})
}

func TestOverheatWarningRule(t *testing.T) {
	rule := &OverheatWarningRule{}
	assert.Equal(t, "overheat_warning", rule.Name())

	t.Run("reduces during overheat", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: 0.9,
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "reduce", v.Action)
		assert.Equal(t, 0.5, v.Factor)
	})

	t.Run("allows normal sentiment", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: 0.3,
		}
		assert.Nil(t, rule.Evaluate(ec))
	})
}

func TestEventPauseRule(t *testing.T) {
	rule := &EventPauseRule{}
	assert.Equal(t, "event_pause", rule.Name())

	t.Run("warns on extreme positive sentiment", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: 0.95,
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "warn", v.Action)
	})

	t.Run("warns on extreme negative sentiment", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: -0.95,
		}
		v := rule.Evaluate(ec)
		assert.NotNil(t, v)
		assert.Equal(t, "warn", v.Action)
	})

	t.Run("allows moderate sentiment", func(t *testing.T) {
		ec := EvalContext{
			Risk:      defaultRisk(),
			AI:        defaultAI(),
			Sentiment: 0.5,
		}
		assert.Nil(t, rule.Evaluate(ec))
	})
}

func TestManagerEvaluate(t *testing.T) {
	t.Run("reject blocks order when max position exceeded", func(t *testing.T) {
		m := NewManager(nil, nil, nil)
		decision := makeDecision(1000, 100) // $100000
		cfg := model.StrategyConfig{Risk: defaultRisk()}

		result, err := m.Evaluate(t.Context(), decision, cfg, 0)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("approves valid order", func(t *testing.T) {
		m := NewManager(nil, nil, nil)
		decision := makeDecision(10, 100) // $1000
		cfg := model.StrategyConfig{Risk: defaultRisk()}

		result, err := m.Evaluate(t.Context(), decision, cfg, 0)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "AAPL", result.Symbol)
	})

	t.Run("AI rules disabled when not enabled", func(t *testing.T) {
		m := NewManager(nil, nil, nil)
		decision := makeDecision(10, 100)
		ai := defaultAI()
		ai.Enabled = false
		cfg := model.StrategyConfig{Risk: defaultRisk(), AI: ai}

		// Sentiment is -0.9 (panic), but AI is disabled
		result, err := m.Evaluate(t.Context(), decision, cfg, -0.9)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("AI panic protection rejects order", func(t *testing.T) {
		m := NewManager(nil, nil, nil)
		decision := makeDecision(10, 100)
		cfg := model.StrategyConfig{Risk: defaultRisk(), AI: defaultAI()}

		result, err := m.Evaluate(t.Context(), decision, cfg, -0.9)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("overheat reduces quantity", func(t *testing.T) {
		m := NewManager(nil, nil, nil)
		decision := makeDecision(100, 50)
		cfg := model.StrategyConfig{Risk: defaultRisk(), AI: defaultAI()}

		result, err := m.Evaluate(t.Context(), decision, cfg, 0.9)
		assert.NoError(t, err)
		require.NotNil(t, result)
		// 100 * 0.5 = 50, floored
		expected := decimal.NewFromFloat(100).Mul(decimal.NewFromFloat(0.5)).Floor()
		assert.True(t, result.Quantity.Equal(expected), "expected %s got %s", expected, result.Quantity)
	})
}

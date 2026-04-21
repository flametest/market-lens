package analysis

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/flametest/market-lens/pkg/model"
)

func defaultFactors() model.AdjustmentFactors {
	return model.AdjustmentFactors{
		StrongPositiveBuy:  1.2,
		StrongPositiveSell: 0.8,
		WeakPositiveBuy:    1.1,
		StrongNegativeBuy:  0.8,
		StrongNegativeSell: 1.2,
		WeakNegativeSell:   1.1,
	}
}

func TestAggregator_TechOnlySignals(t *testing.T) {
	agg := NewAggregator(false, 0, defaultFactors())

	t.Run("all buy signals produce buy", func(t *testing.T) {
		signals := []model.Signal{
			{Symbol: "AAPL", Type: model.SignalBuy, Strength: 0.8, Price: decimal.NewFromFloat(150)},
			{Symbol: "AAPL", Type: model.SignalBuy, Strength: 0.6, Price: decimal.NewFromFloat(150)},
		}
		result := agg.Aggregate(signals, nil)
		assert.NotNil(t, result)
		assert.Greater(t, result.AdjustedConfidence, 0.0)
	})

	t.Run("mixed signals produce near zero", func(t *testing.T) {
		signals := []model.Signal{
			{Symbol: "AAPL", Type: model.SignalBuy, Strength: 0.5, Price: decimal.NewFromFloat(150)},
			{Symbol: "AAPL", Type: model.SignalSell, Strength: 0.5, Price: decimal.NewFromFloat(150)},
		}
		result := agg.Aggregate(signals, nil)
		assert.NotNil(t, result)
		assert.InDelta(t, 0.0, result.AdjustedConfidence, 0.01)
	})

	t.Run("no signals returns nil", func(t *testing.T) {
		result := agg.Aggregate(nil, nil)
		assert.Nil(t, result)
	})
}

func TestAggregator_WithAI(t *testing.T) {
	agg := NewAggregator(true, 0.3, defaultFactors())

	t.Run("positive sentiment boosts buy signal", func(t *testing.T) {
		signals := []model.Signal{
			{Symbol: "AAPL", Type: model.SignalBuy, Strength: 0.8, Price: decimal.NewFromFloat(150)},
		}
		sentiment := &model.SentimentResult{Score: 0.75}
		result := agg.Aggregate(signals, sentiment)

		assert.NotNil(t, result)
		assert.Greater(t, result.AdjustedConfidence, result.OriginalConfidence)
		assert.Equal(t, 1.2, result.AdjustmentFactor)
		assert.NotEmpty(t, result.AdjustmentReason)
	})

	t.Run("negative sentiment boosts sell signal", func(t *testing.T) {
		signals := []model.Signal{
			{Symbol: "AAPL", Type: model.SignalSell, Strength: 0.7, Price: decimal.NewFromFloat(150)},
		}
		sentiment := &model.SentimentResult{Score: -0.8}
		result := agg.Aggregate(signals, sentiment)

		assert.NotNil(t, result)
		assert.Equal(t, 1.2, result.AdjustmentFactor)
	})

	t.Run("neutral sentiment no adjustment", func(t *testing.T) {
		signals := []model.Signal{
			{Symbol: "AAPL", Type: model.SignalBuy, Strength: 0.6, Price: decimal.NewFromFloat(150)},
		}
		sentiment := &model.SentimentResult{Score: 0.1}
		result := agg.Aggregate(signals, sentiment)

		assert.NotNil(t, result)
		assert.Equal(t, 1.0, result.AdjustmentFactor)
	})
}

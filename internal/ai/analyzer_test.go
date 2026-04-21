package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/pkg/model"
)

func TestLocalFallback(t *testing.T) {
	analyzer := NewAnalyzer(nil, nil, nil, "", "gpt-4")

	t.Run("positive text gets positive score", func(t *testing.T) {
		result, err := analyzer.localFallback("AAPL shows bullish growth with record profit surge")
		require.NoError(t, err)
		assert.True(t, result.Score > 0, "score should be positive, got %.2f", result.Score)
		assert.Equal(t, model.SentimentPositive, result.Label)
		assert.Contains(t, result.Symbols, "AAPL")
	})

	t.Run("negative text gets negative score", func(t *testing.T) {
		result, err := analyzer.localFallback("Market crash fears rise as GOOGL declines on weak earnings")
		require.NoError(t, err)
		assert.True(t, result.Score < 0, "score should be negative, got %.2f", result.Score)
		assert.Equal(t, model.SentimentNegative, result.Label)
	})

	t.Run("neutral text gets neutral score", func(t *testing.T) {
		result, err := analyzer.localFallback("The company reported quarterly results today")
		require.NoError(t, err)
		assert.Equal(t, model.SentimentNeutral, result.Label)
	})

	t.Run("extracts stock symbols", func(t *testing.T) {
		result, err := analyzer.localFallback("AAPL and MSFT both reported earnings")
		require.NoError(t, err)
		assert.Contains(t, result.Symbols, "AAPL")
		assert.Contains(t, result.Symbols, "MSFT")
	})

	t.Run("extracts keywords", func(t *testing.T) {
		result, err := analyzer.localFallback("Revenue growth beat expectations for the stock market")
		require.NoError(t, err)
		assert.Contains(t, result.Keywords, "revenue")
		assert.Contains(t, result.Keywords, "growth")
		assert.Contains(t, result.Keywords, "stock")
		assert.Contains(t, result.Keywords, "market")
	})

	t.Run("score clamped to [-1, 1]", func(t *testing.T) {
		result, err := analyzer.localFallback(
			"bullish surge rally growth profit gain upgrade beat strong optimistic rise higher record outperform bullish surge",
		)
		require.NoError(t, err)
		assert.LessOrEqual(t, result.Score, 1.0)
		assert.GreaterOrEqual(t, result.Score, -1.0)
	})

	t.Run("summary truncates long text", func(t *testing.T) {
		longText := make([]byte, 200)
		for i := range longText {
			longText[i] = 'a'
		}
		result, err := analyzer.localFallback(string(longText))
		require.NoError(t, err)
		assert.Len(t, result.Summary, 100)
	})
}

func TestParseResponse(t *testing.T) {
	analyzer := NewAnalyzer(nil, nil, nil, "", "gpt-4")

	t.Run("parses clean JSON", func(t *testing.T) {
		content := `{"score": 0.7, "label": "POSITIVE", "keywords": ["earnings"], "summary": "Good quarter", "symbols": ["AAPL"]}`
		result, err := analyzer.parseResponse(content, "test text")
		require.NoError(t, err)
		assert.InDelta(t, 0.7, result.Score, 0.01)
		assert.Equal(t, model.SentimentPositive, result.Label)
		assert.Contains(t, result.Keywords, "earnings")
		assert.Contains(t, result.Symbols, "AAPL")
	})

	t.Run("parses markdown-wrapped JSON", func(t *testing.T) {
		content := "```json\n{\"score\": -0.5, \"label\": \"NEGATIVE\", \"keywords\": [], \"summary\": \"Bad news\", \"symbols\": []}\n```"
		result, err := analyzer.parseResponse(content, "test text")
		require.NoError(t, err)
		assert.InDelta(t, -0.5, result.Score, 0.01)
		assert.Equal(t, model.SentimentNegative, result.Label)
	})

	t.Run("infers label from score when missing", func(t *testing.T) {
		content := `{"score": 0.8, "keywords": [], "summary": "Great", "symbols": []}`
		result, err := analyzer.parseResponse(content, "test text")
		require.NoError(t, err)
		assert.Equal(t, model.SentimentPositive, result.Label)
	})
}

func TestLabelFromScore(t *testing.T) {
	tests := []struct {
		score float64
		want  model.SentimentLabel
	}{
		{0.5, model.SentimentPositive},
		{0.21, model.SentimentPositive},
		{0.0, model.SentimentNeutral},
		{0.1, model.SentimentNeutral},
		{-0.1, model.SentimentNeutral},
		{-0.3, model.SentimentNegative},
		{-0.8, model.SentimentNegative},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.want, labelFromScore(tt.score))
		})
	}
}

func TestSentimentToSignal(t *testing.T) {
	t.Run("strong positive generates buy signal", func(t *testing.T) {
		sr := model.SentimentResult{
			ID:      "test1",
			Score:   0.8,
			Label:   model.SentimentPositive,
			Symbols: []string{"AAPL"},
		}
		sig := SentimentToSignal(sr)
		assert.Equal(t, model.SignalBuy, sig.Type)
		assert.Equal(t, "AAPL", sig.Symbol)
		assert.Equal(t, "AI", sig.Source)
		assert.Equal(t, 0.8, sig.Strength)
	})

	t.Run("strong negative generates sell signal", func(t *testing.T) {
		sr := model.SentimentResult{
			ID:      "test2",
			Score:   -0.7,
			Label:   model.SentimentNegative,
			Symbols: []string{"GOOGL"},
		}
		sig := SentimentToSignal(sr)
		assert.Equal(t, model.SignalSell, sig.Type)
		assert.Equal(t, "GOOGL", sig.Symbol)
	})

	t.Run("weak positive generates weak buy", func(t *testing.T) {
		sr := model.SentimentResult{
			ID:    "test3",
			Score: 0.3,
			Label: model.SentimentPositive,
		}
		sig := SentimentToSignal(sr)
		assert.Equal(t, model.SignalBuy, sig.Type)
		assert.Less(t, sig.Strength, 0.3)
	})

	t.Run("neutral score gives neutral signal", func(t *testing.T) {
		sr := model.SentimentResult{
			ID:    "test4",
			Score: 0.05,
			Label: model.SentimentNeutral,
		}
		sig := SentimentToSignal(sr)
		assert.Equal(t, model.SignalNeutral, sig.Type)
	})

	t.Run("no symbols gives empty symbol", func(t *testing.T) {
		sr := model.SentimentResult{Score: 0.6}
		sig := SentimentToSignal(sr)
		assert.Equal(t, "", sig.Symbol)
	})
}

func TestAnalyzeWithLocalFallback(t *testing.T) {
	analyzer := NewAnalyzer(nil, nil, nil, "", "gpt-4")

	result, err := analyzer.Analyze(t.Context(), "AAPL rallies on strong earnings beat")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Score > 0)
	assert.Equal(t, model.SentimentPositive, result.Label)
}

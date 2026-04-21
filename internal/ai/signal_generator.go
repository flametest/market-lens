package ai

import (
	"github.com/flametest/market-lens/pkg/model"
)

// SentimentToSignal converts a sentiment result into a trading signal.
func SentimentToSignal(sr model.SentimentResult) model.Signal {
	sigType := model.SignalNeutral
	strength := 0.0

	switch {
	case sr.Score > 0.5:
		sigType = model.SignalBuy
		strength = sr.Score
	case sr.Score > 0.2:
		sigType = model.SignalBuy
		strength = sr.Score * 0.5
	case sr.Score < -0.5:
		sigType = model.SignalSell
		strength = -sr.Score
	case sr.Score < -0.2:
		sigType = model.SignalSell
		strength = -sr.Score * 0.5
	}

	symbol := ""
	if len(sr.Symbols) > 0 {
		symbol = sr.Symbols[0]
	}

	return model.Signal{
		ID:        sr.ID,
		Symbol:    symbol,
		Source:    "AI",
		Type:      sigType,
		Strength:  strength,
		Timestamp: sr.Timestamp,
		Metadata: map[string]any{
			"sentimentScore": sr.Score,
			"sentimentLabel": string(sr.Label),
			"summary":        sr.Summary,
		},
	}
}

package analysis

import (
	"github.com/flametest/market-lens/pkg/model"
)

type Aggregator struct {
	aiEnabled     bool
	aiWeight      float64
	adjustFactors model.AdjustmentFactors
}

func NewAggregator(aiEnabled bool, aiWeight float64, factors model.AdjustmentFactors) *Aggregator {
	return &Aggregator{
		aiEnabled:     aiEnabled,
		aiWeight:      aiWeight,
		adjustFactors: factors,
	}
}

func (a *Aggregator) Aggregate(signals []model.Signal, sentiment *model.SentimentResult) *model.AggregatedSignal {
	if len(signals) == 0 && sentiment == nil {
		return nil
	}

	techScore := a.aggregateTechSignals(signals)
	adjustedConfidence := techScore.confidence
	adjustmentFactor := 1.0
	sentimentScore := 0.0
	reason := ""

	if a.aiEnabled && sentiment != nil {
		sentimentScore = sentiment.Score
		aiSignalScore := sentimentScore

		combinedConfidence := techScore.confidence*(1-a.aiWeight) + aiSignalScore*a.aiWeight
		adjustedConfidence = combinedConfidence

		adjustmentFactor = a.getAdjustmentFactor(sentimentScore, techScore.direction)
		adjustedConfidence *= adjustmentFactor

		if adjustmentFactor != 1.0 {
			reason = a.getAdjustmentReason(sentimentScore)
		}
	}

	bestSignal := model.Signal{Symbol: signals[0].Symbol}
	if len(signals) > 0 {
		bestSignal = signals[0]
	}

	return &model.AggregatedSignal{
		Signal:             bestSignal,
		OriginalConfidence: techScore.confidence,
		AdjustedConfidence: adjustedConfidence,
		AdjustmentFactor:   adjustmentFactor,
		SentimentScore:     sentimentScore,
		AdjustmentReason:   reason,
	}
}

type techResult struct {
	confidence float64
	direction  model.SignalType
}

func (a *Aggregator) aggregateTechSignals(signals []model.Signal) techResult {
	if len(signals) == 0 {
		return techResult{confidence: 0, direction: model.SignalNeutral}
	}

	var buyScore, sellScore, totalWeight float64
	for _, s := range signals {
		w := s.Strength
		totalWeight += w
		if s.Type == model.SignalBuy {
			buyScore += w
		} else if s.Type == model.SignalSell {
			sellScore += w
		}
	}
	if totalWeight == 0 {
		return techResult{confidence: 0, direction: model.SignalNeutral}
	}

	netScore := (buyScore - sellScore) / totalWeight
	direction := model.SignalNeutral
	if netScore > 0 {
		direction = model.SignalBuy
	} else if netScore < 0 {
		direction = model.SignalSell
	}

	return techResult{confidence: netScore, direction: direction}
}

func (a *Aggregator) getAdjustmentFactor(sentimentScore float64, direction model.SignalType) float64 {
	f := a.adjustFactors
	if sentimentScore > 0.7 {
		if direction == model.SignalBuy {
			return f.StrongPositiveBuy
		}
		return f.StrongPositiveSell
	}
	if sentimentScore > 0.3 {
		if direction == model.SignalBuy {
			return f.WeakPositiveBuy
		}
	}
	if sentimentScore < -0.7 {
		if direction == model.SignalBuy {
			return f.StrongNegativeBuy
		}
		return f.StrongNegativeSell
	}
	if sentimentScore < -0.3 {
		if direction == model.SignalSell {
			return f.WeakNegativeSell
		}
	}
	return 1.0
}

func (a *Aggregator) getAdjustmentReason(score float64) string {
	if score > 0.7 {
		return "强利好情绪增强买入信号"
	}
	if score > 0.3 {
		return "弱利好情绪"
	}
	if score < -0.7 {
		return "强利空情绪增强卖出信号"
	}
	if score < -0.3 {
		return "弱利空情绪"
	}
	return ""
}

package indicator

import (
	"math"

	"github.com/flametest/market-lens/pkg/model"
)

type Bollinger struct {
	Period  int
	StdDev  float64
}

func (b *Bollinger) Name() string { return "Bollinger" }

func (b *Bollinger) Compute(candles []model.Candle) Result {
	if len(candles) < b.Period {
		return Result{Name: b.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	values := make([]Value, 0, len(candles)-b.Period+1)
	for i := b.Period - 1; i < len(candles); i++ {
		var sum float64
		for j := i - b.Period + 1; j <= i; j++ {
			sum += candles[j].Close.InexactFloat64()
		}
		mean := sum / float64(b.Period)

		var sqSum float64
		for j := i - b.Period + 1; j <= i; j++ {
			diff := candles[j].Close.InexactFloat64() - mean
			sqSum += diff * diff
		}
		std := math.Sqrt(sqSum / float64(b.Period))

		values = append(values, Value{
			Timestamp: candles[i].Timestamp,
			Value:     mean,
		})

		_ = std
	}

	if len(values) == 0 || len(candles) < b.Period {
		return Result{Name: b.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	lastIdx := len(candles) - 1
	startIdx := lastIdx - b.Period + 1
	var sum float64
	for j := startIdx; j <= lastIdx; j++ {
		sum += candles[j].Close.InexactFloat64()
	}
	mean := sum / float64(b.Period)

	var sqSum float64
	for j := startIdx; j <= lastIdx; j++ {
		diff := candles[j].Close.InexactFloat64() - mean
		sqSum += diff * diff
	}
	std := math.Sqrt(sqSum / float64(b.Period))

	upper := mean + b.StdDev*std
	lower := mean - b.StdDev*std
	price := candles[lastIdx].Close.InexactFloat64()

	if price <= lower {
		return Result{Name: b.Name(), Values: values, Signal: model.SignalBuy,
			Strength: (lower - price) / lower}
	}
	if price >= upper {
		return Result{Name: b.Name(), Values: values, Signal: model.SignalSell,
			Strength: (price - upper) / upper}
	}
	return Result{Name: b.Name(), Values: values, Signal: model.SignalNeutral, Strength: 0}
}

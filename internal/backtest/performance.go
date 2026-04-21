package backtest

import (
	"math"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/pkg/model"
)

func calcSharpe(curve []model.EquityPoint) decimal.Decimal {
	if len(curve) < 2 {
		return decimal.Zero
	}

	returns := make([]float64, len(curve)-1)
	for i := 1; i < len(curve); i++ {
		if curve[i-1].Value.IsZero() {
			returns[i-1] = 0
			continue
		}
		r, _ := curve[i].Value.Sub(curve[i-1].Value).Div(curve[i-1].Value).Float64()
		returns[i-1] = r
	}

	mean := avg(returns)
	stdDev := std(returns)

	if stdDev == 0 {
		return decimal.Zero
	}

	// Annualized Sharpe (252 trading days)
	sharpe := mean / stdDev * math.Sqrt(252)
	return decimal.NewFromFloat(sharpe)
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func std(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := avg(values)
	sum := 0.0
	for _, v := range values {
		d := v - m
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(values)-1))
}

// CalcMaxDrawdown computes the max drawdown from an equity curve.
func CalcMaxDrawdown(curve []model.EquityPoint) decimal.Decimal {
	if len(curve) == 0 {
		return decimal.Zero
	}

	peak := curve[0].Value
	maxDD := decimal.Zero

	for _, pt := range curve {
		if pt.Value.GreaterThan(peak) {
			peak = pt.Value
		}
		if peak.IsZero() {
			continue
		}
		dd := decimal.NewFromInt(1).Sub(pt.Value.Div(peak))
		if dd.GreaterThan(maxDD) {
			maxDD = dd
		}
	}
	return maxDD
}

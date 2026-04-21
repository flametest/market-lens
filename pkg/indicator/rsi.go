package indicator

import (
	"math"

	"github.com/flametest/market-lens/pkg/model"
)

type RSI struct {
	Period     int
	Oversold   float64
	Overbought float64
}

func (r *RSI) Name() string { return "RSI" }

func (r *RSI) Compute(candles []model.Candle) Result {
	if len(candles) < r.Period+1 {
		return Result{Name: r.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	values := make([]Value, 0, len(candles)-r.Period)

	var avgGain, avgLoss float64
	for i := 1; i <= r.Period; i++ {
		change := candles[i].Close.Sub(candles[i-1].Close).InexactFloat64()
		if change > 0 {
			avgGain += change
		} else {
			avgLoss += math.Abs(change)
		}
	}
	avgGain /= float64(r.Period)
	avgLoss /= float64(r.Period)

	rs := 0.0
	if avgLoss != 0 {
		rs = avgGain / avgLoss
	}
	rsi := 100 - 100/(1+rs)
	values = append(values, Value{Timestamp: candles[r.Period].Timestamp, Value: rsi})

	for i := r.Period + 1; i < len(candles); i++ {
		change := candles[i].Close.Sub(candles[i-1].Close).InexactFloat64()
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else {
			loss = math.Abs(change)
		}
		avgGain = (avgGain*float64(r.Period-1) + gain) / float64(r.Period)
		avgLoss = (avgLoss*float64(r.Period-1) + loss) / float64(r.Period)

		if avgLoss == 0 {
			rsi = 100
		} else {
			rs = avgGain / avgLoss
			rsi = 100 - 100/(1+rs)
		}
		values = append(values, Value{Timestamp: candles[i].Timestamp, Value: rsi})
	}

	sig, str := rsiSignal(values, r.Oversold, r.Overbought)
	return Result{Name: r.Name(), Values: values, Signal: sig, Strength: str}
}

func rsiSignal(values []Value, oversold, overbought float64) (model.SignalType, float64) {
	if len(values) == 0 {
		return model.SignalNeutral, 0
	}
	cur := values[len(values)-1].Value
	if cur <= oversold {
		return model.SignalBuy, (oversold - cur) / oversold
	}
	if cur >= overbought {
		return model.SignalSell, (cur - overbought) / (100 - overbought)
	}
	return model.SignalNeutral, 0
}

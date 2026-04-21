package indicator

import (
	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/pkg/model"
)

type MA struct {
	Period int
}

func (m *MA) Name() string { return "MA" }

func (m *MA) Compute(candles []model.Candle) Result {
	if len(candles) < m.Period {
		return Result{Name: m.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	values := make([]Value, 0, len(candles)-m.Period+1)
	for i := m.Period - 1; i < len(candles); i++ {
		var sum decimal.Decimal
		for j := i - m.Period + 1; j <= i; j++ {
			sum = sum.Add(candles[j].Close)
		}
		values = append(values, Value{
			Timestamp: candles[i].Timestamp,
			Value:     sum.Div(decimal.NewFromInt(int64(m.Period))).InexactFloat64(),
		})
	}

	sig, str := maSignal(values)
	return Result{Name: m.Name(), Values: values, Signal: sig, Strength: str}
}

type EMA struct {
	Period int
}

func (e *EMA) Name() string { return "EMA" }

func (e *EMA) Compute(candles []model.Candle) Result {
	if len(candles) < e.Period {
		return Result{Name: e.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	k := 2.0 / float64(e.Period+1)
	values := make([]Value, 0, len(candles)-e.Period+1)

	var prevEMA float64
	for i := 0; i < e.Period; i++ {
		prevEMA += candles[i].Close.InexactFloat64()
	}
	prevEMA /= float64(e.Period)
	values = append(values, Value{Timestamp: candles[e.Period-1].Timestamp, Value: prevEMA})

	for i := e.Period; i < len(candles); i++ {
		price := candles[i].Close.InexactFloat64()
		ema := price*k + prevEMA*(1-k)
		values = append(values, Value{Timestamp: candles[i].Timestamp, Value: ema})
		prevEMA = ema
	}

	sig, str := maSignal(values)
	return Result{Name: e.Name(), Values: values, Signal: sig, Strength: str}
}

func maSignal(values []Value) (model.SignalType, float64) {
	if len(values) < 2 {
		return model.SignalNeutral, 0
	}
	cur := values[len(values)-1].Value
	prev := values[len(values)-2].Value
	if cur > prev {
		return model.SignalBuy, (cur - prev) / cur
	}
	if cur < prev {
		return model.SignalSell, (prev - cur) / cur
	}
	return model.SignalNeutral, 0
}

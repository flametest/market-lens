package indicator

import (
	"github.com/flametest/market-lens/pkg/model"
)

type MACD struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
}

func (m *MACD) Name() string { return "MACD" }

func (m *MACD) Compute(candles []model.Candle) Result {
	if len(candles) < m.SlowPeriod+m.SignalPeriod {
		return Result{Name: m.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	fastEMA := computeEMAValues(candles, m.FastPeriod)
	slowEMA := computeEMAValues(candles, m.SlowPeriod)
	if len(fastEMA) == 0 || len(slowEMA) == 0 {
		return Result{Name: m.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	minLen := len(fastEMA)
	if len(slowEMA) < minLen {
		minLen = len(slowEMA)
	}

	macdLine := make([]Value, 0, minLen)
	offset := len(fastEMA) - len(slowEMA)
	if offset < 0 {
		offset = 0
	}
	for i := 0; i < minLen && i+offset < len(fastEMA); i++ {
		macdVal := fastEMA[i+offset].Value - slowEMA[i].Value
		macdLine = append(macdLine, Value{Timestamp: slowEMA[i].Timestamp, Value: macdVal})
	}

	signalLine := emaOfValues(macdLine, m.SignalPeriod)
	if len(signalLine) < 2 {
		return Result{Name: m.Name(), Values: macdLine, Signal: model.SignalNeutral, Strength: 0}
	}

	curMACD := macdLine[len(macdLine)-1].Value
	curSignal := signalLine[len(signalLine)-1].Value
	prevMACD := macdLine[len(macdLine)-2].Value
	prevSignal := signalLine[len(signalLine)-2].Value

	if prevMACD <= prevSignal && curMACD > curSignal {
		strength := (curMACD - curSignal)
		if strength < 0 {
			strength = -strength
		}
		return Result{Name: m.Name(), Values: macdLine, Signal: model.SignalBuy, Strength: strength}
	}
	if prevMACD >= prevSignal && curMACD < curSignal {
		strength := (curSignal - curMACD)
		if strength < 0 {
			strength = -strength
		}
		return Result{Name: m.Name(), Values: macdLine, Signal: model.SignalSell, Strength: strength}
	}
	return Result{Name: m.Name(), Values: macdLine, Signal: model.SignalNeutral, Strength: 0}
}

func computeEMAValues(candles []model.Candle, period int) []Value {
	if len(candles) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	values := make([]Value, 0, len(candles)-period+1)

	var prev float64
	for i := 0; i < period; i++ {
		prev += candles[i].Close.InexactFloat64()
	}
	prev /= float64(period)
	values = append(values, Value{Timestamp: candles[period-1].Timestamp, Value: prev})

	for i := period; i < len(candles); i++ {
		ema := candles[i].Close.InexactFloat64()*k + prev*(1-k)
		values = append(values, Value{Timestamp: candles[i].Timestamp, Value: ema})
		prev = ema
	}
	return values
}

func emaOfValues(input []Value, period int) []Value {
	if len(input) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	result := make([]Value, 0, len(input)-period+1)

	var prev float64
	for i := 0; i < period; i++ {
		prev += input[i].Value
	}
	prev /= float64(period)
	result = append(result, Value{Timestamp: input[period-1].Timestamp, Value: prev})

	for i := period; i < len(input); i++ {
		ema := input[i].Value*k + prev*(1-k)
		result = append(result, Value{Timestamp: input[i].Timestamp, Value: ema})
		prev = ema
	}
	return result
}

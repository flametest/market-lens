package indicator

import (
	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/pkg/model"
)

type VWAP struct{}

func (v *VWAP) Name() string { return "VWAP" }

func (v *VWAP) Compute(candles []model.Candle) Result {
	if len(candles) == 0 {
		return Result{Name: v.Name(), Signal: model.SignalNeutral, Strength: 0}
	}

	values := make([]Value, 0, len(candles))
	var cumVP decimal.Decimal
	var cumVol decimal.Decimal

	for _, c := range candles {
		typical := c.Open.Add(c.High).Add(c.Low).Add(c.Close).Div(decimal.NewFromInt(4))
		vp := typical.Mul(c.Volume)
		cumVP = cumVP.Add(vp)
		cumVol = cumVol.Add(c.Volume)

		if cumVol.IsZero() {
			values = append(values, Value{Timestamp: c.Timestamp, Value: 0})
			continue
		}
		vwap := cumVP.Div(cumVol)
		values = append(values, Value{
			Timestamp: c.Timestamp,
			Value:     vwap.InexactFloat64(),
		})
	}

	if len(values) < 2 {
		return Result{Name: v.Name(), Values: values, Signal: model.SignalNeutral, Strength: 0}
	}

	curVWAP := values[len(values)-1].Value
	curPrice := candles[len(candles)-1].Close.InexactFloat64()

	if curPrice > curVWAP {
		return Result{Name: v.Name(), Values: values, Signal: model.SignalBuy,
			Strength: (curPrice - curVWAP) / curPrice}
	}
	if curPrice < curVWAP {
		return Result{Name: v.Name(), Values: values, Signal: model.SignalSell,
			Strength: (curVWAP - curPrice) / curVWAP}
	}
	return Result{Name: v.Name(), Values: values, Signal: model.SignalNeutral, Strength: 0}
}

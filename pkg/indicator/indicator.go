package indicator

import (
	"github.com/flametest/market-lens/pkg/model"
)

type Indicator interface {
	Name() string
	Compute(candles []model.Candle) Result
}

type Result struct {
	Name     string
	Values   []Value
	Signal   model.SignalType
	Strength float64
}

type Value struct {
	Timestamp int64
	Value     float64
}

package exchange

import (
	"context"

	"github.com/flametest/market-lens/pkg/model"
)

type DataProvider interface {
	Name() string
	Connect(ctx context.Context) error
	Subscribe(symbols []string) error
	Unsubscribe(symbols []string) error
	FetchCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error)
	FetchQuote(ctx context.Context, symbol string) (*model.Tick, error)
	SearchSymbols(ctx context.Context, query string) ([]model.SymbolInfo, error)
	SetOnTick(fn func(model.Tick))
	Close() error
}

package finnhub

import (
	"context"
	"log/slog"

	"github.com/flametest/market-lens/pkg/exchange"
	"github.com/flametest/market-lens/pkg/model"
)

type Provider struct {
	ws   *WSClient
	rest *RESTClient
}

func NewProvider(wsURL, baseURL, apiKey string, logger *slog.Logger) *Provider {
	return &Provider{
		ws:   NewWSClient(wsURL, apiKey, logger),
		rest: NewRESTClient(baseURL, apiKey, logger),
	}
}

func (p *Provider) Name() string { return "finnhub" }

func (p *Provider) Connect(ctx context.Context) error {
	return p.ws.Connect(ctx)
}

func (p *Provider) Subscribe(symbols []string) error {
	return p.ws.Subscribe(symbols)
}

func (p *Provider) Unsubscribe(symbols []string) error {
	return p.ws.Unsubscribe(symbols)
}

func (p *Provider) FetchCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error) {
	return p.rest.FetchCandles(ctx, symbol, interval, start/1000, end/1000)
}

func (p *Provider) FetchQuote(ctx context.Context, symbol string) (*model.Tick, error) {
	return p.rest.FetchQuote(ctx, symbol)
}

func (p *Provider) SearchSymbols(ctx context.Context, query string) ([]model.SymbolInfo, error) {
	return p.rest.SearchSymbols(ctx, query)
}

func (p *Provider) SetOnTick(fn func(model.Tick)) {
	p.ws.SetOnTick(fn)
}

func (p *Provider) Close() error {
	return p.ws.Close()
}

var _ exchange.DataProvider = (*Provider)(nil)

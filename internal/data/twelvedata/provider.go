package twelvedata

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/flametest/market-lens/pkg/exchange"
	"github.com/flametest/market-lens/pkg/model"
)

type Provider struct {
	rest    *RESTClient
	mu      sync.Mutex
	symbols []string
	onTick  func(model.Tick)
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewProvider(baseURL, apiKey string, logger *slog.Logger) *Provider {
	return &Provider{
		rest:   NewRESTClient(baseURL, apiKey, logger),
		logger: logger,
	}
}

func (p *Provider) Name() string { return "twelvedata" }

func (p *Provider) Connect(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)
	return nil
}

func (p *Provider) Subscribe(symbols []string) error {
	p.mu.Lock()
	p.symbols = append(p.symbols, symbols...)
	p.mu.Unlock()

	if p.onTick != nil {
		go p.pollLoop()
	}
	return nil
}

func (p *Provider) Unsubscribe(symbols []string) error {
	p.mu.Lock()
	filtered := make([]string, 0, len(p.symbols))
	unsub := make(map[string]bool, len(symbols))
	for _, s := range symbols {
		unsub[s] = true
	}
	for _, s := range p.symbols {
		if !unsub[s] {
			filtered = append(filtered, s)
		}
	}
	p.symbols = filtered
	p.mu.Unlock()
	return nil
}

func (p *Provider) FetchCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error) {
	return p.rest.FetchCandles(ctx, symbol, interval, start, end)
}

func (p *Provider) FetchQuote(ctx context.Context, symbol string) (*model.Tick, error) {
	return p.rest.FetchQuote(ctx, symbol)
}

func (p *Provider) SearchSymbols(ctx context.Context, query string) ([]model.SymbolInfo, error) {
	return p.rest.SearchSymbols(ctx, query)
}

func (p *Provider) SetOnTick(fn func(model.Tick)) {
	p.onTick = fn
}

func (p *Provider) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *Provider) pollLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Initial fetch
	p.pollOnce()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce()
		}
	}
}

func (p *Provider) pollOnce() {
	p.mu.Lock()
	symbols := make([]string, len(p.symbols))
	copy(symbols, p.symbols)
	onTick := p.onTick
	p.mu.Unlock()

	if onTick == nil || len(symbols) == 0 {
		return
	}

	for _, symbol := range symbols {
		tick, err := p.rest.FetchQuote(p.ctx, symbol)
		if err != nil {
			p.logger.Debug("poll quote failed",
				slog.String("symbol", symbol),
				slog.String("error", err.Error()),
			)
			continue
		}
		onTick(*tick)
	}
}

var _ exchange.DataProvider = (*Provider)(nil)

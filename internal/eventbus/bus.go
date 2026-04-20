package eventbus

import (
	"log/slog"
	"sync"

	"github.com/flametest/market-lens/pkg/model"
)

type subscriber struct {
	ch chan any
	fn func(any)
}

type InMemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
	logger      *slog.Logger
	closed      bool
}

func NewInMemoryBus(logger *slog.Logger) *InMemoryBus {
	return &InMemoryBus{
		subscribers: make(map[string][]*subscriber),
		logger:      logger,
	}
}

func (b *InMemoryBus) Publish(topic string, data any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	subs, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for _, s := range subs {
		if s.fn != nil {
			go s.fn(data)
			continue
		}
		select {
		case s.ch <- data:
		default:
			b.logger.Warn("subscriber channel full, dropping event",
				slog.String("topic", topic),
			)
		}
	}
}

func (b *InMemoryBus) Subscribe(topic string) <-chan any {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan any, 256)
	b.subscribers[topic] = append(b.subscribers[topic], &subscriber{ch: ch})
	return ch
}

func (b *InMemoryBus) SubscribeFunc(topic string, fn func(any)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], &subscriber{fn: fn})
}

func (b *InMemoryBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	for _, subs := range b.subscribers {
		for _, s := range subs {
			if s.ch != nil {
				close(s.ch)
			}
		}
	}
	b.subscribers = make(map[string][]*subscriber)
}

func TopicsForSymbol(symbol string) []string {
	return []string{
		model.EventMarketTick + ":" + symbol,
		model.EventAnalysisSignal + ":" + symbol,
		model.EventSignalAggregated + ":" + symbol,
	}
}

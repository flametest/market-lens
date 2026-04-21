package api

import (
	"log/slog"

	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/model"
)

// Bridge subscribes to EventBus topics and forwards events to WebSocket clients.
type Bridge struct {
	hub    *Hub
	bus    *eventbus.InMemoryBus
	logger *slog.Logger
}

func NewBridge(hub *Hub, bus *eventbus.InMemoryBus, logger *slog.Logger) *Bridge {
	return &Bridge{hub: hub, bus: bus, logger: logger}
}

func (b *Bridge) Start() {
	topics := map[string]string{
		model.EventMarketTick:        "tick",
		model.EventMarketCandle:      "candle",
		model.EventAnalysisSignal:    "signal",
		model.EventAnalysisSentiment: "sentiment",
		model.EventStrategyDecision:  "strategy_decision",
		model.EventRiskApproved:      "risk_approved",
		model.EventRiskRejected:      "risk_rejected",
		model.EventRiskAIWarning:     "risk_warning",
		model.EventExecutionFill:     "fill",
		model.EventExecutionReject:   "execution_reject",
	}

	for topic, msgType := range topics {
		msgType := msgType
		b.bus.SubscribeFunc(topic, func(ev any) {
			b.hub.Broadcast(msgType, ev)
		})
	}

	if b.logger != nil {
		b.logger.Info("ws bridge started", slog.Int("topics", len(topics)))
	}
}

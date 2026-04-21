package risk

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/model"
)

type Violation struct {
	Rule    string
	Message string
	Action  string  // "reject", "reduce", "warn"
	Factor  float64 // for reduce actions
}

type EvalContext struct {
	Decision  model.OrderDecision
	Risk      model.RiskConfig
	AI        model.AIConfig
	Sentiment float64

	Account     *model.Account
	Positions   []model.Position
	RecentOrders []model.Order
}

type Rule interface {
	Name() string
	Evaluate(ec EvalContext) *Violation
}

type Manager struct {
	repo    *data.Repository
	bus     *eventbus.InMemoryBus
	logger  *slog.Logger
	rules   []Rule
	aiRules []Rule
}

func NewManager(repo *data.Repository, bus *eventbus.InMemoryBus, logger *slog.Logger) *Manager {
	return &Manager{
		repo:   repo,
		bus:    bus,
		logger: logger,
		rules: []Rule{
			&MaxPositionRule{},
			&MaxOrderAmountRule{},
			&MaxDailyTradesRule{},
			&MinTradeIntervalRule{},
			&MaxDrawdownRule{},
		},
		aiRules: []Rule{
			&PanicProtectionRule{},
			&OverheatWarningRule{},
			&EventPauseRule{},
		},
	}
}

func (m *Manager) Evaluate(ctx context.Context, decision model.OrderDecision, cfg model.StrategyConfig, sentimentScore float64) (*model.OrderDecision, error) {
	ec := EvalContext{
		Decision:  decision,
		Risk:      cfg.Risk,
		AI:        cfg.AI,
		Sentiment: sentimentScore,
	}

	// Gather state for rules that need it
	if m.repo != nil {
		orders, _ := m.repo.GetOrders(ctx, "", 200)
		ec.RecentOrders = orders
	}

	var violations []Violation

	for _, rule := range m.rules {
		if v := rule.Evaluate(ec); v != nil {
			violations = append(violations, *v)
		}
	}

	if cfg.AI.Enabled && cfg.AI.RiskEnhancement {
		for _, rule := range m.aiRules {
			if v := rule.Evaluate(ec); v != nil {
				violations = append(violations, *v)
			}
		}
	}

	for _, v := range violations {
		switch v.Action {
		case "reject":
			if m.bus != nil {
				m.bus.Publish(model.EventRiskRejected, map[string]any{
					"decision": decision,
					"rule":     v.Rule,
					"reason":   v.Message,
				})
			}
			if m.logger != nil {
				m.logger.Warn("order rejected",
					slog.String("strategy", decision.Strategy),
					slog.String("rule", v.Rule),
					slog.String("reason", v.Message),
				)
			}
			return nil, nil

		case "reduce":
			if v.Factor > 0 {
				decision.Quantity = decision.Quantity.Mul(decimal.NewFromFloat(v.Factor)).Floor()
				if m.logger != nil {
					m.logger.Info("order reduced",
						slog.String("rule", v.Rule),
						slog.Float64("factor", v.Factor),
					)
				}
			}

		case "warn":
			if m.logger != nil {
				m.logger.Warn("risk warning", slog.String("rule", v.Rule), slog.String("message", v.Message))
			}
			if m.bus != nil {
				m.bus.Publish(model.EventRiskAIWarning, model.RiskEvent{
					ID:        data.GenerateID(),
					Type:      v.Rule,
					Action:    v.Action,
					Reason:    v.Message,
					Timestamp: time.Now().Unix(),
				})
			}
		}
	}

	if decision.Quantity.IsZero() {
		return nil, nil
	}

	if m.bus != nil {
		m.bus.Publish(model.EventRiskApproved, decision)
	}
	return &decision, nil
}

// --- Basic Rules ---

type MaxPositionRule struct{}

func (r *MaxPositionRule) Name() string { return "max_position" }

func (r *MaxPositionRule) Evaluate(ec EvalContext) *Violation {
	if ec.Risk.MaxPosition.IsZero() {
		return nil
	}
	orderValue := ec.Decision.Quantity.Mul(ec.Decision.Price)
	if orderValue.GreaterThan(ec.Risk.MaxPosition) {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("order value %s exceeds max position %s", orderValue.StringFixed(2), ec.Risk.MaxPosition.StringFixed(2)),
			Action:  "reject",
		}
	}
	return nil
}

type MaxOrderAmountRule struct{}

func (r *MaxOrderAmountRule) Name() string { return "max_order_amount" }

func (r *MaxOrderAmountRule) Evaluate(ec EvalContext) *Violation {
	if ec.Risk.MaxOrderAmount.IsZero() {
		return nil
	}
	orderValue := ec.Decision.Quantity.Mul(ec.Decision.Price)
	if orderValue.GreaterThan(ec.Risk.MaxOrderAmount) {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("order value %s exceeds max order amount %s", orderValue.StringFixed(2), ec.Risk.MaxOrderAmount.StringFixed(2)),
			Action:  "reject",
		}
	}
	return nil
}

type MaxDailyTradesRule struct{}

func (r *MaxDailyTradesRule) Name() string { return "max_daily_trades" }

func (r *MaxDailyTradesRule) Evaluate(ec EvalContext) *Violation {
	if ec.Risk.MaxDailyTrades <= 0 {
		return nil
	}
	startOfDay := time.Now().Truncate(24 * time.Hour).Unix()
	count := 0
	for _, o := range ec.RecentOrders {
		if o.CreatedAt >= startOfDay {
			count++
		}
	}
	if count >= ec.Risk.MaxDailyTrades {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("daily trade limit %d reached (%d trades today)", ec.Risk.MaxDailyTrades, count),
			Action:  "reject",
		}
	}
	return nil
}

type MinTradeIntervalRule struct{}

func (r *MinTradeIntervalRule) Name() string { return "min_trade_interval" }

func (r *MinTradeIntervalRule) Evaluate(ec EvalContext) *Violation {
	if ec.Risk.MinTradeInterval <= 0 || len(ec.RecentOrders) == 0 {
		return nil
	}
	lastTime := ec.RecentOrders[0].CreatedAt
	elapsed := time.Now().Unix() - lastTime
	if elapsed < ec.Risk.MinTradeInterval {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("min interval not met: %ds elapsed, %ds required", elapsed, ec.Risk.MinTradeInterval),
			Action:  "reject",
		}
	}
	return nil
}

type MaxDrawdownRule struct{}

func (r *MaxDrawdownRule) Name() string { return "max_drawdown" }

func (r *MaxDrawdownRule) Evaluate(ec EvalContext) *Violation {
	if ec.Risk.MaxDrawdown <= 0 || ec.Account == nil {
		return nil
	}
	if ec.Account.TotalValue.IsZero() || ec.Account.Cash.IsZero() {
		return nil
	}
	drawdown := decimal.NewFromFloat(1).Sub(ec.Account.TotalValue.Div(ec.Account.Cash))
	drawdownPct, _ := drawdown.Float64()
	if math.Abs(drawdownPct) > ec.Risk.MaxDrawdown {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("drawdown %.2f%% exceeds max %.2f%%", math.Abs(drawdownPct)*100, ec.Risk.MaxDrawdown*100),
			Action:  "reject",
		}
	}
	return nil
}

// --- AI-Enhanced Rules ---

type PanicProtectionRule struct{}

func (r *PanicProtectionRule) Name() string { return "panic_protection" }

func (r *PanicProtectionRule) Evaluate(ec EvalContext) *Violation {
	threshold := ec.AI.RiskThresholds.PanicThreshold
	if threshold == 0 {
		threshold = -0.8
	}
	if ec.Sentiment < threshold {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("panic protection: sentiment %.2f below threshold %.2f", ec.Sentiment, threshold),
			Action:  "reject",
		}
	}
	return nil
}

type OverheatWarningRule struct{}

func (r *OverheatWarningRule) Name() string { return "overheat_warning" }

func (r *OverheatWarningRule) Evaluate(ec EvalContext) *Violation {
	threshold := ec.AI.RiskThresholds.OverheatThreshold
	if threshold == 0 {
		threshold = 0.8
	}
	if ec.Sentiment > threshold {
		reduction := ec.AI.RiskThresholds.PositionReduction
		if reduction <= 0 {
			reduction = 0.5
		}
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("overheat warning: sentiment %.2f above %.2f, reducing by %.0f%%", ec.Sentiment, threshold, (1-reduction)*100),
			Action:  "reduce",
			Factor:  reduction,
		}
	}
	return nil
}

type EventPauseRule struct{}

func (r *EventPauseRule) Name() string { return "event_pause" }

func (r *EventPauseRule) Evaluate(ec EvalContext) *Violation {
	threshold := ec.AI.RiskThresholds.EventPauseThreshold
	if threshold == 0 {
		threshold = 0.9
	}
	absSentiment := math.Abs(ec.Sentiment)
	if absSentiment > threshold {
		return &Violation{
			Rule:    r.Name(),
			Message: fmt.Sprintf("event pause: extreme sentiment %.2f exceeds %.2f", ec.Sentiment, threshold),
			Action:  "warn",
		}
	}
	return nil
}

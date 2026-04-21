package execution

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/model"
)

const defaultAccountID = "paper_01"

type Engine struct {
	repo       *data.Repository
	bus        *eventbus.InMemoryBus
	logger     *slog.Logger
	commission decimal.Decimal
	slippage   decimal.Decimal
}

func NewEngine(repo *data.Repository, bus *eventbus.InMemoryBus, logger *slog.Logger) *Engine {
	return &Engine{
		repo:       repo,
		bus:        bus,
		logger:     logger,
		commission: decimal.NewFromFloat(0.001), // 0.1%
		slippage:   decimal.NewFromFloat(0.01),
	}
}

func (e *Engine) Start(ctx context.Context) error {
	// Ensure default paper trading account exists
	if _, err := e.repo.GetAccount(ctx, defaultAccountID); err != nil {
		now := time.Now().Unix()
		account := model.Account{
			ID:         defaultAccountID,
			Cash:       decimal.NewFromInt(1000000), // $1M paper money
			TotalValue: decimal.NewFromInt(1000000),
			Currency:   "USD",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := e.repo.CreateAccount(ctx, account); err != nil {
			return err
		}
		e.logInfo("created default paper trading account", slog.String("id", defaultAccountID))
	}

	// Subscribe to risk-approved decisions
	e.bus.SubscribeFunc(model.EventRiskApproved, func(ev any) {
		decision, ok := ev.(model.OrderDecision)
		if !ok {
			return
		}
		e.executeOrder(ctx, decision)
	})

	// Also subscribe directly to strategy decisions (when no risk manager in path)
	e.bus.SubscribeFunc(model.EventStrategyDecision, func(ev any) {
		decision, ok := ev.(model.OrderDecision)
		if !ok {
			return
		}
		e.executeOrder(ctx, decision)
	})

	e.logInfo("execution engine started")
	return nil
}

func (e *Engine) executeOrder(ctx context.Context, decision model.OrderDecision) {
	account, err := e.repo.GetAccount(ctx, defaultAccountID)
	if err != nil {
		e.logError("failed to get account", slog.String("error", err.Error()))
		return
	}

	// Get latest price from decision
	price := decision.Price
	if price.IsZero() {
		e.logWarn("order rejected: no price", slog.String("symbol", decision.Symbol))
		e.bus.Publish(model.EventExecutionReject, decision)
		return
	}

	// Calculate fill price with slippage
	fillPrice := e.applySlippage(price, decision.Side)
	orderValue := fillPrice.Mul(decision.Quantity)
	fee := orderValue.Mul(e.commission)

	order := model.Order{
		ID:        data.GenerateID(),
		AccountID: defaultAccountID,
		Symbol:    decision.Symbol,
		Side:      decision.Side,
		Type:      decision.Type,
		Status:    model.OrderFilled,
		Quantity:  decision.Quantity,
		FilledQty: decision.Quantity,
		Price:     price,
		AvgFillPrice: fillPrice,
		Fee:       fee,
		Slippage:  fillPrice.Sub(price).Abs(),
		Strategy:  decision.Strategy,
		Reason:    decision.Reason,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	switch decision.Side {
	case model.OrderBuy:
		totalCost := orderValue.Add(fee)
		if totalCost.GreaterThan(account.Cash) {
			// Reduce quantity
			qty := account.Cash.Div(fillPrice.Mul(decimal.NewFromInt(1).Add(e.commission))).Floor()
			if qty.IsZero() {
				order.Status = model.OrderRejected
				e.repo.SaveOrder(ctx, order)
				e.bus.Publish(model.EventExecutionReject, decision)
				return
			}
			order.Quantity = qty
			order.FilledQty = qty
			orderValue = fillPrice.Mul(qty)
			fee = orderValue.Mul(e.commission)
			order.Fee = fee
			totalCost = orderValue.Add(fee)
		}
		account.Cash = account.Cash.Sub(totalCost)
		e.updatePosition(ctx, account.ID, decision.Symbol, decision.Side, order.FilledQty, fillPrice)

	case model.OrderSell:
		pos, err := e.repo.GetPosition(ctx, account.ID, decision.Symbol)
		if err != nil || pos == nil || pos.Quantity.IsZero() {
			order.Status = model.OrderRejected
			e.repo.SaveOrder(ctx, order)
			e.bus.Publish(model.EventExecutionReject, decision)
			return
		}
		sellQty := order.FilledQty
		if sellQty.GreaterThan(pos.Quantity) {
			sellQty = pos.Quantity
			order.Quantity = sellQty
			order.FilledQty = sellQty
			orderValue = fillPrice.Mul(sellQty)
			fee = orderValue.Mul(e.commission)
			order.Fee = fee
		}
		pnl := fillPrice.Sub(pos.AvgPrice).Mul(sellQty).Sub(fee)
		account.Cash = account.Cash.Add(orderValue).Sub(fee)
		e.closeOrReducePosition(ctx, account.ID, decision.Symbol, sellQty, fillPrice, pnl)
	}

	account.TotalValue = account.Cash
	e.repo.UpdateAccount(ctx, *account)
	e.repo.SaveOrder(ctx, order)

	// Update total value with current positions
	e.updateTotalValue(ctx, account)

	fill := model.Fill{
		OrderID:   order.ID,
		Symbol:    decision.Symbol,
		Side:      decision.Side,
		Price:     fillPrice,
		Quantity:  order.FilledQty,
		Fee:       fee,
		Slippage:  order.Slippage,
		Strategy:  decision.Strategy,
		Timestamp: time.Now().Unix(),
	}
	e.repo.SaveFill(ctx, fill)

	e.bus.Publish(model.EventExecutionFill, fill)
	e.logInfo("order executed",
		slog.String("symbol", decision.Symbol),
		slog.String("side", string(decision.Side)),
		slog.String("qty", order.FilledQty.String()),
		slog.String("price", fillPrice.StringFixed(2)),
	)
}

func (e *Engine) applySlippage(price decimal.Decimal, side model.OrderSide) decimal.Decimal {
	switch side {
	case model.OrderBuy:
		return price.Add(e.slippage)
	case model.OrderSell:
		return price.Sub(e.slippage)
	default:
		return price
	}
}

func (e *Engine) updatePosition(ctx context.Context, accountID, symbol string, side model.OrderSide, qty, price decimal.Decimal) {
	pos, err := e.repo.GetPosition(ctx, accountID, symbol)
	if err != nil || pos == nil {
		pos = &model.Position{
			ID:        data.GenerateID(),
			AccountID: accountID,
			Symbol:    symbol,
			Side:      side,
			Quantity:  qty,
			AvgPrice:  price,
			OpenedAt:  time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		}
	} else {
		totalQty := pos.Quantity.Add(qty)
		avgPrice := pos.AvgPrice.Mul(pos.Quantity).Add(price.Mul(qty)).Div(totalQty)
		pos.Quantity = totalQty
		pos.AvgPrice = avgPrice
		pos.UpdatedAt = time.Now().Unix()
	}
	e.repo.UpsertPosition(ctx, *pos)
}

func (e *Engine) closeOrReducePosition(ctx context.Context, accountID, symbol string, qty, exitPrice, pnl decimal.Decimal) {
	pos, err := e.repo.GetPosition(ctx, accountID, symbol)
	if err != nil || pos == nil {
		return
	}

	pos.RealizedPnL = pos.RealizedPnL.Add(pnl)
	pos.Quantity = pos.Quantity.Sub(qty)
	pos.UpdatedAt = time.Now().Unix()

	if pos.Quantity.IsZero() {
		// Position fully closed — set to zero quantity
		pos.Quantity = decimal.Zero
	}
	e.repo.UpsertPosition(ctx, *pos)
}

func (e *Engine) updateTotalValue(ctx context.Context, account *model.Account) {
	positions, err := e.repo.ListPositions(ctx, account.ID)
	if err != nil {
		return
	}
	total := account.Cash
	for _, p := range positions {
		if p.Quantity.IsZero() {
			continue
		}
		total = total.Add(p.AvgPrice.Mul(p.Quantity))
	}
	account.TotalValue = total
	e.repo.UpdateAccount(ctx, *account)
}

// SubmitOrder allows manual order submission via API.
func (e *Engine) SubmitOrder(ctx context.Context, decision model.OrderDecision) (*model.Order, error) {
	account, err := e.repo.GetAccount(ctx, defaultAccountID)
	if err != nil {
		return nil, err
	}

	price := decision.Price
	fillPrice := e.applySlippage(price, decision.Side)
	orderValue := fillPrice.Mul(decision.Quantity)
	fee := orderValue.Mul(e.commission)

	order := model.Order{
		ID:        data.GenerateID(),
		AccountID: defaultAccountID,
		Symbol:    decision.Symbol,
		Side:      decision.Side,
		Type:      decision.Type,
		Status:    model.OrderFilled,
		Quantity:  decision.Quantity,
		FilledQty: decision.Quantity,
		Price:     price,
		AvgFillPrice: fillPrice,
		Fee:       fee,
		Slippage:  fillPrice.Sub(price).Abs(),
		Strategy:  decision.Strategy,
		Reason:    decision.Reason,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	switch decision.Side {
	case model.OrderBuy:
		totalCost := orderValue.Add(fee)
		if totalCost.GreaterThan(account.Cash) {
			return nil, nil
		}
		account.Cash = account.Cash.Sub(totalCost)
		e.updatePosition(ctx, account.ID, decision.Symbol, decision.Side, order.FilledQty, fillPrice)

	case model.OrderSell:
		pos, err := e.repo.GetPosition(ctx, account.ID, decision.Symbol)
		if err != nil || pos == nil {
			return nil, nil
		}
		sellQty := decision.Quantity
		if sellQty.GreaterThan(pos.Quantity) {
			sellQty = pos.Quantity
		}
		order.Quantity = sellQty
		order.FilledQty = sellQty
		orderValue = fillPrice.Mul(sellQty)
		fee = orderValue.Mul(e.commission)
		order.Fee = fee
		pnl := fillPrice.Sub(pos.AvgPrice).Mul(sellQty).Sub(fee)
		account.Cash = account.Cash.Add(orderValue).Sub(fee)
		e.closeOrReducePosition(ctx, account.ID, decision.Symbol, sellQty, fillPrice, pnl)
	}

	account.TotalValue = account.Cash
	e.repo.UpdateAccount(ctx, *account)
	e.repo.SaveOrder(ctx, order)
	e.updateTotalValue(ctx, account)

	return &order, nil
}

// GetAccountID returns the default paper trading account ID.
func (e *Engine) GetAccountID() string {
	return defaultAccountID
}

func (e *Engine) logInfo(msg string, args ...any) {
	if e.logger != nil {
		e.logger.Info(msg, args...)
	}
}

func (e *Engine) logError(msg string, args ...any) {
	if e.logger != nil {
		e.logger.Error(msg, args...)
	}
}

func (e *Engine) logWarn(msg string, args ...any) {
	if e.logger != nil {
		e.logger.Warn(msg, args...)
	}
}

package execution

import (
	"context"
	"log/slog"
	"sync"
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

	mu            sync.RWMutex
	pendingOrders map[string]model.Order // orderID -> order (status=NEW)
}

func NewEngine(repo *data.Repository, bus *eventbus.InMemoryBus, logger *slog.Logger) *Engine {
	return &Engine{
		repo:          repo,
		bus:           bus,
		logger:        logger,
		commission:    decimal.NewFromFloat(0.001), // 0.1%
		slippage:      decimal.NewFromFloat(0.01),
		pendingOrders: make(map[string]model.Order),
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

	// Subscribe to market ticks for LIMIT order matching
	e.bus.SubscribeFunc(model.EventMarketTick, func(ev any) {
		tick, ok := ev.(model.Tick)
		if !ok {
			return
		}
		e.matchPendingOrders(ctx, tick.Symbol, tick.Price)
	})

	e.logInfo("execution engine started")
	return nil
}

// matchPendingOrders checks all pending LIMIT orders for a symbol against the current price.
// BUY LIMIT: fills when market price <= limit price
// SELL LIMIT: fills when market price >= limit price
func (e *Engine) matchPendingOrders(ctx context.Context, symbol string, marketPrice decimal.Decimal) {
	if marketPrice.IsZero() {
		return
	}

	e.mu.Lock()
	var toFill []model.Order
	for id, order := range e.pendingOrders {
		if order.Symbol != symbol {
			continue
		}
		matched := false
		if order.Side == model.OrderBuy && marketPrice.LessThanOrEqual(order.Price) {
			matched = true
		} else if order.Side == model.OrderSell && marketPrice.GreaterThanOrEqual(order.Price) {
			matched = true
		}
		if matched {
			delete(e.pendingOrders, id)
			toFill = append(toFill, order)
		}
	}
	e.mu.Unlock()

	for _, order := range toFill {
		e.fillOrder(ctx, order)
	}
}

func (e *Engine) executeOrder(ctx context.Context, decision model.OrderDecision) {
	account, err := e.repo.GetAccount(ctx, defaultAccountID)
	if err != nil {
		e.logError("failed to get account", slog.String("error", err.Error()))
		return
	}

	price := decision.Price
	if price.IsZero() {
		e.logWarn("order rejected: no price", slog.String("symbol", decision.Symbol))
		e.bus.Publish(model.EventExecutionReject, decision)
		return
	}

	// Strategy-driven orders are always MARKET — fill immediately
	e.fillOrderFromDecision(ctx, *account, decision)
}

// fillOrder fills a previously-pending LIMIT order.
func (e *Engine) fillOrder(ctx context.Context, order model.Order) {
	account, err := e.repo.GetAccount(ctx, defaultAccountID)
	if err != nil {
		e.logError("failed to get account for limit fill", slog.String("error", err.Error()))
		return
	}

	fillPrice := e.applySlippage(order.Price, order.Side)
	orderValue := fillPrice.Mul(order.Quantity)
	fee := orderValue.Mul(e.commission)

	order.Status = model.OrderFilled
	order.FilledQty = order.Quantity
	order.AvgFillPrice = fillPrice
	order.Fee = fee
	order.Slippage = fillPrice.Sub(order.Price).Abs()
	order.UpdatedAt = time.Now().Unix()

	switch order.Side {
	case model.OrderBuy:
		totalCost := orderValue.Add(fee)
		if totalCost.GreaterThan(account.Cash) {
			qty := account.Cash.Div(fillPrice.Mul(decimal.NewFromInt(1).Add(e.commission))).Floor()
			if qty.IsZero() {
				order.Status = model.OrderRejected
				e.repo.SaveOrder(ctx, order)
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
		e.updatePosition(ctx, account.ID, order.Symbol, order.Side, order.FilledQty, fillPrice)

	case model.OrderSell:
		pos, err := e.repo.GetPosition(ctx, account.ID, order.Symbol)
		if err != nil || pos == nil || pos.Quantity.IsZero() {
			order.Status = model.OrderRejected
			e.repo.SaveOrder(ctx, order)
			return
		}
		sellQty := order.Quantity
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
		e.closeOrReducePosition(ctx, account.ID, order.Symbol, sellQty, fillPrice, pnl)
	}

	account.TotalValue = account.Cash
	e.repo.UpdateAccount(ctx, *account)
	e.repo.SaveOrder(ctx, order)
	e.updateTotalValue(ctx, account)

	fill := model.Fill{
		OrderID:   order.ID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Price:     fillPrice,
		Quantity:  order.FilledQty,
		Fee:       fee,
		Slippage:  order.Slippage,
		Strategy:  order.Strategy,
		Timestamp: time.Now().Unix(),
	}
	e.repo.SaveFill(ctx, fill)
	e.bus.Publish(model.EventExecutionFill, fill)

	e.logInfo("limit order filled",
		slog.String("symbol", order.Symbol),
		slog.String("side", string(order.Side)),
		slog.String("qty", order.FilledQty.String()),
		slog.String("price", fillPrice.StringFixed(2)),
	)
}

func (e *Engine) fillOrderFromDecision(ctx context.Context, account model.Account, decision model.OrderDecision) {
	fillPrice := e.applySlippage(decision.Price, decision.Side)
	orderValue := fillPrice.Mul(decision.Quantity)
	fee := orderValue.Mul(e.commission)

	order := model.Order{
		ID:           data.GenerateID(),
		AccountID:    defaultAccountID,
		Symbol:       decision.Symbol,
		Side:         decision.Side,
		Type:         decision.Type,
		Status:       model.OrderFilled,
		Quantity:     decision.Quantity,
		FilledQty:    decision.Quantity,
		Price:        decision.Price,
		AvgFillPrice: fillPrice,
		Fee:          fee,
		Slippage:     fillPrice.Sub(decision.Price).Abs(),
		Strategy:     decision.Strategy,
		Reason:       decision.Reason,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	switch decision.Side {
	case model.OrderBuy:
		totalCost := orderValue.Add(fee)
		if totalCost.GreaterThan(account.Cash) {
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
	e.repo.UpdateAccount(ctx, account)
	e.repo.SaveOrder(ctx, order)
	e.updateTotalValue(ctx, &account)

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

// SubmitOrder handles manual API orders.
// MARKET orders fill immediately. LIMIT orders are saved as NEW and wait for price match.
func (e *Engine) SubmitOrder(ctx context.Context, decision model.OrderDecision) (*model.Order, error) {
	// MARKET orders fill immediately
	if decision.Type == model.OrderMarket || decision.Type == "" {
		account, err := e.repo.GetAccount(ctx, defaultAccountID)
		if err != nil {
			return nil, err
		}
		e.fillOrderFromDecision(ctx, *account, decision)
		// Retrieve the order we just saved (not ideal but works)
		return e.findLatestOrder(ctx, decision.Symbol, decision.Strategy)
	}

	// LIMIT orders — save as NEW, wait for tick match
	order := model.Order{
		ID:           data.GenerateID(),
		AccountID:    defaultAccountID,
		Symbol:       decision.Symbol,
		Side:         decision.Side,
		Type:         decision.Type,
		Status:       model.OrderNew,
		Quantity:     decision.Quantity,
		FilledQty:    decimal.Zero,
		Price:        decision.Price,
		AvgFillPrice: decimal.Zero,
		Fee:          decimal.Zero,
		Slippage:     decimal.Zero,
		Strategy:     decision.Strategy,
		Reason:       decision.Reason,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	if err := e.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.pendingOrders[order.ID] = order
	e.mu.Unlock()

	e.logInfo("limit order pending",
		slog.String("id", order.ID),
		slog.String("symbol", decision.Symbol),
		slog.String("side", string(decision.Side)),
		slog.String("limitPrice", decision.Price.StringFixed(2)),
		slog.String("qty", decision.Quantity.String()),
	)

	return &order, nil
}

// CancelOrder cancels a pending LIMIT order.
func (e *Engine) CancelOrder(ctx context.Context, orderID string) error {
	e.mu.Lock()
	order, exists := e.pendingOrders[orderID]
	if !exists {
		e.mu.Unlock()
		return nil
	}
	delete(e.pendingOrders, orderID)
	e.mu.Unlock()

	order.Status = model.OrderCancelled
	order.UpdatedAt = time.Now().Unix()
	return e.repo.SaveOrder(ctx, order)
}

func (e *Engine) findLatestOrder(ctx context.Context, symbol, strategy string) (*model.Order, error) {
	orders, err := e.repo.GetOrders(ctx, defaultAccountID, 10)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		if o.Symbol == symbol && o.Strategy == strategy && o.Status == model.OrderFilled {
			return &o, nil
		}
	}
	return nil, nil
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

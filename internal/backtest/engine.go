package backtest

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/strategy"
	"github.com/flametest/market-lens/pkg/indicator"
	"github.com/flametest/market-lens/pkg/model"
)

type Engine struct {
	repo *data.Repository
}

func NewEngine(repo *data.Repository) *Engine {
	return &Engine{repo: repo}
}

func (e *Engine) Run(ctx context.Context, cfg model.BacktestConfig) (*model.BacktestResult, error) {
	candles, err := e.repo.GetCandles(ctx, cfg.Symbol, cfg.Interval, cfg.StartDate, cfg.EndDate)
	if err != nil {
		return nil, fmt.Errorf("load candles: %w", err)
	}
	if len(candles) < 30 {
		return nil, fmt.Errorf("insufficient data: %d candles, need at least 30", len(candles))
	}

	s, err := createStrategy(cfg.Strategy)
	if err != nil {
		return nil, err
	}

	sim := newSimulator(cfg.InitialCash, cfg.Commission, cfg.Slippage)

	for i := 20; i < len(candles); i++ {
		window := candles[:i+1]
		sim.recordEquity(candles[i])

		sig := generateSignal(cfg.Strategy.Name, window)
		if sig.Type == model.SignalNeutral {
			continue
		}

		decision, err := s.OnSignal(ctx, sig, window)
		if err != nil || decision == nil {
			continue
		}

		if !checkRisk(decision, cfg.Risk, sim) {
			continue
		}

		if i+1 < len(candles) {
			sim.execute(decision.Side, decision.Quantity, candles[i+1])
		}
	}

	// Close any open position at the last candle
	if sim.hasPosition() && len(candles) > 0 {
		sim.execute(model.OrderSell, sim.positionQuantity(), candles[len(candles)-1])
	}

	return sim.buildResult(cfg.InitialCash, cfg.StartDate, cfg.EndDate), nil
}

func (e *Engine) RunWithCompare(ctx context.Context, cfg model.BacktestConfig) (*model.BacktestResult, *model.BacktestResult, error) {
	// Run without AI adjustments
	baseResult, err := e.Run(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	return baseResult, nil, nil
}

func createStrategy(sc model.StrategyConfig) (strategy.Strategy, error) {
	switch sc.Name {
	case "ma_crossover":
		return strategy.NewMACrossover(sc), nil
	case "rsi_reversion":
		return strategy.NewRSIReversion(sc), nil
	default:
		return nil, fmt.Errorf("unsupported strategy: %s", sc.Name)
	}
}

func generateSignal(name string, candles []model.Candle) model.Signal {
	if len(candles) == 0 {
		return model.Signal{}
	}
	last := candles[len(candles)-1]

	switch name {
	case "ma_crossover":
		return model.Signal{
			Symbol:    last.Symbol,
			Source:    "MA",
			Type:      model.SignalBuy,
			Strength:  1.0,
			Price:     last.Close,
			Timestamp: last.Timestamp,
		}
	case "rsi_reversion":
		rsi := &indicator.RSI{Period: 14, Oversold: 30, Overbought: 70}
		result := rsi.Compute(candles)
		return model.Signal{
			Symbol:    last.Symbol,
			Source:    "RSI",
			Type:      result.Signal,
			Strength:  result.Strength,
			Price:     last.Close,
			Timestamp: last.Timestamp,
		}
	default:
		return model.Signal{
			Symbol:    last.Symbol,
			Source:    name,
			Type:      model.SignalBuy,
			Strength:  1.0,
			Price:     last.Close,
			Timestamp: last.Timestamp,
		}
	}
}

func checkRisk(decision *model.OrderDecision, risk model.RiskConfig, sim *simulator) bool {
	orderValue := decision.Quantity.Mul(decision.Price)
	if !risk.MaxOrderAmount.IsZero() && orderValue.GreaterThan(risk.MaxOrderAmount) {
		return false
	}
	return true
}

// --- simulator ---

type simPosition struct {
	entryPrice decimal.Decimal
	quantity   decimal.Decimal
	entryTime  int64
	side       model.OrderSide
}

type simulator struct {
	cash        decimal.Decimal
	commission  decimal.Decimal
	slippage    decimal.Decimal
	position    *simPosition
	trades      []model.SimulatedTrade
	equityCurve []model.EquityPoint
	peakEquity  decimal.Decimal
	maxDrawdown decimal.Decimal
}

func newSimulator(cash, commission, slippage decimal.Decimal) *simulator {
	return &simulator{
		cash:        cash,
		commission:  commission,
		slippage:    slippage,
		trades:      make([]model.SimulatedTrade, 0),
		equityCurve: make([]model.EquityPoint, 0),
		peakEquity:  cash,
		maxDrawdown: decimal.Zero,
	}
}

func (s *simulator) equity(price decimal.Decimal) decimal.Decimal {
	e := s.cash
	if s.position != nil {
		e = e.Add(s.position.quantity.Mul(price))
	}
	return e
}

func (s *simulator) recordEquity(candle model.Candle) {
	eq := s.equity(candle.Close)
	s.equityCurve = append(s.equityCurve, model.EquityPoint{
		Timestamp: candle.Timestamp,
		Value:     eq,
	})
	if eq.GreaterThan(s.peakEquity) {
		s.peakEquity = eq
	}
	if s.peakEquity.GreaterThan(decimal.Zero) {
		dd := decimal.NewFromInt(1).Sub(eq.Div(s.peakEquity))
		if dd.GreaterThan(s.maxDrawdown) {
			s.maxDrawdown = dd
		}
	}
}

func (s *simulator) execute(side model.OrderSide, qty decimal.Decimal, candle model.Candle) {
	if qty.IsZero() {
		return
	}

	switch side {
	case model.OrderBuy:
		s.executeBuy(qty, candle)
	case model.OrderSell:
		s.executeSell(qty, candle)
	}
}

func (s *simulator) executeBuy(qty decimal.Decimal, candle model.Candle) {
	fillPrice := candle.Open.Add(s.slippage)
	orderValue := fillPrice.Mul(qty)
	fee := orderValue.Mul(s.commission)
	totalCost := orderValue.Add(fee)

	if totalCost.GreaterThan(s.cash) {
		if s.commission.GreaterThan(decimal.Zero) {
			qty = s.cash.Div(fillPrice.Mul(decimal.NewFromInt(1).Add(s.commission))).Floor()
		} else {
			qty = s.cash.Div(fillPrice).Floor()
		}
		if qty.IsZero() {
			return
		}
		orderValue = fillPrice.Mul(qty)
		fee = orderValue.Mul(s.commission)
		totalCost = orderValue.Add(fee)
	}

	// Close existing short
	if s.position != nil && s.position.side == model.OrderSell {
		pnl := s.position.entryPrice.Sub(fillPrice).Mul(s.position.quantity).Sub(fee)
		s.cash = s.cash.Add(s.position.entryPrice.Mul(s.position.quantity)).Sub(fee)
		s.recordTrade(model.OrderSell, fillPrice, s.position.quantity, pnl, fee, candle.Timestamp)
		s.position = nil
	}

	if s.position != nil && s.position.side == model.OrderBuy {
		totalQty := s.position.quantity.Add(qty)
		avgPrice := s.position.entryPrice.Mul(s.position.quantity).Add(fillPrice.Mul(qty)).Div(totalQty)
		s.position.entryPrice = avgPrice
		s.position.quantity = totalQty
	} else {
		s.position = &simPosition{
			entryPrice: fillPrice,
			quantity:   qty,
			entryTime:  candle.Timestamp,
			side:       model.OrderBuy,
		}
	}
	s.cash = s.cash.Sub(totalCost)
}

func (s *simulator) executeSell(qty decimal.Decimal, candle model.Candle) {
	fillPrice := candle.Open.Sub(s.slippage)

	if s.position != nil && s.position.side == model.OrderBuy {
		actualQty := qty
		if actualQty.GreaterThan(s.position.quantity) {
			actualQty = s.position.quantity
		}
		orderValue := fillPrice.Mul(actualQty)
		fee := orderValue.Mul(s.commission)
		pnl := fillPrice.Sub(s.position.entryPrice).Mul(actualQty).Sub(fee)

		s.cash = s.cash.Add(orderValue).Sub(fee)
		s.recordTrade(model.OrderBuy, fillPrice, actualQty, pnl, fee, candle.Timestamp)
		s.position = nil
	}
}

func (s *simulator) recordTrade(side model.OrderSide, exitPrice, qty, pnl, fee decimal.Decimal, exitTime int64) {
	if s.position == nil {
		return
	}
	symbol := ""
	if len(s.trades) == 0 && s.position != nil {
		// Get symbol from position context (set default)
	}
	s.trades = append(s.trades, model.SimulatedTrade{
		Symbol:     symbol,
		Side:       side,
		EntryTime:  s.position.entryTime,
		ExitTime:   exitTime,
		EntryPrice: s.position.entryPrice,
		ExitPrice:  exitPrice,
		Quantity:   qty,
		PnL:        pnl,
		Fee:        fee,
	})
}

func (s *simulator) hasPosition() bool {
	return s.position != nil
}

func (s *simulator) positionQuantity() decimal.Decimal {
	if s.position == nil {
		return decimal.Zero
	}
	return s.position.quantity
}

func (s *simulator) buildResult(initialCash decimal.Decimal, startMs, endMs int64) *model.BacktestResult {
	finalEquity := s.cash
	if s.position != nil && len(s.equityCurve) > 0 {
		finalEquity = s.equityCurve[len(s.equityCurve)-1].Value
	}

	totalReturn := decimal.Zero
	if !initialCash.IsZero() {
		totalReturn = finalEquity.Sub(initialCash).Div(initialCash)
	}

	annualReturn := decimal.Zero
	durationMs := endMs - startMs
	if durationMs > 0 {
		years := decimal.NewFromInt(durationMs).Div(decimal.NewFromInt(365 * 24 * 3600 * 1000))
		if !years.IsZero() {
			annualReturn = totalReturn.Div(years)
		}
	}

	totalTrades := len(s.trades)
	profitTrades := 0
	for _, t := range s.trades {
		if t.PnL.GreaterThan(decimal.Zero) {
			profitTrades++
		}
	}

	winRate := decimal.Zero
	if totalTrades > 0 {
		winRate = decimal.NewFromInt(int64(profitTrades)).Div(decimal.NewFromInt(int64(totalTrades)))
	}

	return &model.BacktestResult{
		TotalReturn:  totalReturn,
		AnnualReturn: annualReturn,
		MaxDrawdown:  s.maxDrawdown,
		SharpeRatio:  calcSharpe(s.equityCurve),
		WinRate:      winRate,
		TotalTrades:  totalTrades,
		ProfitTrades: profitTrades,
		LossTrades:   totalTrades - profitTrades,
		Trades:       s.trades,
		EquityCurve:  s.equityCurve,
	}
}

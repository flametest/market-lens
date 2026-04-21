package data

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/flametest/market-lens/pkg/model"
)

type Repository struct {
	db *DB
}

func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveSignal(ctx context.Context, sig model.Signal) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO signals (signal_id, symbol, source, type, strength, price, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sig.ID, sig.Symbol, sig.Source, sig.Type, sig.Strength,
		sig.Price.String(), sig.Timestamp,
	)
	return err
}

func (r *Repository) GetSignals(ctx context.Context, symbol string, limit int) ([]model.Signal, error) {
	query := `SELECT signal_id, symbol, source, type, strength, price, timestamp FROM signals`
	args := []any{}
	if symbol != "" {
		query += ` WHERE symbol = ?`
		args = append(args, symbol)
	}
	query += ` ORDER BY timestamp DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []model.Signal
	for rows.Next() {
		var s model.Signal
		var price string
		if err := rows.Scan(&s.ID, &s.Symbol, &s.Source, &s.Type, &s.Strength, &price, &s.Timestamp); err != nil {
			return nil, err
		}
		s.Price = dec(price)
		signals = append(signals, s)
	}
	return signals, rows.Err()
}

func (r *Repository) GetCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error) {
	query := `SELECT symbol, interval, timestamp, open, high, low, close, volume, source
			  FROM candles WHERE symbol = ? AND interval = ?`
	args := []any{symbol, interval}
	if start > 0 {
		query += ` AND timestamp >= ?`
		args = append(args, start)
	}
	if end > 0 {
		query += ` AND timestamp <= ?`
		args = append(args, end)
	}
	query += ` ORDER BY timestamp ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandles(rows)
}

func (r *Repository) SaveCandle(ctx context.Context, c model.Candle) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO candles (symbol, interval, timestamp, open, high, low, close, volume, source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Symbol, c.Interval, c.Timestamp,
		c.Open.String(), c.High.String(), c.Low.String(),
		c.Close.String(), c.Volume.String(), c.Source,
	)
	return err
}

func (r *Repository) SaveStrategyConfig(ctx context.Context, sc model.StrategyConfig) error {
	params := marshalJSON(sc.Params)
	symbols := marshalJSON(sc.Symbols)
	risk := marshalJSON(sc.Risk)
	ai := marshalJSON(sc.AI)
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO strategy_configs (id, name, display_name, enabled, params, symbols, risk_config, ai_config, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.ID, sc.Name, sc.DisplayName, sc.Enabled, params, symbols, risk, ai, sc.CreatedAt, time.Now().Unix(),
	)
	return err
}

func (r *Repository) GetStrategyConfigs(ctx context.Context) ([]model.StrategyConfig, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, display_name, enabled, params, symbols, risk_config, ai_config, created_at, updated_at
		 FROM strategy_configs ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.StrategyConfig
	for rows.Next() {
		var sc model.StrategyConfig
		var params, symbols, risk, ai string
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.DisplayName, &sc.Enabled, &params, &symbols, &risk, &ai, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		unmarshalJSON(params, &sc.Params)
		unmarshalJSON(symbols, &sc.Symbols)
		unmarshalJSON(risk, &sc.Risk)
		unmarshalJSON(ai, &sc.AI)
		configs = append(configs, sc)
	}
	return configs, rows.Err()
}

func (r *Repository) GetAccount(ctx context.Context, id string) (*model.Account, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, cash, total_value, currency, created_at, updated_at FROM accounts WHERE id = ?`, id)
	var a model.Account
	var cash, totalValue string
	if err := row.Scan(&a.ID, &cash, &totalValue, &a.Currency, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.Cash = dec(cash)
	a.TotalValue = dec(totalValue)
	return &a, nil
}

func (r *Repository) CreateAccount(ctx context.Context, a model.Account) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO accounts (id, cash, total_value, currency, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.Cash.String(), a.TotalValue.String(), a.Currency, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *Repository) UpdateAccount(ctx context.Context, a model.Account) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET cash = ?, total_value = ?, updated_at = ? WHERE id = ?`,
		a.Cash.String(), a.TotalValue.String(), time.Now().Unix(), a.ID,
	)
	return err
}

func (r *Repository) SaveOrder(ctx context.Context, o model.Order) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO orders (id, account_id, symbol, side, type, status, quantity, filled_qty, price, avg_fill_price, fee, slippage, strategy, reason, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, o.AccountID, o.Symbol, o.Side, o.Type, o.Status,
		o.Quantity.String(), o.FilledQty.String(), o.Price.String(),
		o.AvgFillPrice.String(), o.Fee.String(), o.Slippage.String(),
		o.Strategy, o.Reason, o.CreatedAt, o.UpdatedAt,
	)
	return err
}

func (r *Repository) GetOrders(ctx context.Context, accountID string, limit int) ([]model.Order, error) {
	query := `SELECT id, account_id, symbol, side, type, status, quantity, filled_qty, price, avg_fill_price, fee, slippage, strategy, reason, created_at, updated_at
			  FROM orders WHERE account_id = ? ORDER BY created_at DESC`
	args := []any{accountID}
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		var qty, filled, price, avgPrice, fee, slippage string
		if err := rows.Scan(&o.ID, &o.AccountID, &o.Symbol, &o.Side, &o.Type, &o.Status,
			&qty, &filled, &price, &avgPrice, &fee, &slippage, &o.Strategy, &o.Reason, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		o.Quantity = dec(qty)
		o.FilledQty = dec(filled)
		o.Price = dec(price)
		o.AvgFillPrice = dec(avgPrice)
		o.Fee = dec(fee)
		o.Slippage = dec(slippage)
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *Repository) GetPosition(ctx context.Context, accountID, symbol string) (*model.Position, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, account_id, symbol, side, quantity, avg_price, unrealized_pnl, realized_pnl, opened_at, updated_at
		 FROM positions WHERE account_id = ? AND symbol = ?`, accountID, symbol)
	var p model.Position
	var qty, avgPrice, uPnl, rPnl string
	if err := row.Scan(&p.ID, &p.AccountID, &p.Symbol, &p.Side, &qty, &avgPrice, &uPnl, &rPnl, &p.OpenedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Quantity = dec(qty)
	p.AvgPrice = dec(avgPrice)
	p.UnrealizedPnL = dec(uPnl)
	p.RealizedPnL = dec(rPnl)
	return &p, nil
}

func (r *Repository) ListPositions(ctx context.Context, accountID string) ([]model.Position, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, account_id, symbol, side, quantity, avg_price, unrealized_pnl, realized_pnl, opened_at, updated_at
		 FROM positions WHERE account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []model.Position
	for rows.Next() {
		var p model.Position
		var qty, avgPrice, uPnl, rPnl string
		if err := rows.Scan(&p.ID, &p.AccountID, &p.Symbol, &p.Side, &qty, &avgPrice, &uPnl, &rPnl, &p.OpenedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Quantity = dec(qty)
		p.AvgPrice = dec(avgPrice)
		p.UnrealizedPnL = dec(uPnl)
		p.RealizedPnL = dec(rPnl)
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

func (r *Repository) UpsertPosition(ctx context.Context, p model.Position) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO positions (id, account_id, symbol, side, quantity, avg_price, unrealized_pnl, realized_pnl, opened_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.AccountID, p.Symbol, p.Side,
		p.Quantity.String(), p.AvgPrice.String(), p.UnrealizedPnL.String(), p.RealizedPnL.String(),
		p.OpenedAt, p.UpdatedAt,
	)
	return err
}

func (r *Repository) SaveFill(ctx context.Context, f model.Fill) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fills (fill_id, order_id, symbol, side, price, quantity, fee, slippage, strategy, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		GenerateID(), f.OrderID, f.Symbol, f.Side,
		f.Price.String(), f.Quantity.String(), f.Fee.String(), f.Slippage.String(),
		f.Strategy, f.Timestamp,
	)
	return err
}

func (r *Repository) CreateBacktestRun(ctx context.Context, run model.BacktestRun) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO backtest_runs (id, config, status, ai_mode, result, compare_result, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, marshalJSON(run.Config), run.Status, run.AIMode,
		marshalJSON(run.Result), marshalJSON(run.CompareResult), run.Error, run.CreatedAt,
	)
	return err
}

func (r *Repository) UpdateBacktestResult(ctx context.Context, id string, result *model.BacktestResult, status, errStr string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE backtest_runs SET result = ?, status = ?, error = ?, completed_at = ? WHERE id = ?`,
		marshalJSON(result), status, errStr, time.Now().Unix(), id,
	)
	return err
}

func (r *Repository) ListBacktestRuns(ctx context.Context) ([]model.BacktestRun, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, config, status, ai_mode, result, error, created_at, completed_at
		 FROM backtest_runs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []model.BacktestRun
	for rows.Next() {
		var run model.BacktestRun
		var config, result, errStr string
		var completedAt *int64
		if err := rows.Scan(&run.ID, &config, &run.Status, &run.AIMode, &result, &errStr, &run.CreatedAt, &completedAt); err != nil {
			return nil, err
		}
		unmarshalJSON(config, &run.Config)
		if result != "" {
			var br model.BacktestResult
			unmarshalJSON(result, &br)
			run.Result = &br
		}
		run.Error = errStr
		if completedAt != nil {
			run.CompletedAt = *completedAt
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *Repository) GetBacktestRun(ctx context.Context, id string) (*model.BacktestRun, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, config, status, ai_mode, result, error, created_at, completed_at
		 FROM backtest_runs WHERE id = ?`, id)
	var run model.BacktestRun
	var config, result, errStr string
	var completedAt *int64
	if err := row.Scan(&run.ID, &config, &run.Status, &run.AIMode, &result, &errStr, &run.CreatedAt, &completedAt); err != nil {
		return nil, err
	}
	unmarshalJSON(config, &run.Config)
	if result != "" {
		var br model.BacktestResult
		unmarshalJSON(result, &br)
		run.Result = &br
	}
	run.Error = errStr
	if completedAt != nil {
		run.CompletedAt = *completedAt
	}
	return &run, nil
}

func (r *Repository) SaveSentimentResult(ctx context.Context, sr model.SentimentResult) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sentiment_results (id, text, score, label, keywords, summary, symbols, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sr.ID, sr.Text, sr.Score, sr.Label,
		marshalJSON(sr.Keywords), sr.Summary, marshalJSON(sr.Symbols), sr.Timestamp,
	)
	return err
}

func (r *Repository) GetSentimentResults(ctx context.Context, symbol string, limit int) ([]model.SentimentResult, error) {
	query := `SELECT id, text, score, label, keywords, summary, symbols, timestamp FROM sentiment_results`
	args := []any{}
	if symbol != "" {
		query += ` WHERE symbols LIKE ?`
		args = append(args, "%"+symbol+"%")
	}
	query += ` ORDER BY timestamp DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.SentimentResult
	for rows.Next() {
		var sr model.SentimentResult
		var keywords, symbols string
		if err := rows.Scan(&sr.ID, &sr.Text, &sr.Score, &sr.Label, &keywords, &sr.Summary, &symbols, &sr.Timestamp); err != nil {
			return nil, err
		}
		unmarshalJSON(keywords, &sr.Keywords)
		unmarshalJSON(symbols, &sr.Symbols)
		results = append(results, sr)
	}
	return results, rows.Err()
}

func (r *Repository) SaveRiskEvent(ctx context.Context, re model.RiskEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO risk_events (id, type, sentiment_score, action, reason, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		re.ID, re.Type, re.SentimentScore, re.Action, re.Reason, re.Timestamp,
	)
	return err
}

func (r *Repository) GetRiskEvents(ctx context.Context, limit int) ([]model.RiskEvent, error) {
	query := `SELECT id, type, sentiment_score, action, reason, timestamp FROM risk_events ORDER BY timestamp DESC`
	args := []any{}
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.RiskEvent
	for rows.Next() {
		var re model.RiskEvent
		if err := rows.Scan(&re.ID, &re.Type, &re.SentimentScore, &re.Action, &re.Reason, &re.Timestamp); err != nil {
			return nil, err
		}
		events = append(events, re)
	}
	return events, rows.Err()
}

func GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

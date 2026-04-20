package data

import (
	"context"
)

func (db *DB) Migrate(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS symbols (
			code        TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			market      TEXT NOT NULL,
			exchange    TEXT NOT NULL,
			type        TEXT NOT NULL,
			lot_size    TEXT NOT NULL DEFAULT '1',
			tick_size   TEXT NOT NULL DEFAULT '0.01',
			currency    TEXT NOT NULL DEFAULT 'USD',
			enabled     INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS ticks (
			symbol    TEXT NOT NULL,
			price     TEXT NOT NULL,
			volume    TEXT NOT NULL,
			high      TEXT NOT NULL,
			low       TEXT NOT NULL,
			open_price TEXT NOT NULL,
			source    TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT (unixepoch()),
			UNIQUE(symbol, timestamp, source)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ticks_symbol_ts ON ticks(symbol, timestamp)`,
		`CREATE TABLE IF NOT EXISTS candles (
			symbol    TEXT NOT NULL,
			interval  TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			open      TEXT NOT NULL,
			high      TEXT NOT NULL,
			low       TEXT NOT NULL,
			close     TEXT NOT NULL,
			volume    TEXT NOT NULL,
			source    TEXT NOT NULL,
			UNIQUE(symbol, interval, timestamp)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_candles_symbol_interval_ts ON candles(symbol, interval, timestamp)`,
		`CREATE TABLE IF NOT EXISTS signals (
			signal_id  TEXT PRIMARY KEY,
			symbol     TEXT NOT NULL,
			source     TEXT NOT NULL,
			type       TEXT NOT NULL,
			strength   REAL NOT NULL,
			price      TEXT NOT NULL,
			metadata   TEXT,
			timestamp  INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		)`,
		`CREATE INDEX IF NOT EXISTS idx_signals_symbol_ts ON signals(symbol, timestamp)`,
		`CREATE TABLE IF NOT EXISTS strategy_configs (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL UNIQUE,
			display_name TEXT,
			enabled      INTEGER NOT NULL DEFAULT 0,
			params       TEXT,
			symbols      TEXT,
			risk_config  TEXT,
			ai_config    TEXT,
			created_at   INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at   INTEGER NOT NULL DEFAULT (unixepoch())
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id          TEXT PRIMARY KEY,
			cash        TEXT NOT NULL,
			total_value TEXT NOT NULL,
			currency    TEXT NOT NULL DEFAULT 'USD',
			created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at  INTEGER NOT NULL DEFAULT (unixepoch())
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id              TEXT PRIMARY KEY,
			account_id      TEXT NOT NULL,
			symbol          TEXT NOT NULL,
			side            TEXT NOT NULL,
			type            TEXT NOT NULL,
			status          TEXT NOT NULL,
			quantity        TEXT NOT NULL,
			filled_qty      TEXT NOT NULL DEFAULT '0',
			price           TEXT NOT NULL,
			avg_fill_price  TEXT NOT NULL DEFAULT '0',
			fee             TEXT NOT NULL DEFAULT '0',
			slippage        TEXT NOT NULL DEFAULT '0',
			strategy        TEXT,
			reason          TEXT,
			created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
		)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_account_status ON orders(account_id, status)`,
		`CREATE TABLE IF NOT EXISTS positions (
			id               TEXT PRIMARY KEY,
			account_id       TEXT NOT NULL,
			symbol           TEXT NOT NULL,
			side             TEXT NOT NULL,
			quantity         TEXT NOT NULL,
			avg_price        TEXT NOT NULL,
			unrealized_pnl   TEXT NOT NULL DEFAULT '0',
			realized_pnl     TEXT NOT NULL DEFAULT '0',
			opened_at        INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at       INTEGER NOT NULL DEFAULT (unixepoch()),
			UNIQUE(account_id, symbol)
		)`,
		`CREATE TABLE IF NOT EXISTS fills (
			fill_id   TEXT PRIMARY KEY,
			order_id  TEXT NOT NULL,
			symbol    TEXT NOT NULL,
			side      TEXT NOT NULL,
			price     TEXT NOT NULL,
			quantity  TEXT NOT NULL,
			fee       TEXT NOT NULL DEFAULT '0',
			slippage  TEXT NOT NULL DEFAULT '0',
			strategy  TEXT,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fills_symbol_ts ON fills(symbol, timestamp)`,
		`CREATE TABLE IF NOT EXISTS backtest_runs (
			id             TEXT PRIMARY KEY,
			config         TEXT NOT NULL,
			status         TEXT NOT NULL,
			ai_mode        TEXT NOT NULL DEFAULT 'off',
			result         TEXT,
			compare_result TEXT,
			error          TEXT,
			created_at     INTEGER NOT NULL DEFAULT (unixepoch()),
			completed_at   INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS sentiment_results (
			id        TEXT PRIMARY KEY,
			text      TEXT NOT NULL,
			score     REAL NOT NULL,
			label     TEXT NOT NULL,
			keywords  TEXT,
			summary   TEXT,
			symbols   TEXT,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS risk_events (
			id              TEXT PRIMARY KEY,
			type            TEXT NOT NULL,
			sentiment_score REAL NOT NULL,
			action          TEXT NOT NULL,
			reason          TEXT,
			timestamp       INTEGER NOT NULL
		)`,
	}

	for _, m := range migrations {
		if _, err := db.ExecContext(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

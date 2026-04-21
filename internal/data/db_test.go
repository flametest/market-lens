package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDB_Migrate(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	tables := []string{"symbols", "ticks", "candles", "signals", "strategy_configs",
		"accounts", "orders", "positions", "fills", "backtest_runs",
		"sentiment_results", "risk_events"}

	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx,
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should exist", table)
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	err = db.Migrate(context.Background())
	require.NoError(t, err)
	return db
}

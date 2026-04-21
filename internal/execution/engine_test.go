package execution

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/model"
)

func newTestEnv(t *testing.T) (*Engine, *data.Repository) {
	t.Helper()
	db, err := data.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	require.NoError(t, db.Migrate(ctx))

	repo := data.NewRepository(db)
	bus := eventbus.NewInMemoryBus(nil)
	t.Cleanup(func() { bus.Close() })

	engine := NewEngine(repo, bus, nil)
	return engine, repo
}

func TestEngine_Start(t *testing.T) {
	engine, repo := newTestEnv(t)

	err := engine.Start(context.Background())
	require.NoError(t, err)

	account, err := repo.GetAccount(context.Background(), "paper_01")
	require.NoError(t, err)
	assert.Equal(t, "paper_01", account.ID)
	assert.True(t, account.Cash.Equal(decimal.NewFromInt(1000000)))
}

func TestEngine_SubmitOrder(t *testing.T) {
	engine, repo := newTestEnv(t)
	ctx := context.Background()

	// Create account
	now := time.Now().Unix()
	require.NoError(t, repo.CreateAccount(ctx, model.Account{
		ID: "paper_01", Cash: dec("100000"), TotalValue: dec("100000"),
		Currency: "USD", CreatedAt: now, UpdatedAt: now,
	}))

	t.Run("buy creates position and deducts cash", func(t *testing.T) {
		order, err := engine.SubmitOrder(ctx, model.OrderDecision{
			Symbol:   "AAPL",
			Side:     model.OrderBuy,
			Type:     model.OrderMarket,
			Quantity: dec("10"),
			Price:    dec("150"),
			Strategy: "test",
		})
		require.NoError(t, err)
		require.NotNil(t, order)
		assert.Equal(t, model.OrderFilled, order.Status)
		assert.Equal(t, "AAPL", order.Symbol)

		// Check cash deducted
		account, _ := repo.GetAccount(ctx, "paper_01")
		assert.True(t, account.Cash.LessThan(dec("100000")))

		// Check position created
		pos, err := repo.GetPosition(ctx, "paper_01", "AAPL")
		require.NoError(t, err)
		assert.True(t, pos.Quantity.Equal(dec("10")))
	})

	t.Run("sell closes position and adds cash", func(t *testing.T) {
		order, err := engine.SubmitOrder(ctx, model.OrderDecision{
			Symbol:   "AAPL",
			Side:     model.OrderSell,
			Type:     model.OrderMarket,
			Quantity: dec("10"),
			Price:    dec("160"),
			Strategy: "test",
		})
		require.NoError(t, err)
		require.NotNil(t, order)
		assert.Equal(t, model.OrderFilled, order.Status)

		// Check cash increased
		account, _ := repo.GetAccount(ctx, "paper_01")
		assert.True(t, account.Cash.GreaterThan(dec("0")))

		// Check position zeroed
		pos, _ := repo.GetPosition(ctx, "paper_01", "AAPL")
		assert.True(t, pos.Quantity.IsZero())
	})
}

func TestFundConservation(t *testing.T) {
	engine, repo := newTestEnv(t)
	ctx := context.Background()

	initialCash := dec("100000")
	now := time.Now().Unix()
	require.NoError(t, repo.CreateAccount(ctx, model.Account{
		ID: "paper_01", Cash: initialCash, TotalValue: initialCash,
		Currency: "USD", CreatedAt: now, UpdatedAt: now,
	}))

	// Buy 100 shares at $100
	buyOrder, err := engine.SubmitOrder(ctx, model.OrderDecision{
		Symbol: "TEST", Side: model.OrderBuy, Type: model.OrderMarket,
		Quantity: dec("100"), Price: dec("100"), Strategy: "test",
	})
	require.NoError(t, err)
	require.NotNil(t, buyOrder)

	account, _ := repo.GetAccount(ctx, "paper_01")
	pos, _ := repo.GetPosition(ctx, "paper_01", "TEST")

	// Cash + position value should approximate initial (minus fees/slippage)
	totalAfterBuy := account.Cash.Add(pos.AvgPrice.Mul(pos.Quantity))
	assert.True(t, totalAfterBuy.LessThanOrEqual(initialCash), "total should decrease due to fees")

	// Sell 100 shares at $100
	sellOrder, err := engine.SubmitOrder(ctx, model.OrderDecision{
		Symbol: "TEST", Side: model.OrderSell, Type: model.OrderMarket,
		Quantity: dec("100"), Price: dec("100"), Strategy: "test",
	})
	require.NoError(t, err)
	require.NotNil(t, sellOrder)

	account2, _ := repo.GetAccount(ctx, "paper_01")
	// After buy+sell at same price, cash should be less due to slippage + commission
	assert.True(t, account2.Cash.LessThan(initialCash), "cash should be less after round trip")
	assert.True(t, account2.Cash.GreaterThan(dec("90000")), "but not too much less")
}

func TestRejectInsufficientFunds(t *testing.T) {
	engine, repo := newTestEnv(t)
	ctx := context.Background()

	now := time.Now().Unix()
	require.NoError(t, repo.CreateAccount(ctx, model.Account{
		ID: "paper_01", Cash: dec("100"), TotalValue: dec("100"),
		Currency: "USD", CreatedAt: now, UpdatedAt: now,
	}))

	order, err := engine.SubmitOrder(ctx, model.OrderDecision{
		Symbol: "AAPL", Side: model.OrderBuy, Type: model.OrderMarket,
		Quantity: dec("10"), Price: dec("200"), Strategy: "test",
	})
	require.NoError(t, err)
	assert.Nil(t, order)
}

func TestRejectSellWithoutPosition(t *testing.T) {
	engine, repo := newTestEnv(t)
	ctx := context.Background()

	now := time.Now().Unix()
	require.NoError(t, repo.CreateAccount(ctx, model.Account{
		ID: "paper_01", Cash: dec("100000"), TotalValue: dec("100000"),
		Currency: "USD", CreatedAt: now, UpdatedAt: now,
	}))

	order, err := engine.SubmitOrder(ctx, model.OrderDecision{
		Symbol: "AAPL", Side: model.OrderSell, Type: model.OrderMarket,
		Quantity: dec("10"), Price: dec("150"), Strategy: "test",
	})
	require.NoError(t, err)
	assert.Nil(t, order)
}

func TestPartialSell(t *testing.T) {
	engine, repo := newTestEnv(t)
	ctx := context.Background()

	now := time.Now().Unix()
	require.NoError(t, repo.CreateAccount(ctx, model.Account{
		ID: "paper_01", Cash: dec("100000"), TotalValue: dec("100000"),
		Currency: "USD", CreatedAt: now, UpdatedAt: now,
	}))

	// Buy 100 shares
	_, err := engine.SubmitOrder(ctx, model.OrderDecision{
		Symbol: "AAPL", Side: model.OrderBuy, Type: model.OrderMarket,
		Quantity: dec("100"), Price: dec("100"), Strategy: "test",
	})
	require.NoError(t, err)

	// Sell 30 shares
	order, err := engine.SubmitOrder(ctx, model.OrderDecision{
		Symbol: "AAPL", Side: model.OrderSell, Type: model.OrderMarket,
		Quantity: dec("30"), Price: dec("110"), Strategy: "test",
	})
	require.NoError(t, err)
	require.NotNil(t, order)

	// Position should have 70 remaining
	pos, _ := repo.GetPosition(ctx, "paper_01", "AAPL")
	assert.True(t, pos.Quantity.Equal(dec("70")))
}

func TestSlippageApplication(t *testing.T) {
	engine, _ := newTestEnv(t)

	t.Run("buy adds slippage", func(t *testing.T) {
		price := dec("100.00")
		fillPrice := engine.applySlippage(price, model.OrderBuy)
		assert.True(t, fillPrice.GreaterThan(price))
	})

	t.Run("sell subtracts slippage", func(t *testing.T) {
		price := dec("100.00")
		fillPrice := engine.applySlippage(price, model.OrderSell)
		assert.True(t, fillPrice.LessThan(price))
	})
}

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

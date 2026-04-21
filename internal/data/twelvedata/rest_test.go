package twelvedata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"

	"github.com/flametest/market-lens/pkg/model"
)

func TestToTDInterval(t *testing.T) {
	tests := []struct {
		interval model.Interval
		want     string
	}{
		{model.Interval1Min, "1min"},
		{model.Interval5Min, "5min"},
		{model.Interval15Min, "15min"},
		{model.Interval30Min, "30min"},
		{model.Interval1Hour, "1h"},
		{model.Interval4Hour, "4h"},
		{model.Interval1Day, "1day"},
		{model.Interval1Week, "1week"},
		{model.Interval1Month, "1month"},
	}
	for _, tt := range tests {
		t.Run(string(tt.interval), func(t *testing.T) {
			assert.Equal(t, tt.want, toTDInterval(tt.interval))
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"2024-01-15 09:30:00"},
		{"2024-01-15 09:30"},
		{"2024-01-15"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTimestamp(tt.input)
			require.NoError(t, err)
			assert.NotZero(t, got)
		})
	}
}

func TestParseTimestamp_Invalid(t *testing.T) {
	_, err := parseTimestamp("not-a-date")
	assert.Error(t, err)
}

func TestS2d(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"198.42", "198.42"},
		{"0", "0"},
		{"150.5", "150.5"},
		{"", "0"},
		{"abc", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := s2d(tt.input)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

func TestRESTClient_FetchCandles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "symbol=AAPL")
		assert.Contains(t, r.URL.String(), "interval=1day")
		assert.Contains(t, r.URL.String(), "apikey=test-key")

		resp := map[string]any{
			"meta":   map[string]any{"symbol": "AAPL", "interval": "1day"},
			"values": []map[string]string{
				{"datetime": "2024-01-15", "open": "150.00", "high": "155.00", "low": "149.00", "close": "153.00", "volume": "1000000"},
				{"datetime": "2024-01-16", "open": "153.00", "high": "156.00", "low": "150.00", "close": "154.00", "volume": "1200000"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	candles, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 0, 0)
	require.NoError(t, err)
	require.Len(t, candles, 2)

	assert.Equal(t, "AAPL", candles[0].Symbol)
	assert.True(t, candles[0].Open.Equal(decimal.RequireFromString("150.00")))
	assert.True(t, candles[0].Close.Equal(decimal.RequireFromString("153.00")))
	assert.True(t, candles[0].Volume.Equal(decimal.RequireFromString("1000000")))
	assert.Equal(t, model.Interval1Day, candles[0].Interval)
	assert.Equal(t, "twelvedata", candles[0].Source)
	assert.NotZero(t, candles[0].Timestamp)
}

func TestRESTClient_FetchCandles_WithDateRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "start_date=")
		assert.Contains(t, r.URL.String(), "end_date=")

		resp := map[string]any{
			"values": []map[string]string{
				{"datetime": "2024-01-15 09:30:00", "open": "150.00", "high": "155.00", "low": "149.00", "close": "153.00", "volume": "500000"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	candles, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 1705276800000, 1705363200000)
	require.NoError(t, err)
	require.Len(t, candles, 1)
	assert.True(t, candles[0].Close.Equal(decimal.RequireFromString("153.00")))
}

func TestRESTClient_FetchCandles_ApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Invalid API key",
		})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "bad-key", slog.Default())
	_, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestRESTClient_FetchCandles_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	_, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 429")
}

func TestRESTClient_FetchCandles_EmptyValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"meta":   map[string]any{"symbol": "AAPL"},
			"values": []map[string]string{},
		})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	candles, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, candles)
}

func TestRESTClient_FetchQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "symbol=AAPL")

		resp := map[string]any{
			"symbol":         "AAPL",
			"name":           "Apple Inc.",
			"open":           "195.50",
			"high":           "199.62",
			"low":            "195.18",
			"close":          "198.42",
			"volume":         "52436789",
			"previous_close": "195.18",
			"change":         "3.24",
			"percent_change": "1.66",
			"timestamp":      1700000000,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	tick, err := client.FetchQuote(context.Background(), "AAPL")
	require.NoError(t, err)

	assert.Equal(t, "AAPL", tick.Symbol)
	assert.True(t, tick.Price.Equal(decimal.RequireFromString("198.42")))
	assert.True(t, tick.Open.Equal(decimal.RequireFromString("195.50")))
	assert.True(t, tick.High.Equal(decimal.RequireFromString("199.62")))
	assert.True(t, tick.Low.Equal(decimal.RequireFromString("195.18")))
	assert.True(t, tick.Change.Equal(decimal.RequireFromString("3.24")))
	assert.True(t, tick.ChangePercent.Equal(decimal.RequireFromString("1.66")))
	assert.Equal(t, int64(1700000000000), tick.Timestamp)
	assert.Equal(t, "twelvedata", tick.Source)
}

func TestRESTClient_FetchQuote_ApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Rate limit exceeded",
		})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	_, err := client.FetchQuote(context.Background(), "AAPL")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rate limit exceeded")
}

func TestRESTClient_FetchQuote_ZeroPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"symbol":    "BAD",
			"close":     "0",
			"open":      "0",
			"high":      "0",
			"low":       "0",
			"volume":    "0",
			"change":    "0",
			"percent_change": "0",
			"timestamp": 1700000000,
		})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	_, err := client.FetchQuote(context.Background(), "BAD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no quote data")
}

func TestRESTClient_SearchSymbols(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "symbol=Apple")

		resp := map[string]any{
			"data": []map[string]string{
				{"symbol": "AAPL", "instrument_name": "Apple Inc.", "exchange": "NASDAQ", "instrument_type": "Common Stock", "country": "United States", "currency": "USD"},
				{"symbol": "AAPL34", "instrument_name": "Apple BDR", "exchange": "BVMF", "instrument_type": "Common Stock", "country": "Brazil", "currency": "BRL"},
				{"symbol": "AAPL.O", "instrument_name": "Apple Option", "exchange": "OPRA", "instrument_type": "Option", "country": "United States", "currency": "USD"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	symbols, err := client.SearchSymbols(context.Background(), "Apple")
	require.NoError(t, err)

	assert.Len(t, symbols, 2) // Option is filtered out
	assert.Equal(t, "AAPL", symbols[0].Code)
	assert.Equal(t, "Apple Inc.", symbols[0].Name)
	assert.Equal(t, "NASDAQ", symbols[0].Exchange)
	assert.Equal(t, "USD", symbols[0].Currency)
	assert.Equal(t, model.MarketUSStock, symbols[0].Market)
	assert.True(t, symbols[0].Enabled)

	assert.Equal(t, "AAPL34", symbols[1].Code)
}

func TestRESTClient_SearchSymbols_WithETF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]string{
				{"symbol": "SPY", "instrument_name": "SPDR S&P 500 ETF", "exchange": "NYSE", "instrument_type": "ETF", "currency": "USD"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	symbols, err := client.SearchSymbols(context.Background(), "SPY")
	require.NoError(t, err)
	require.Len(t, symbols, 1)
	assert.Equal(t, "SPY", symbols[0].Code)
}

func TestRESTClient_SearchSymbols_ApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Invalid symbol",
		})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	_, err := client.SearchSymbols(context.Background(), "!!!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid symbol")
}

func TestRESTClient_APIKeyInURL(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		json.NewEncoder(w).Encode(map[string]any{"values": []map[string]string{}})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "my-secret-key", slog.Default())
	_, _ = client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 0, 0)
	assert.Contains(t, capturedURL, "apikey=my-secret-key")
}

func TestProvider_Name(t *testing.T) {
	p := NewProvider("https://api.twelvedata.com", "wss://ws.twelvedata.com/v1/quotes/price", "test-key", slog.Default())
	assert.Equal(t, "twelvedata", p.Name())
}

func TestProvider_SubscribeUnsubscribe(t *testing.T) {
	// Use a mock WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Keep connection alive, read messages
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:] // http:// -> ws://
	p := NewProvider("https://api.twelvedata.com", wsURL, "test-key", slog.Default())
	ctx := context.Background()

	require.NoError(t, p.Connect(ctx))

	require.NoError(t, p.Subscribe([]string{"AAPL", "GOOGL"}))
	p.ws.mu.Lock()
	assert.True(t, p.ws.symbols["AAPL"])
	assert.True(t, p.ws.symbols["GOOGL"])
	p.ws.mu.Unlock()

	require.NoError(t, p.Unsubscribe([]string{"AAPL"}))
	p.ws.mu.Lock()
	assert.False(t, p.ws.symbols["AAPL"])
	assert.True(t, p.ws.symbols["GOOGL"])
	p.ws.mu.Unlock()

	require.NoError(t, p.Close())
}

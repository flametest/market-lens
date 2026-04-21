package finnhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"

	"github.com/flametest/market-lens/pkg/model"
)

func TestToResolution(t *testing.T) {
	tests := []struct {
		interval model.Interval
		want     string
	}{
		{model.Interval1Min, "1"},
		{model.Interval5Min, "5"},
		{model.Interval15Min, "15"},
		{model.Interval30Min, "30"},
		{model.Interval1Hour, "60"},
		{model.Interval4Hour, "240"},
		{model.Interval1Day, "D"},
		{model.Interval1Week, "W"},
		{model.Interval1Month, "M"},
	}
	for _, tt := range tests {
		t.Run(string(tt.interval), func(t *testing.T) {
			assert.Equal(t, tt.want, toFinnhubResolution(tt.interval))
		})
	}
}

func TestRESTClient_FetchCandles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("X-Finnhub-Token"))
		assert.Equal(t, "AAPL", r.URL.Query().Get("symbol"))
		assert.Equal(t, "D", r.URL.Query().Get("resolution"))

		resp := map[string]any{
			"s": []string{"ok"},
			"t": []float64{1700000000, 1700086400},
			"o": []float64{150.0, 151.0},
			"h": []float64{155.0, 156.0},
			"l": []float64{149.0, 150.0},
			"c": []float64{153.0, 154.0},
			"v": []float64{1000000, 1200000},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	candles, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 1700000000, 1700172800)
	require.NoError(t, err)
	require.Len(t, candles, 2)

	assert.Equal(t, "AAPL", candles[0].Symbol)
	assert.True(t, candles[0].Close.Equal(f2d(153.0)))
	assert.True(t, candles[1].Close.Equal(f2d(154.0)))
	assert.Equal(t, int64(1700000000000), candles[0].Timestamp)
}

func TestRESTClient_FetchCandles_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"s": []string{"no_data"}})
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	candles, err := client.FetchCandles(context.Background(), "AAPL", model.Interval1Day, 0, 0)
	require.NoError(t, err)
	assert.Nil(t, candles)
}

func TestRESTClient_FetchQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"c": 198.42, "h": 199.62, "l": 195.18, "o": 195.50,
			"pc": 195.18, "t": 1700000000,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	tick, err := client.FetchQuote(context.Background(), "AAPL")
	require.NoError(t, err)

	assert.Equal(t, "AAPL", tick.Symbol)
	assert.True(t, tick.Price.Equal(f2d(198.42)))
	assert.True(t, tick.Change.GreaterThan(f2d(0)))
}

func TestRESTClient_SearchSymbols(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"count": 2,
			"result": []map[string]string{
				{"symbol": "AAPL", "description": "Apple Inc.", "type": "Common Stock"},
				{"symbol": "AAPL.X", "description": "Apple Option", "type": "Option"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRESTClient(server.URL, "test-key", slog.Default())
	symbols, err := client.SearchSymbols(context.Background(), "AAPL")
	require.NoError(t, err)
	assert.Len(t, symbols, 1)
	assert.Equal(t, "AAPL", symbols[0].Code)
	assert.Equal(t, "Apple Inc.", symbols[0].Name)
}

func TestF2d(t *testing.T) {
	tests := []struct {
		input float64
		str   string
	}{
		{150.5, "150.5"},
		{0.0, "0"},
		{198.42, "198.42"},
	}
	for _, tt := range tests {
		d := f2d(tt.input)
		assert.Equal(t, tt.str, d.String())
	}
}

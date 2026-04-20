package finnhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/pkg/model"
)

type RESTClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewRESTClient(baseURL, apiKey string, logger *slog.Logger) *RESTClient {
	return &RESTClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (c *RESTClient) FetchCandles(ctx context.Context, symbol string, interval model.Interval, from, to int64) ([]model.Candle, error) {
	resolution := toFinnhubResolution(interval)
	url := fmt.Sprintf("%s/stock/candle?symbol=%s&resolution=%s&from=%d&to=%d",
		c.baseURL, symbol, resolution, from, to)

	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		S []any     `json:"s"`
		T []float64 `json:"t"`
		O []float64 `json:"o"`
		H []float64 `json:"h"`
		L []float64 `json:"l"`
		C []float64 `json:"c"`
		V []float64 `json:"v"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse candles response: %w", err)
	}

	if len(resp.S) == 0 || fmt.Sprintf("%v", resp.S[0]) == "no_data" {
		return nil, nil
	}

	candles := make([]model.Candle, 0, len(resp.T))
	for i := range resp.T {
		candles = append(candles, model.Candle{
			Symbol:    symbol,
			Open:      f2d(resp.O[i]),
			High:      f2d(resp.H[i]),
			Low:       f2d(resp.L[i]),
			Close:     f2d(resp.C[i]),
			Volume:    f2d(resp.V[i]),
			Timestamp: int64(resp.T[i]) * 1000,
			Interval:  interval,
			Source:    "finnhub",
		})
	}
	return candles, nil
}

func (c *RESTClient) FetchQuote(ctx context.Context, symbol string) (*model.Tick, error) {
	url := fmt.Sprintf("%s/quote?symbol=%s", c.baseURL, symbol)

	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		CurrentPrice float64 `json:"c"`
		High         float64 `json:"h"`
		Low          float64 `json:"l"`
		Open         float64 `json:"o"`
		PrevClose    float64 `json:"pc"`
		Timestamp    float64 `json:"t"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse quote response: %w", err)
	}

	if resp.CurrentPrice == 0 {
		return nil, fmt.Errorf("no quote data for %s", symbol)
	}

	change := resp.CurrentPrice - resp.PrevClose
	var changePct float64
	if resp.PrevClose > 0 {
		changePct = (change / resp.PrevClose) * 100
	}

	return &model.Tick{
		Symbol:        symbol,
		Price:         f2d(resp.CurrentPrice),
		High:          f2d(resp.High),
		Low:           f2d(resp.Low),
		Open:          f2d(resp.Open),
		Change:        f2d(change),
		ChangePercent: f2d(changePct),
		Volume:        decimal.Zero,
		Timestamp:     int64(resp.Timestamp) * 1000,
		Source:        "finnhub",
	}, nil
}

func (c *RESTClient) SearchSymbols(ctx context.Context, query string) ([]model.SymbolInfo, error) {
	url := fmt.Sprintf("%s/search?q=%s", c.baseURL, query)

	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Count  int `json:"count"`
		Result []struct {
			Symbol      string `json:"symbol"`
			Description string `json:"description"`
			Type        string `json:"type"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	symbols := make([]model.SymbolInfo, 0, len(resp.Result))
	for _, r := range resp.Result {
		if r.Type != "Common Stock" {
			continue
		}
		symbols = append(symbols, model.SymbolInfo{
			Code:     r.Symbol,
			Name:     r.Description,
			Market:   model.MarketUSStock,
			Exchange: "NASDAQ",
			Type:     "stock",
			Currency: "USD",
			Enabled:  true,
		})
	}
	return symbols, nil
}

func (c *RESTClient) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Finnhub-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

func toFinnhubResolution(interval model.Interval) string {
	switch interval {
	case model.Interval1Min:
		return "1"
	case model.Interval5Min:
		return "5"
	case model.Interval15Min:
		return "15"
	case model.Interval30Min:
		return "30"
	case model.Interval1Hour:
		return "60"
	case model.Interval4Hour:
		return "240"
	case model.Interval1Day:
		return "D"
	case model.Interval1Week:
		return "W"
	case model.Interval1Month:
		return "M"
	default:
		return "D"
	}
}

func f2d(f float64) decimal.Decimal {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	d, _ := decimal.NewFromString(s)
	return d
}

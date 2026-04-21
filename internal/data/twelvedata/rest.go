package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

func (c *RESTClient) FetchCandles(ctx context.Context, symbol string, interval model.Interval, start, end int64) ([]model.Candle, error) {
	tdInterval := toTDInterval(interval)
	reqURL := fmt.Sprintf("%s/time_series?symbol=%s&interval=%s&outputsize=5000&apikey=%s",
		c.baseURL, url.QueryEscape(symbol), tdInterval, c.apiKey)

	if start > 0 && end > 0 {
		startDate := time.UnixMilli(start).Format("2006-01-02 15:04:05")
		endDate := time.UnixMilli(end).Format("2006-01-02 15:04:05")
		reqURL = fmt.Sprintf("%s/time_series?symbol=%s&interval=%s&start_date=%s&end_date=%s&outputsize=5000&apikey=%s",
			c.baseURL, url.QueryEscape(symbol), tdInterval, url.QueryEscape(startDate), url.QueryEscape(endDate), c.apiKey)
	}

	body, err := c.get(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []struct {
			Datetime string `json:"datetime"`
			Open     string `json:"open"`
			High     string `json:"high"`
			Low      string `json:"low"`
			Close    string `json:"close"`
			Volume   string `json:"volume"`
		} `json:"values"`
		Status string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse candles response: %w", err)
	}

	if resp.Status == "error" {
		return nil, fmt.Errorf("twelvedata error: %s", resp.Message)
	}

	candles := make([]model.Candle, 0, len(resp.Values))
	for _, v := range resp.Values {
		ts, err := parseTimestamp(v.Datetime)
		if err != nil {
			continue
		}
		candles = append(candles, model.Candle{
			Symbol:    symbol,
			Open:      s2d(v.Open),
			High:      s2d(v.High),
			Low:       s2d(v.Low),
			Close:     s2d(v.Close),
			Volume:    s2d(v.Volume),
			Timestamp: ts,
			Interval:  interval,
			Source:    "twelvedata",
		})
	}
	return candles, nil
}

func (c *RESTClient) FetchQuote(ctx context.Context, symbol string) (*model.Tick, error) {
	reqURL := fmt.Sprintf("%s/quote?symbol=%s&apikey=%s", c.baseURL, url.QueryEscape(symbol), c.apiKey)

	body, err := c.get(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Symbol        string `json:"symbol"`
		Name          string `json:"name"`
		Open          string `json:"open"`
		High          string `json:"high"`
		Low           string `json:"low"`
		Close         string `json:"close"`
		Volume        string `json:"volume"`
		PreviousClose string `json:"previous_close"`
		Change        string `json:"change"`
		PercentChange string `json:"percent_change"`
		Timestamp     int64  `json:"timestamp"`
		Status        string `json:"status"`
		Message       string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse quote response: %w", err)
	}

	if resp.Status == "error" {
		return nil, fmt.Errorf("twelvedata error: %s", resp.Message)
	}

	closePrice := s2d(resp.Close)
	if closePrice.IsZero() {
		return nil, fmt.Errorf("no quote data for %s", symbol)
	}

	change := s2d(resp.Change)
	changePct := s2d(resp.PercentChange)

	return &model.Tick{
		Symbol:        symbol,
		Price:         closePrice,
		High:          s2d(resp.High),
		Low:           s2d(resp.Low),
		Open:          s2d(resp.Open),
		Volume:        s2d(resp.Volume),
		Change:        change,
		ChangePercent: changePct,
		Timestamp:     resp.Timestamp * 1000,
		Source:        "twelvedata",
	}, nil
}

func (c *RESTClient) SearchSymbols(ctx context.Context, query string) ([]model.SymbolInfo, error) {
	reqURL := fmt.Sprintf("%s/symbol_search?symbol=%s&outputsize=10&apikey=%s",
		c.baseURL, url.QueryEscape(query), c.apiKey)

	body, err := c.get(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			Symbol        string `json:"symbol"`
			InstrumentName string `json:"instrument_name"`
			Exchange      string `json:"exchange"`
			InstrumentType string `json:"instrument_type"`
			Country       string `json:"country"`
			Currency      string `json:"currency"`
		} `json:"data"`
		Status string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	if resp.Status == "error" {
		return nil, fmt.Errorf("twelvedata error: %s", resp.Message)
	}

	symbols := make([]model.SymbolInfo, 0, len(resp.Data))
	for _, r := range resp.Data {
		if r.InstrumentType != "Common Stock" && r.InstrumentType != "ETF" {
			continue
		}
		symbols = append(symbols, model.SymbolInfo{
			Code:     r.Symbol,
			Name:     r.InstrumentName,
			Market:   model.MarketUSStock,
			Exchange: r.Exchange,
			Type:     "stock",
			Currency: r.Currency,
			Enabled:  true,
		})
	}
	return symbols, nil
}

func (c *RESTClient) get(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "market-lens/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twelvedata api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

func toTDInterval(interval model.Interval) string {
	switch interval {
	case model.Interval1Min:
		return "1min"
	case model.Interval5Min:
		return "5min"
	case model.Interval15Min:
		return "15min"
	case model.Interval30Min:
		return "30min"
	case model.Interval1Hour:
		return "1h"
	case model.Interval4Hour:
		return "4h"
	case model.Interval1Day:
		return "1day"
	case model.Interval1Week:
		return "1week"
	case model.Interval1Month:
		return "1month"
	default:
		return "1day"
	}
}

func parseTimestamp(datetime string) (int64, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, datetime); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("cannot parse timestamp: %s", datetime)
}

func s2d(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

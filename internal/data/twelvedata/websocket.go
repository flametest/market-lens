package twelvedata

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/pkg/model"
)

type WSClient struct {
	url     string
	apiKey  string
	conn    *websocket.Conn
	mu      sync.Mutex
	symbols map[string]bool
	onTick  func(model.Tick)
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	ready   chan struct{}
}

func NewWSClient(baseURL, apiKey string, logger *slog.Logger) *WSClient {
	return &WSClient{
		url:     baseURL,
		apiKey:  apiKey,
		symbols: make(map[string]bool),
		logger:  logger,
		ready:   make(chan struct{}),
	}
}

func (w *WSClient) Connect(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.ready = make(chan struct{})

	url := w.url + "?apikey=" + w.apiKey
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	w.conn = conn

	go w.readLoop()
	go w.pingLoop()

	close(w.ready)
	return nil
}

func (w *WSClient) Subscribe(symbols []string) error {
	w.waitReady()

	w.mu.Lock()
	for _, s := range symbols {
		w.symbols[s] = true
	}
	w.mu.Unlock()

	return w.sendAction("subscribe", symbols)
}

func (w *WSClient) Unsubscribe(symbols []string) error {
	w.waitReady()

	w.mu.Lock()
	for _, s := range symbols {
		delete(w.symbols, s)
	}
	w.mu.Unlock()

	return w.sendAction("unsubscribe", symbols)
}

func (w *WSClient) SetOnTick(fn func(model.Tick)) {
	w.onTick = fn
}

func (w *WSClient) Close() error {
	if w.cancel != nil {
		w.cancel()
	}
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

func (w *WSClient) sendAction(action string, symbols []string) error {
	msg := struct {
		Action string `json:"action"`
		Params struct {
			Symbols string `json:"symbols"`
		} `json:"params"`
	}{
		Action: action,
	}
	for i, s := range symbols {
		if i > 0 {
			msg.Params.Symbols += ","
		}
		msg.Params.Symbols += s
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn == nil {
		return nil
	}
	return w.conn.WriteJSON(msg)
}

func (w *WSClient) readLoop() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		_, message, err := w.conn.ReadMessage()
		if err != nil {
			w.logger.Debug("ws read error", slog.String("error", err.Error()))
			if w.ctx.Err() != nil {
				return
			}
			w.reconnect()
			return
		}

		var ev struct {
			Event      string `json:"event"`
			Symbol     string `json:"symbol"`
			Price      string `json:"price"`
			DayVolume  string `json:"day_volume"`
			Timestamp  int64  `json:"timestamp"`
			Status     string `json:"status"`
		}
		if err := json.Unmarshal(message, &ev); err != nil {
			continue
		}

		if ev.Event == "price" && ev.Symbol != "" {
			if w.onTick != nil {
				price := s2d(ev.Price)
				if price.IsZero() {
					continue
				}
				tick := model.Tick{
					Symbol:    ev.Symbol,
					Price:     price,
					Volume:    s2d(ev.DayVolume),
					Timestamp: ev.Timestamp * 1000,
					Source:    "twelvedata",
				}
				w.onTick(tick)
			}
		}
	}
}

func (w *WSClient) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			if w.conn != nil {
				w.conn.WriteMessage(websocket.PingMessage, nil)
			}
			w.mu.Unlock()
		}
	}
}

func (w *WSClient) reconnect() {
	backoff := time.Second

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		w.logger.Info("reconnecting websocket", slog.Duration("backoff", backoff))
		time.Sleep(backoff)

		url := w.url + "?apikey=" + w.apiKey
		conn, _, err := websocket.DefaultDialer.DialContext(w.ctx, url, nil)
		if err != nil {
			w.logger.Error("ws reconnect failed", slog.String("error", err.Error()))
			backoff = backoff * 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			continue
		}

		w.mu.Lock()
		w.conn = conn
		var symbols []string
		for s := range w.symbols {
			symbols = append(symbols, s)
		}
		w.mu.Unlock()

		if len(symbols) > 0 {
			w.sendAction("subscribe", symbols)
		}

		go w.readLoop()
		return
	}
}

func (w *WSClient) waitReady() {
	select {
	case <-w.ready:
	case <-time.After(10 * time.Second):
	}
}

// Ensure price decimal is accessible
var _ = decimal.Decimal{}

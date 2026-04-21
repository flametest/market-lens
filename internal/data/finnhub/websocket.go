package finnhub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

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

func NewWSClient(url, apiKey string, logger *slog.Logger) *WSClient {
	return &WSClient{
		url:     url,
		apiKey:  apiKey,
		symbols: make(map[string]bool),
		logger:  logger,
		ready:   make(chan struct{}),
	}
}

func (w *WSClient) Connect(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.ready = make(chan struct{})

	url := w.url + "?token=" + w.apiKey
	conn, _, err := websocket.DefaultDialer.DialContext(w.ctx, url, nil)
	if err != nil {
		return err
	}
	w.conn = conn

	go w.readLoop()
	go w.pingLoop()

	close(w.ready)
	w.logger.Info("finnhub websocket connected")
	return nil
}

func (w *WSClient) Subscribe(symbols []string) error {
	<-w.ready
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, s := range symbols {
		w.symbols[s] = true
		msg, _ := json.Marshal(map[string]string{"type": "subscribe", "symbol": s})
		if err := w.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return err
		}
	}
	return nil
}

func (w *WSClient) Unsubscribe(symbols []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, s := range symbols {
		delete(w.symbols, s)
		msg, _ := json.Marshal(map[string]string{"type": "unsubscribe", "symbol": s})
		if err := w.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return err
		}
	}
	return nil
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

func (w *WSClient) readLoop() {
	defer func() {
		w.logger.Info("finnhub websocket read loop ended")
	}()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		_, message, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				w.logger.Error("websocket closed unexpectedly", slog.String("error", err.Error()))
				w.reconnect()
				return
			}
			return
		}

		w.handleMessage(message)
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
			err := w.conn.WriteMessage(websocket.PingMessage, nil)
			w.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (w *WSClient) handleMessage(message []byte) {
	var msg struct {
		Type string `json:"type"`
		Data []struct {
			P float64 `json:"p"`
			S string  `json:"s"`
			T float64 `json:"t"`
			V float64 `json:"v"`
		} `json:"data"`
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	if msg.Type != "trade" || len(msg.Data) == 0 {
		return
	}

	for _, d := range msg.Data {
		tick := model.Tick{
			Symbol:    d.S,
			Price:     f2d(d.P),
			Volume:    f2d(d.V),
			Timestamp: int64(d.T) * 1000,
			Source:    "finnhub",
		}
		if w.onTick != nil {
			w.onTick(tick)
		}
	}
}

func (w *WSClient) reconnect() {
	backoff := time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		w.logger.Info("reconnecting finnhub websocket", slog.Duration("backoff", backoff))
		time.Sleep(backoff)

		url := w.url + "?token=" + w.apiKey
		conn, _, err := websocket.DefaultDialer.DialContext(w.ctx, url, nil)
		if err != nil {
			w.logger.Error("reconnect failed", slog.String("error", err.Error()))
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		w.conn = conn
		w.logger.Info("finnhub websocket reconnected")

		w.mu.Lock()
		for s := range w.symbols {
			msg, _ := json.Marshal(map[string]string{"type": "subscribe", "symbol": s})
			w.conn.WriteMessage(websocket.TextMessage, msg)
		}
		w.mu.Unlock()

		go w.readLoop()
		return
	}
}

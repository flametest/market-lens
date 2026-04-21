package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	register   chan *wsClient
	unregister chan *wsClient
	broadcast  chan []byte
	logger     *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*wsClient]struct{}),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		broadcast:  make(chan []byte, 256),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Info("ws client connected", slog.Int("total", len(h.clients)))
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Info("ws client disconnected", slog.Int("total", len(h.clients)))
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(msgType string, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg := wsMessage{Type: msgType, Data: dataBytes}
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- payload:
	default:
		if h.logger != nil {
			h.logger.Warn("broadcast channel full, dropping message")
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *wsClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func HandleWebSocket(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &wsClient{
			conn: conn,
			send: make(chan []byte, 256),
			hub:  hub,
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flametest/market-lens/internal/eventbus"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer func() {
		// Signal hub to stop by closing broadcast channel indirectly
		for hub.ClientCount() > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}()

	server := httptest.NewServer(HandleWebSocket(hub))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	t.Run("client receives broadcast messages", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Wait for registration
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())

		// Broadcast a message
		hub.Broadcast("tick", map[string]string{"symbol": "AAPL", "price": "150.00"})

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)

		var received wsMessage
		require.NoError(t, json.Unmarshal(msg, &received))
		assert.Equal(t, "tick", received.Type)

		var data map[string]string
		require.NoError(t, json.Unmarshal(received.Data, &data))
		assert.Equal(t, "AAPL", data["symbol"])
		assert.Equal(t, "150.00", data["price"])
	})

	t.Run("multiple clients receive messages", func(t *testing.T) {
		conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn1.Close()

		conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn2.Close()

		time.Sleep(50 * time.Millisecond)

		hub.Broadcast("signal", map[string]string{"type": "BUY"})

		for i, conn := range []*websocket.Conn{conn1, conn2} {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, msg, err := conn.ReadMessage()
			require.NoError(t, err, "client %d should receive message", i)

			var received wsMessage
			require.NoError(t, json.Unmarshal(msg, &received))
			assert.Equal(t, "signal", received.Type)
		}
	})

	t.Run("client disconnect reduces count", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		initialCount := hub.ClientCount()

		conn.Close()
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, initialCount-1, hub.ClientCount())
	})
}

func TestBridge(t *testing.T) {
	t.Run("bridge forwards events to hub", func(t *testing.T) {
		hub := NewHub(nil)
		go hub.Run()

			bus := eventbus.NewInMemoryBus(nil)
		bridge := NewBridge(hub, bus, nil)
		bridge.Start()

		server := httptest.NewServer(HandleWebSocket(hub))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		time.Sleep(50 * time.Millisecond)

		// Publish an event to the bus
		bus.Publish("market.tick", map[string]string{"symbol": "AAPL"})

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)

		var received wsMessage
		require.NoError(t, json.Unmarshal(msg, &received))
		assert.Equal(t, "tick", received.Type)
	})
}

func TestWsMessageSerialization(t *testing.T) {
	t.Run("message serializes correctly", func(t *testing.T) {
		hub := NewHub(nil)

		data := map[string]any{
			"symbol": "GOOGL",
			"price":  2850.50,
		}

		dataBytes, _ := json.Marshal(data)
		msg := wsMessage{Type: "candle", Data: dataBytes}
		payload, err := json.Marshal(msg)
		require.NoError(t, err)

		assert.Contains(t, string(payload), `"type":"candle"`)
		assert.Contains(t, string(payload), `"symbol":"GOOGL"`)

		_ = hub // just to avoid unused var
	})
}

// eventbus import used above

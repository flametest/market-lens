package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		content := `
server:
  port: 9090
  cors_origins: ["http://localhost:3000"]
database:
  path: "./test.db"
finnhub:
  api_key: "test-key"
  websocket_url: "wss://ws.test.io"
  base_url: "https://api.test.io"
ai:
  provider: "openai"
  model: "gpt-4"
  api_key: "sk-test"
  max_tokens: 500
  temperature: 0.3
trading:
  initial_cash: "50000.00"
  commission_rate: "0.002"
  slippage_rate: "0.001"
  currency: "USD"
`
		err := os.WriteFile(cfgPath, []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := Load(cfgPath)
		require.NoError(t, err)

		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, []string{"http://localhost:3000"}, cfg.Server.CORSOrigins)
		assert.Equal(t, "./test.db", cfg.Database.Path)
		assert.Equal(t, "test-key", cfg.Finnhub.APIKey)
		assert.Equal(t, "wss://ws.test.io", cfg.Finnhub.WebSocketURL)
		assert.Equal(t, "USD", cfg.Trading.Currency)
	})

	t.Run("env var substitution", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		content := `
server:
  port: 8081
database:
  path: "./data/test.db"
finnhub:
  api_key: "${TEST_API_KEY}"
ai:
  provider: "openai"
trading:
  commission_rate: "0.001"
`
		os.Setenv("TEST_API_KEY", "env-key-123")
		defer os.Unsetenv("TEST_API_KEY")

		err := os.WriteFile(cfgPath, []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := Load(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, "env-key-123", cfg.Finnhub.APIKey)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := Load("/nonexistent/config.yaml")
		assert.Error(t, err)
	})

	t.Run("defaults applied", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		err := os.WriteFile(cfgPath, []byte("server:\n  port: 0\n"), 0644)
		require.NoError(t, err)

		cfg, err := Load(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, 8081, cfg.Server.Port)
		assert.Equal(t, "./data/market-lens.db", cfg.Database.Path)
		assert.Equal(t, "USD", cfg.Trading.Currency)
		assert.Equal(t, "100000.00", cfg.Trading.InitialCash)
	})
}

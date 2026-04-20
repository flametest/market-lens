package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Finnhub  FinnhubConfig  `yaml:"finnhub"`
	AI       AIConfig       `yaml:"ai"`
	Trading  TradingConfig  `yaml:"trading"`
}

type ServerConfig struct {
	Port        int      `yaml:"port"`
	CORSOrigins []string `yaml:"cors_origins"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type FinnhubConfig struct {
	APIKey       string `yaml:"api_key"`
	WebSocketURL string `yaml:"websocket_url"`
	BaseURL      string `yaml:"base_url"`
}

type AIConfig struct {
	Provider     string  `yaml:"provider"`
	Model        string  `yaml:"model"`
	APIKey       string  `yaml:"api_key"`
	MaxTokens    int     `yaml:"max_tokens"`
	Temperature  float64 `yaml:"temperature"`
}

type TradingConfig struct {
	InitialCash    string `yaml:"initial_cash"`
	CommissionRate string `yaml:"commission_rate"`
	SlippageRate   string `yaml:"slippage_rate"`
	Currency       string `yaml:"currency"`
}

var envPattern = regexp.MustCompile(`\$\{(\w+)\}`)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	expanded := envPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		key := envPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return ""
	})

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 {
		c.Server.Port = 8081
	}
	if c.Database.Path == "" {
		c.Database.Path = "./data/market-lens.db"
	}
	if c.Finnhub.BaseURL == "" {
		c.Finnhub.BaseURL = "https://finnhub.io/api/v1"
	}
	if c.Finnhub.WebSocketURL == "" {
		c.Finnhub.WebSocketURL = "wss://ws.finnhub.io"
	}
	if c.Trading.Currency == "" {
		c.Trading.Currency = "USD"
	}
	if c.Trading.InitialCash == "" {
		c.Trading.InitialCash = "100000.00"
	}
	if _, err := strconv.ParseFloat(c.Trading.CommissionRate, 64); c.Trading.CommissionRate != "" && err != nil {
		return fmt.Errorf("invalid commission_rate: %w", err)
	}
	if _, err := strconv.ParseFloat(c.Trading.SlippageRate, 64); c.Trading.SlippageRate != "" && err != nil {
		return fmt.Errorf("invalid slippage_rate: %w", err)
	}
	return nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/flametest/market-lens/internal/ai"
	"github.com/flametest/market-lens/internal/analysis"
	"github.com/flametest/market-lens/internal/api"
	"github.com/flametest/market-lens/internal/backtest"
	"github.com/flametest/market-lens/internal/config"
	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/data/finnhub"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/internal/execution"
	"github.com/flametest/market-lens/internal/strategy"
	"github.com/flametest/market-lens/internal/strategy/risk"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		logger.Error("failed to create data directory", slog.String("error", err.Error()))
		os.Exit(1)
	}

	db, err := data.Open(cfg.Database.Path)
	if err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		logger.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("database migrated")

	bus := eventbus.NewInMemoryBus(logger)
	defer bus.Close()

	repo := data.NewRepository(db)

	provider := finnhub.NewProvider(
		cfg.Finnhub.WebSocketURL,
		cfg.Finnhub.BaseURL,
		cfg.Finnhub.APIKey,
		logger,
	)
	mgr := data.NewManager(provider, db, bus, logger)

	analysisEngine := analysis.NewEngine(repo, bus, logger)

	strategyEngine := strategy.NewEngine(repo, bus, logger)
	riskManager := risk.NewManager(repo, bus, logger)
	backtestEngine := backtest.NewEngine(repo)
	execEngine := execution.NewEngine(repo, bus, logger)
	analyzer := ai.NewAnalyzer(repo, bus, logger, cfg.AI.APIKey, cfg.AI.Model)

	router := api.NewRouter(cfg.Server.CORSOrigins, mgr, repo, strategyEngine, backtestEngine, execEngine, analyzer)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", slog.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	if cfg.Finnhub.APIKey != "" {
		defaultSymbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "NVDA"}
		go func() {
			if err := mgr.Start(context.Background(), defaultSymbols); err != nil {
				logger.Error("failed to start data manager", slog.String("error", err.Error()))
			}
		}()
	}

	go func() {
		if err := analysisEngine.Start(context.Background()); err != nil {
			logger.Error("failed to start analysis engine", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := strategyEngine.Start(context.Background()); err != nil {
			logger.Error("failed to start strategy engine", slog.String("error", err.Error()))
		}
	}()

	_ = riskManager

	go func() {
		if err := execEngine.Start(context.Background()); err != nil {
			logger.Error("failed to start execution engine", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	mgr.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}
	logger.Info("server stopped")
}

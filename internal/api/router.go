package api

import (
	"net/http"

	"github.com/flametest/market-lens/internal/ai"
	"github.com/flametest/market-lens/internal/backtest"
	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/execution"
	"github.com/flametest/market-lens/internal/strategy"
)

func NewRouter(corsOrigins []string, mgr *data.Manager, repo *data.Repository, strategyEngine *strategy.Engine, backtestEngine *backtest.Engine, execEngine *execution.Engine, analyzer *ai.Analyzer, hub *Hub) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/system/health", handleHealth)

	if mgr != nil {
		market := NewMarketHandler(mgr)
		market.RegisterRoutes(mux)
	}

	if repo != nil {
		signals := NewSignalHandler(repo)
		signals.RegisterRoutes(mux)
	}

	if strategyEngine != nil {
		strategies := NewStrategyHandler(strategyEngine, repo)
		strategies.RegisterRoutes(mux)
	}

	if backtestEngine != nil {
		bt := NewBacktestHandler(backtestEngine, repo)
		bt.RegisterRoutes(mux)
	}

	if execEngine != nil {
		trading := NewTradingHandler(execEngine)
		trading.RegisterRoutes(mux)
	}

	if analyzer != nil {
		aiHandler := NewAIHandler(analyzer, repo)
		aiHandler.RegisterRoutes(mux)
	}

	if hub != nil {
		mux.HandleFunc("GET /ws", HandleWebSocket(hub))
	}

	var handler http.Handler = mux
	handler = CORSMiddleware(corsOrigins)(handler)
	return handler
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

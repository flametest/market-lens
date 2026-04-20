package api

import (
	"net/http"

	"github.com/flametest/market-lens/internal/data"
)

func NewRouter(corsOrigins []string, mgr *data.Manager) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/system/health", handleHealth)

	if mgr != nil {
		market := NewMarketHandler(mgr)
		market.RegisterRoutes(mux)
	}

	var handler http.Handler = mux
	handler = CORSMiddleware(corsOrigins)(handler)
	return handler
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/strategy"
	"github.com/flametest/market-lens/pkg/model"
)

type StrategyHandler struct {
	engine *strategy.Engine
	repo   *data.Repository
}

func NewStrategyHandler(engine *strategy.Engine, repo *data.Repository) *StrategyHandler {
	return &StrategyHandler{engine: engine, repo: repo}
}

func (h *StrategyHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/strategies", h.listStrategies)
	mux.HandleFunc("POST /api/v1/strategies", h.createStrategy)
	mux.HandleFunc("GET /api/v1/strategies/{id}", h.getStrategy)
	mux.HandleFunc("PUT /api/v1/strategies/{id}", h.updateStrategy)
}

func (h *StrategyHandler) listStrategies(w http.ResponseWriter, r *http.Request) {
	configs, err := h.repo.GetStrategyConfigs(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if configs == nil {
		configs = []model.StrategyConfig{}
	}
	WriteJSON(w, http.StatusOK, configs)
}

func (h *StrategyHandler) createStrategy(w http.ResponseWriter, r *http.Request) {
	var cfg model.StrategyConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}
	if cfg.Name == "" {
		WriteError(w, http.StatusBadRequest, 400, "name is required")
		return
	}

	now := time.Now().Unix()
	cfg.ID = data.GenerateID()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now

	if err := h.repo.SaveStrategyConfig(r.Context(), cfg); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, cfg)
}

func (h *StrategyHandler) getStrategy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	configs := h.engine.GetStrategies()
	for _, cfg := range configs {
		if cfg.ID == id {
			WriteJSON(w, http.StatusOK, cfg)
			return
		}
	}
	WriteError(w, http.StatusNotFound, 404, "strategy not found")
}

func (h *StrategyHandler) updateStrategy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var cfg model.StrategyConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}

	cfg.ID = id
	cfg.UpdatedAt = time.Now().Unix()

	if err := h.repo.SaveStrategyConfig(r.Context(), cfg); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	h.engine.UpdateConfig(cfg.Name, cfg)

	WriteJSON(w, http.StatusOK, cfg)
}

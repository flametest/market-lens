package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/flametest/market-lens/internal/backtest"
	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/pkg/model"
)

type BacktestHandler struct {
	engine *backtest.Engine
	repo   *data.Repository
}

func NewBacktestHandler(engine *backtest.Engine, repo *data.Repository) *BacktestHandler {
	return &BacktestHandler{engine: engine, repo: repo}
}

func (h *BacktestHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/backtest", h.runBacktest)
	mux.HandleFunc("GET /api/v1/backtest", h.listRuns)
	mux.HandleFunc("GET /api/v1/backtest/{id}", h.getRun)
}

func (h *BacktestHandler) runBacktest(w http.ResponseWriter, r *http.Request) {
	var cfg model.BacktestConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}

	if cfg.Symbol == "" {
		WriteError(w, http.StatusBadRequest, 400, "symbol is required")
		return
	}
	if cfg.Strategy.Name == "" {
		WriteError(w, http.StatusBadRequest, 400, "strategy name is required")
		return
	}

	run := model.BacktestRun{
		ID:        data.GenerateID(),
		Config:    cfg,
		Status:    "running",
		AIMode:    cfg.AIMode,
		CreatedAt: time.Now().Unix(),
	}

	if err := h.repo.CreateBacktestRun(r.Context(), run); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	result, err := h.engine.Run(r.Context(), cfg)
	if err != nil {
		h.repo.UpdateBacktestResult(r.Context(), run.ID, nil, "failed", err.Error())
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	if err := h.repo.UpdateBacktestResult(r.Context(), run.ID, result, "completed", ""); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	run.Result = result
	run.Status = "completed"
	WriteJSON(w, http.StatusOK, run)
}

func (h *BacktestHandler) listRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.repo.ListBacktestRuns(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if runs == nil {
		runs = make([]model.BacktestRun, 0)
	}
	WriteJSON(w, http.StatusOK, runs)
}

func (h *BacktestHandler) getRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := h.repo.GetBacktestRun(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, 404, "backtest run not found")
		return
	}
	WriteJSON(w, http.StatusOK, run)
}

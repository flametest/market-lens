package api

import (
	"net/http"
	"strconv"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/pkg/model"
)

type SignalHandler struct {
	repo *data.Repository
}

func NewSignalHandler(repo *data.Repository) *SignalHandler {
	return &SignalHandler{repo: repo}
}

func (h *SignalHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/signals", h.listSignals)
	mux.HandleFunc("GET /api/v1/signals/{symbol}", h.getSignalsBySymbol)
}

func (h *SignalHandler) listSignals(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	signals, err := h.repo.GetSignals(r.Context(), "", limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if signals == nil {
		signals = make([]model.Signal, 0)
	}
	WriteJSON(w, http.StatusOK, signals)
}

func (h *SignalHandler) getSignalsBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	if symbol == "" {
		WriteError(w, http.StatusBadRequest, 400, "symbol is required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	signals, err := h.repo.GetSignals(r.Context(), symbol, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, signals)
}

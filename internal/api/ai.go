package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/flametest/market-lens/internal/ai"
	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/pkg/model"
)

type AIHandler struct {
	analyzer *ai.Analyzer
	repo     *data.Repository
}

func NewAIHandler(analyzer *ai.Analyzer, repo *data.Repository) *AIHandler {
	return &AIHandler{analyzer: analyzer, repo: repo}
}

func (h *AIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/ai/analyze", h.analyze)
	mux.HandleFunc("GET /api/v1/ai/sentiment", h.getSentimentHistory)
	mux.HandleFunc("GET /api/v1/ai/risk-events", h.getRiskEvents)
	mux.HandleFunc("GET /api/v1/ai/config", h.getConfig)
	mux.HandleFunc("PUT /api/v1/ai/config", h.updateConfig)
}

func (h *AIHandler) analyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}
	if req.Text == "" {
		WriteError(w, http.StatusBadRequest, 400, "text is required")
		return
	}

	result, err := h.analyzer.Analyze(r.Context(), req.Text)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func (h *AIHandler) getSentimentHistory(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	results, err := h.analyzer.GetHistory(r.Context(), symbol, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if results == nil {
		results = make([]model.SentimentResult, 0)
	}
	WriteJSON(w, http.StatusOK, results)
}

func (h *AIHandler) getRiskEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	events, err := h.analyzer.GetRiskEvents(r.Context(), limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if events == nil {
		events = make([]model.RiskEvent, 0)
	}
	WriteJSON(w, http.StatusOK, events)
}

func (h *AIHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":             true,
		"signalParticipation": true,
		"confidenceAdjustment": true,
		"riskEnhancement":     true,
		"signalWeight":        0.3,
	})
}

func (h *AIHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg map[string]any
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}
	WriteJSON(w, http.StatusOK, cfg)
}

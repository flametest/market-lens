package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/flametest/market-lens/internal/data"
)

var startTime = time.Now()

type SettingsHandler struct {
	mgr  *data.Manager
	repo *data.Repository
}

func NewSettingsHandler(mgr *data.Manager, repo *data.Repository) *SettingsHandler {
	return &SettingsHandler{mgr: mgr, repo: repo}
}

func (h *SettingsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings/provider", h.getProvider)
	mux.HandleFunc("PUT /api/v1/settings/provider", h.setProvider)
	mux.HandleFunc("GET /api/v1/settings/system", h.getSystemMetrics)
	mux.HandleFunc("GET /api/v1/settings/risk", h.getRiskConfig)
	mux.HandleFunc("PUT /api/v1/settings/risk", h.setRiskConfig)
}

func (h *SettingsHandler) getProvider(w http.ResponseWriter, r *http.Request) {
	active := h.mgr.ActiveProviderName()
	all := h.mgr.ProviderNames()
	WriteJSON(w, http.StatusOK, map[string]any{
		"active":    active,
		"providers": all,
	})
}

func (h *SettingsHandler) setProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, 400, "name is required")
		return
	}

	if err := h.mgr.SwitchProvider(req.Name); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"active": h.mgr.ActiveProviderName(),
	})
}

func (h *SettingsHandler) getSystemMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	dbStatus := "healthy"
	if h.repo != nil {
		if err := h.repo.Ping(r.Context()); err != nil {
			dbStatus = "unhealthy"
		}
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"goroutines": runtime.NumGoroutine(),
		"memoryMB":   m.Alloc / 1024 / 1024,
		"heapMB":     m.HeapAlloc / 1024 / 1024,
		"sysMB":      m.Sys / 1024 / 1024,
		"gcPauseNs":  m.PauseTotalNs,
		"uptime":     int64(time.Since(startTime).Seconds()),
		"dbStatus":   dbStatus,
		"provider":   h.mgr.ActiveProviderName(),
	})
}

func (h *SettingsHandler) getRiskConfig(w http.ResponseWriter, r *http.Request) {
	val, err := h.repo.GetSetting(r.Context(), "risk_config")
	if err != nil || val == "" {
		WriteJSON(w, http.StatusOK, map[string]any{
			"rules": []map[string]any{
				{"name": "Max Position Size", "value": "1000 shares", "enabled": true},
				{"name": "Max Drawdown", "value": "15%", "enabled": true},
				{"name": "Daily Trade Limit", "value": "50", "enabled": true},
				{"name": "Max Order Amount", "value": "$50,000", "enabled": true},
				{"name": "Min Trade Interval", "value": "30s", "enabled": true},
			},
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(val))
}

func (h *SettingsHandler) setRiskConfig(w http.ResponseWriter, r *http.Request) {
	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}
	if err := h.repo.SaveSetting(r.Context(), "risk_config", string(body)); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

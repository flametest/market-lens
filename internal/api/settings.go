package api

import (
	"encoding/json"
	"net/http"

	"github.com/flametest/market-lens/internal/data"
)

type SettingsHandler struct {
	mgr *data.Manager
}

func NewSettingsHandler(mgr *data.Manager) *SettingsHandler {
	return &SettingsHandler{mgr: mgr}
}

func (h *SettingsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings/provider", h.getProvider)
	mux.HandleFunc("PUT /api/v1/settings/provider", h.setProvider)
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

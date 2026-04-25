package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/execution"
	"github.com/flametest/market-lens/pkg/model"
)

type TradingHandler struct {
	engine *execution.Engine
	mgr    quoteFetcher
	repo   *data.Repository
}

type quoteFetcher interface {
	GetQuote(ctx context.Context, symbol string) (*model.Tick, error)
}

func NewTradingHandler(engine *execution.Engine, mgr quoteFetcher, repo *data.Repository) *TradingHandler {
	return &TradingHandler{engine: engine, mgr: mgr, repo: repo}
}

func (h *TradingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/trading/account", h.getAccount)
	mux.HandleFunc("GET /api/v1/trading/positions", h.listPositions)
	mux.HandleFunc("GET /api/v1/trading/orders", h.listOrders)
	mux.HandleFunc("POST /api/v1/trading/orders", h.placeOrder)
	mux.HandleFunc("DELETE /api/v1/trading/orders/{id}", h.cancelOrder)
}

func (h *TradingHandler) getAccount(w http.ResponseWriter, r *http.Request) {
	account, err := h.repo.GetAccount(r.Context(), h.engine.GetAccountID())
	if err != nil || account == nil {
		WriteJSON(w, http.StatusOK, map[string]any{
			"accountId":     h.engine.GetAccountID(),
			"cash":          "0",
			"totalValue":    "0",
			"unrealizedPnl": "0",
			"realizedPnl":   "0",
			"dailyPnl":      0,
		})
		return
	}
	WriteJSON(w, http.StatusOK, account)
}

func (h *TradingHandler) listPositions(w http.ResponseWriter, r *http.Request) {
	positions, err := h.repo.ListPositions(r.Context(), h.engine.GetAccountID())
	if err != nil {
		WriteJSON(w, http.StatusOK, []model.Position{})
		return
	}
	var active []model.Position
	for _, p := range positions {
		if !p.Quantity.IsZero() {
			active = append(active, p)
		}
	}
	if active == nil {
		active = []model.Position{}
	}
	WriteJSON(w, http.StatusOK, active)
}

func (h *TradingHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	orders, err := h.repo.GetOrders(r.Context(), h.engine.GetAccountID(), limit)
	if err != nil {
		WriteJSON(w, http.StatusOK, []model.Order{})
		return
	}
	if orders == nil {
		orders = []model.Order{}
	}
	WriteJSON(w, http.StatusOK, orders)
}

func (h *TradingHandler) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbol   string          `json:"symbol"`
		Side     model.OrderSide `json:"side"`
		Type     model.OrderType `json:"type"`
		Quantity float64         `json:"quantity"`
		Price    float64         `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, 400, "invalid request body")
		return
	}

	if req.Symbol == "" || req.Quantity <= 0 {
		WriteError(w, http.StatusBadRequest, 400, "symbol and quantity are required")
		return
	}

	price := decimal.NewFromFloat(req.Price)
	if req.Type == model.OrderMarket || req.Type == "" {
		req.Type = model.OrderMarket
		if price.IsZero() || !price.IsPositive() {
			quote, err := h.mgr.GetQuote(r.Context(), req.Symbol)
			if err != nil || quote == nil {
				WriteError(w, http.StatusBadRequest, 400, "failed to get market price for "+req.Symbol)
				return
			}
			price = quote.Price
		}
	} else if price.IsZero() || !price.IsPositive() {
		WriteError(w, http.StatusBadRequest, 400, "price is required for limit orders")
		return
	}

	decision := model.OrderDecision{
		Symbol:   req.Symbol,
		Side:     req.Side,
		Type:     req.Type,
		Quantity: decimal.NewFromFloat(req.Quantity),
		Price:    price,
		Strategy: "manual",
		Reason:   "manual order",
	}

	if decision.Type == "" {
		decision.Type = model.OrderMarket
	}

	order, err := h.engine.SubmitOrder(r.Context(), decision)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if order == nil {
		WriteError(w, http.StatusBadRequest, 400, "order rejected: insufficient funds or no position")
		return
	}

	WriteJSON(w, http.StatusCreated, order)
}

func (h *TradingHandler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, 400, "order id is required")
		return
	}
	if err := h.engine.CancelOrder(r.Context(), id); err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

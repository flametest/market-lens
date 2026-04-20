package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/pkg/model"
)

type MarketHandler struct {
	mgr *data.Manager
}

func NewMarketHandler(mgr *data.Manager) *MarketHandler {
	return &MarketHandler{mgr: mgr}
}

func (h *MarketHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/market/symbols", h.listSymbols)
	mux.HandleFunc("GET /api/v1/market/symbols/{code}/candles", h.getCandles)
	mux.HandleFunc("GET /api/v1/market/symbols/{code}/quote", h.getQuote)
	mux.HandleFunc("GET /api/v1/market/symbols/{code}/ticks", h.getLatestTick)
}

func (h *MarketHandler) listSymbols(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query != "" {
		symbols, err := h.mgr.SearchSymbols(r.Context(), query)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, 500, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, symbols)
		return
	}

	defaultSymbols := []model.SymbolInfo{
		{Code: "AAPL", Name: "Apple Inc.", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "GOOGL", Name: "Alphabet Inc.", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "MSFT", Name: "Microsoft Corp.", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "AMZN", Name: "Amazon.com Inc.", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "NVDA", Name: "NVIDIA Corp.", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "META", Name: "Meta Platforms", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "TSLA", Name: "Tesla Inc.", Market: model.MarketUSStock, Exchange: "NASDAQ", Type: "stock", Currency: "USD", Enabled: true},
		{Code: "JPM", Name: "JPMorgan Chase", Market: model.MarketUSStock, Exchange: "NYSE", Type: "stock", Currency: "USD", Enabled: true},
	}
	WriteJSON(w, http.StatusOK, defaultSymbols)
}

func (h *MarketHandler) getCandles(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("code")
	if symbol == "" {
		WriteError(w, http.StatusBadRequest, 400, "symbol is required")
		return
	}

	interval := model.Interval(r.URL.Query().Get("interval"))
	if interval == "" {
		interval = model.Interval1Day
	}

	start, _ := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	end, _ := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
	if end == 0 {
		end = time.Now().UnixMilli()
	}
	if start == 0 {
		start = end - 90*24*3600*1000
	}

	candles, err := h.mgr.FetchAndStoreCandles(r.Context(), symbol, interval, start, end)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}

	if candles == nil {
		candles = []model.Candle{}
	}
	WriteJSON(w, http.StatusOK, candles)
}

func (h *MarketHandler) getQuote(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("code")
	if symbol == "" {
		WriteError(w, http.StatusBadRequest, 400, "symbol is required")
		return
	}

	tick, err := h.mgr.GetQuote(r.Context(), symbol)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, tick)
}

func (h *MarketHandler) getLatestTick(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("code")
	if symbol == "" {
		WriteError(w, http.StatusBadRequest, 400, "symbol is required")
		return
	}

	tick, err := h.mgr.GetLatestTick(r.Context(), symbol)
	if err != nil {
		WriteError(w, http.StatusNotFound, 404, "no tick data found")
		return
	}
	WriteJSON(w, http.StatusOK, tick)
}

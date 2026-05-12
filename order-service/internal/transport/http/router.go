package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"order-service/internal/domain"
	"order-service/internal/middleware"
	"order-service/internal/usecase"
)

type Router struct {
	mux     *http.ServeMux
	orderUC usecase.OrderUsecase
}

func NewRouter(orderUC usecase.OrderUsecase, limiter *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	h := &handler{uc: orderUC}

	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders", h.listOrders)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("PATCH /orders/{id}/status", h.updateStatus)

	// Apply rate limiter globally
	return limiter.Middleware(mux)
}

type handler struct {
	uc usecase.OrderUsecase
}

// POST /orders
func (h *handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string  `json:"customer_id"`
		ItemName   string  `json:"item_name"`
		Amount     float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	order, err := h.uc.CreateOrder(r.Context(), req.CustomerID, req.ItemName, req.Amount)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, order, http.StatusCreated)
}

// GET /orders/{id}
func (h *handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	order, err := h.uc.GetOrder(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "record") {
			jsonError(w, "order not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, order, http.StatusOK)
}

// GET /orders
func (h *handler) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.uc.ListRecent(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, orders, http.StatusOK)
}

// PATCH /orders/{id}/status
func (h *handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Status    string `json:"status"`
		PaymentID string `json:"payment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	order, err := h.uc.UpdateOrderStatus(r.Context(), id, domain.OrderStatus(req.Status), req.PaymentID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, order, http.StatusOK)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func parseID(r *http.Request, name string) (uint, error) {
	raw := r.PathValue(name)
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

func jsonResponse(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

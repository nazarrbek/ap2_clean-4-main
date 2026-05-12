package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"payment-service/internal/usecase"
)

func NewRouter(paymentUC usecase.PaymentUsecase) http.Handler {
	mux := http.NewServeMux()
	h := &handler{uc: paymentUC}

	mux.HandleFunc("POST /payments", h.createPayment)
	mux.HandleFunc("GET /payments/{id}", h.getPayment)
	mux.HandleFunc("GET /payments/order/{orderID}", h.getByOrder)

	return mux
}

type handler struct {
	uc usecase.PaymentUsecase
}

func (h *handler) createPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID uint    `json:"order_id"`
		Email   string  `json:"customer_email"`
		Amount  float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	payment, err := h.uc.ProcessPayment(r.Context(), req.OrderID, req.Email, req.Amount)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, payment, http.StatusCreated)
}

func (h *handler) getPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseUint(r.PathValue("id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := h.uc.GetPayment(r.Context(), id)
	if err != nil {
		jsonError(w, "payment not found", http.StatusNotFound)
		return
	}
	jsonResponse(w, p, http.StatusOK)
}

func (h *handler) getByOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseUint(r.PathValue("orderID"))
	if err != nil {
		jsonError(w, "invalid order id", http.StatusBadRequest)
		return
	}
	p, err := h.uc.GetByOrderID(r.Context(), orderID)
	if err != nil {
		jsonError(w, "payment not found", http.StatusNotFound)
		return
	}
	jsonResponse(w, p, http.StatusOK)
}

func parseUint(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	return uint(n), err
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

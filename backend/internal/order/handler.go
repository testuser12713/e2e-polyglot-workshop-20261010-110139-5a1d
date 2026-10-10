package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Handler serves the workshop order routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds an order handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// createOrderRequest is the body of POST /api/workshop/orders.
type createOrderRequest struct {
	VehicleID          int    `json:"vehicleId"`
	RequestedDate      string `json:"requestedDate"`
	ProblemDescription string `json:"problemDescription"`
}

// addItemsRequest is the body of POST /api/workshop/orders/{id}/items.
type addItemsRequest struct {
	Items []model.OrderItemInput `json:"items"`
}

// Create handles POST /api/workshop/orders.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	ord, err := CreateOrder(r.Context(), h.Pool, req.VehicleID, req.RequestedDate, req.ProblemDescription)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, ord)
}

// Get handles GET /api/workshop/orders/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "resource not found")
		return
	}

	ord, err := LoadOrder(r.Context(), h.Pool, id)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ord)
}

// AddItems handles POST /api/workshop/orders/{id}/items.
func (h *Handler) AddItems(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "resource not found")
		return
	}

	var req addItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	ord, err := AddItems(r.Context(), h.Pool, id, req.Items)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ord)
}

// ChangeStatus handles POST /api/workshop/orders/{id}/status.
func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "change order status")
}

// pathID reads the {id} path value of a request and parses it as a positive
// integer. A missing or non-numeric id is not a resource that can exist.
func pathID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// writeOrderError maps a service error to the contract's error body. It never
// logs user values, so no name, plate or description can leak (AC-29).
func writeOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "resource not found")
	case errors.Is(err, ErrValidation):
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "invalid request")
	default:
		log.Printf("order: internal error")
		httpx.Error(w, httpx.CodeInternal, http.StatusInternalServerError, "internal server error")
	}
}

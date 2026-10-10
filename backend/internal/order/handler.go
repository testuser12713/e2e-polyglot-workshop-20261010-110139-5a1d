package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"workshop/backend/internal/auth"
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

// changeStatusRequest is the body of POST /api/workshop/orders/{id}/status.
type changeStatusRequest struct {
	Status string `json:"status"`
}

// ChangeStatus handles POST /api/workshop/orders/{id}/status. The auth guard
// has already accepted the bearer token, so the acting employee is read from
// the same claims to attribute the history entry.
func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "resource not found")
		return
	}

	var req changeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	ord, err := ChangeStatus(r.Context(), h.Pool, id, req.Status, employeeIDFromRequest(r))
	if err != nil {
		writeOrderError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ord)
}

// employeeIDFromRequest returns the id of the signed-in employee so a status
// change can be attributed to them. The auth guard has already verified the
// request's token; this reads the claims it carries. An absent or unparsable
// id yields 0 (an unattributed change) rather than failing the request.
func employeeIDFromRequest(r *http.Request) int {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return 0
	}
	claims, err := auth.VerifyToken(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return 0
	}
	id, err := strconv.Atoi(claims.Sub)
	if err != nil {
		return 0
	}
	return id
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
	case errors.Is(err, ErrInvalidTransition):
		httpx.Error(w, httpx.CodeInvalidTransition, http.StatusConflict, "order status transition is not allowed")
	case errors.Is(err, ErrValidation):
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "invalid request")
	default:
		log.Printf("order: internal error")
		httpx.Error(w, httpx.CodeInternal, http.StatusInternalServerError, "internal server error")
	}
}

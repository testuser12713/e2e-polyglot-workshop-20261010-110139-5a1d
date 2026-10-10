package customer

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/store"
)

// Handler serves the workshop customer routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a customer handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// customerRequest is the body of POST and PUT /api/workshop/customers.
type customerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Create handles POST /api/workshop/customers.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := decodeBody(r, &req); err != nil {
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "malformed JSON body")
		return
	}

	c, err := Create(r.Context(), h.Pool, req.Name, req.Email, req.Phone)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

// Get handles GET /api/workshop/customers/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "customer not found")
		return
	}

	c, err := Get(r.Context(), h.Pool, id)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

// Update handles PUT /api/workshop/customers/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "customer not found")
		return
	}

	var req customerRequest
	if err := decodeBody(r, &req); err != nil {
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, "malformed JSON body")
		return
	}

	c, err := Update(r.Context(), h.Pool, id, req.Name, req.Email, req.Phone)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

// pathID reads the {id} path segment. A missing or non-positive id is not a
// valid resource id.
func pathID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// decodeBody decodes a JSON request body. An empty body is not an error: the
// service turns the missing fields into a validation error.
func decodeBody(r *http.Request, dst any) error {
	err := json.NewDecoder(r.Body).Decode(dst)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// writeError maps a service error to the unified error body.
func writeError(w http.ResponseWriter, err error) {
	var ve ValidationError
	if errors.As(err, &ve) {
		httpx.Error(w, httpx.CodeValidation, http.StatusUnprocessableEntity, ve.Message)
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, httpx.CodeNotFound, http.StatusNotFound, "customer not found")
		return
	}
	httpx.Error(w, httpx.CodeInternal, http.StatusInternalServerError, "internal server error")
}

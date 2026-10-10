package vehicle

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Handler serves the workshop vehicle routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a vehicle handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// vehicleRequest is the body of POST and PUT /api/workshop/vehicles.
type vehicleRequest struct {
	CustomerID int    `json:"customerId"`
	Plate      string `json:"plate"`
	Brand      string `json:"brand"`
	Model      string `json:"model"`
	MileageKm  int    `json:"mileageKm"`
}

// pathID reads the {id} wildcard of a vehicle route as a positive integer.
func pathID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// decodeVehicle reads the JSON body; a missing or malformed body is a
// validation failure.
func decodeVehicle(r *http.Request) (vehicleRequest, error) {
	var req vehicleRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return vehicleRequest{}, errors.New("request body is required")
		}
		return vehicleRequest{}, errors.New("malformed JSON body")
	}
	return req, nil
}

// writeServiceError maps a service error to the shared error body and status.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrDuplicatePlate):
		httpx.Error(w, model.CodeDuplicatePlate, http.StatusConflict, "a vehicle with this plate already exists")
	case errors.Is(err, ErrCustomerNotFound):
		httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "customer not found")
	case errors.Is(err, ErrVehicleNotFound):
		httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "vehicle not found")
	default:
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
	}
}

// Create handles POST /api/workshop/vehicles.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := decodeVehicle(r)
	if err != nil {
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, err.Error())
		return
	}

	v, err := Create(r.Context(), h.Pool, req.CustomerID, req.Plate, req.Brand, req.Model, req.MileageKm)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, v)
}

// Get handles GET /api/workshop/vehicles/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "vehicle not found")
		return
	}

	v, err := Get(r.Context(), h.Pool, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

// Update handles PUT /api/workshop/vehicles/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "vehicle not found")
		return
	}

	req, err := decodeVehicle(r)
	if err != nil {
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, err.Error())
		return
	}

	v, err := Update(r.Context(), h.Pool, id, req.CustomerID, req.Plate, req.Brand, req.Model, req.MileageKm)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

// Package public serves the unauthenticated customer area: requesting an
// appointment and looking up an order status. Both routes answer in the shared
// error body and never leak customer contact data.
package public

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"workshop/backend/internal/customer"
	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/order"
	"workshop/backend/internal/store"
	"workshop/backend/internal/vehicle"
)

// Handler serves the public customer routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a public handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// appointmentRequest is the body of POST /api/public/appointments.
type appointmentRequest struct {
	Name               string `json:"name"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	Plate              string `json:"plate"`
	Brand              string `json:"brand"`
	Model              string `json:"model"`
	MileageKm          int    `json:"mileageKm"`
	RequestedDate      string `json:"requestedDate"`
	ProblemDescription string `json:"problemDescription"`
}

// appointmentResponse is the answer to a successful appointment request.
type appointmentResponse struct {
	OrderNumber string `json:"orderNumber"`
	Status      string `json:"status"`
}

// decodeAppointment reads the JSON body. An absent body is not an error here:
// the service turns the missing fields into a validation error.
func decodeAppointment(r *http.Request) (appointmentRequest, error) {
	var req appointmentRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if errors.Is(err, io.EOF) {
		return req, nil
	}
	return req, err
}

// CreateAppointment handles POST /api/public/appointments. It creates the
// customer, the vehicle and the order in ONE transaction by calling the shared
// customer, vehicle and order services, so a rejected plate or an unusable
// field leaves no partial rows behind. It answers 201 with the new order number
// and status.
func (h *Handler) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	req, err := decodeAppointment(r)
	if err != nil {
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, "malformed JSON body")
		return
	}

	ctx := r.Context()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		writeAppointmentError(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cust, err := customer.Create(ctx, tx, req.Name, req.Email, req.Phone)
	if err != nil {
		writeAppointmentError(w, err)
		return
	}

	veh, err := vehicle.Create(ctx, tx, cust.ID, req.Plate, req.Brand, req.Model, req.MileageKm)
	if err != nil {
		writeAppointmentError(w, err)
		return
	}

	ord, err := order.CreateOrder(ctx, tx, veh.ID, req.RequestedDate, req.ProblemDescription)
	if err != nil {
		writeAppointmentError(w, err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeAppointmentError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, appointmentResponse{OrderNumber: ord.OrderNumber, Status: ord.Status})
}

// writeAppointmentError maps a service error to the contract's error body. It
// never logs user values, so no name, plate or description can leak (AC-29).
func writeAppointmentError(w http.ResponseWriter, err error) {
	var validation customer.ValidationError
	switch {
	case errors.As(err, &validation):
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, validation.Message)
	case errors.Is(err, vehicle.ErrValidation), errors.Is(err, order.ErrValidation):
		httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, "invalid request")
	case errors.Is(err, vehicle.ErrDuplicatePlate):
		httpx.Error(w, model.CodeDuplicatePlate, http.StatusConflict, "a vehicle with this plate already exists")
	default:
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "internal server error")
	}
}

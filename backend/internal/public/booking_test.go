package public

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// newBookingServer mounts the appointment handler on the route this ticket
// owns, without the router, so the handler and its transaction are exercised
// against the real PostgreSQL database.
func newBookingServer(t *testing.T, pool store.Pool) *httptest.Server {
	t.Helper()
	h := NewHandler(pool)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/public/appointments", h.CreateAppointment)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postAppointment(t *testing.T, url string, req appointmentRequest) *http.Response {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST appointment: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decodeErrorBody(t *testing.T, resp *http.Response) httpx.ErrorBody {
	t.Helper()
	var eb httpx.ErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&eb); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return eb
}

// cleanupAppointment removes exactly the rows a successful request created by
// their ids: the order first (it references the vehicle), then the vehicle and
// the customer.
func cleanupAppointment(t *testing.T, pool store.Pool, orderNumber, plate, email string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE order_number = $1`, orderNumber)
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE plate_normalized = $1`, plate)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE email = $1`, email)
	})
}

func TestCreateAppointmentCreatesCustomerVehicleAndOrder(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	suffix := uniqueSuffix()
	plate := "B-" + suffix
	normalized := strings.ToUpper(strings.ReplaceAll(plate, "-", ""))
	email := "booking-" + suffix + "@example.test"

	srv := newBookingServer(t, pool)
	req := appointmentRequest{
		Name:               "Anna Beispiel",
		Email:              email,
		Phone:              "0170-1234567",
		Plate:              plate,
		Brand:              "VW",
		Model:              "Golf",
		MileageKm:          88000,
		RequestedDate:      "2026-11-20",
		ProblemDescription: "Bremsen quietschen",
	}

	resp := postAppointment(t, srv.URL+"/api/public/appointments", req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", resp.StatusCode)
	}

	var out appointmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.OrderNumber == "" {
		t.Fatal("orderNumber is empty")
	}
	if out.Status != model.StatusRequested {
		t.Fatalf("status = %q, want %q", out.Status, model.StatusRequested)
	}
	cleanupAppointment(t, pool, out.OrderNumber, normalized, email)

	// The customer exists with the submitted contact data.
	var (
		customerID int
		name       string
	)
	if err := pool.QueryRow(ctx, `SELECT id, name FROM customers WHERE email = $1`, email).
		Scan(&customerID, &name); err != nil {
		t.Fatalf("customer was not created: %v", err)
	}
	if name != req.Name {
		t.Fatalf("customer name = %q, want %q", name, req.Name)
	}

	// The vehicle exists, is linked to the customer and carries the normalised plate.
	var (
		vehicleID       int
		customerFK      int
		storedMileageKm int
	)
	if err := pool.QueryRow(ctx,
		`SELECT id, customer_id, mileage_km FROM vehicles WHERE plate_normalized = $1`, normalized).
		Scan(&vehicleID, &customerFK, &storedMileageKm); err != nil {
		t.Fatalf("vehicle was not created: %v", err)
	}
	if vehicleID == 0 || customerFK != customerID {
		t.Fatalf("vehicle customer_id = %d, want %d", customerFK, customerID)
	}
	if storedMileageKm != req.MileageKm {
		t.Fatalf("mileage_km = %d, want %d", storedMileageKm, req.MileageKm)
	}

	// The order exists, references the vehicle and carries the order number.
	var (
		orderID   int
		vehicleFK int
		status    string
	)
	if err := pool.QueryRow(ctx,
		`SELECT id, vehicle_id, status FROM orders WHERE order_number = $1`, out.OrderNumber).
		Scan(&orderID, &vehicleFK, &status); err != nil {
		t.Fatalf("order was not created: %v", err)
	}
	if orderID == 0 || vehicleFK != vehicleID {
		t.Fatalf("order vehicle_id = %d, want %d", vehicleFK, vehicleID)
	}
	if status != model.StatusRequested {
		t.Fatalf("stored status = %q, want %q", status, model.StatusRequested)
	}
}

func TestCreateAppointmentInvalidInputLeavesNoRows(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	suffix := uniqueSuffix()
	email := "booking-invalid-" + suffix + "@example.test"
	plate := "I-" + suffix
	normalized := strings.ToUpper(strings.ReplaceAll(plate, "-", ""))

	srv := newBookingServer(t, pool)
	// The problem description is empty, so order.CreateOrder rejects the request
	// after the customer and vehicle rows were already inserted in the
	// transaction. The rollback must remove them.
	req := appointmentRequest{
		Name:               "Fehler Kunde",
		Email:              email,
		Phone:              "0170-0000000",
		Plate:              plate,
		Brand:              "Audi",
		Model:              "A4",
		MileageKm:          10000,
		RequestedDate:      "2026-11-20",
		ProblemDescription: "   ",
	}

	resp := postAppointment(t, srv.URL+"/api/public/appointments", req)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", resp.StatusCode)
	}
	if eb := decodeErrorBody(t, resp); eb.Error.Code != model.CodeValidation {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeValidation)
	}

	var customerExists, vehicleExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM customers WHERE email = $1)`, email).Scan(&customerExists); err != nil {
		t.Fatalf("check customer: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vehicles WHERE plate_normalized = $1)`, normalized).Scan(&vehicleExists); err != nil {
		t.Fatalf("check vehicle: %v", err)
	}
	if customerExists {
		t.Error("a customer row survived the failed appointment")
	}
	if vehicleExists {
		t.Error("a vehicle row survived the failed appointment")
	}
}

func TestCreateAppointmentDuplicatePlate(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	suffix := uniqueSuffix()
	plate := "D-" + suffix
	normalized := strings.ToUpper(strings.ReplaceAll(plate, "-", ""))
	firstEmail := "booking-dup-a-" + suffix + "@example.test"
	secondEmail := "booking-dup-b-" + suffix + "@example.test"

	srv := newBookingServer(t, pool)
	base := appointmentRequest{
		Name:               "Erster Kunde",
		Phone:              "0170-1111111",
		Plate:              plate,
		Brand:              "VW",
		Model:              "Golf",
		MileageKm:          50000,
		RequestedDate:      "2026-11-20",
		ProblemDescription: "Inspektion",
	}

	first := base
	first.Email = firstEmail
	resp := postAppointment(t, srv.URL+"/api/public/appointments", first)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first appointment: status %d, want 201", resp.StatusCode)
	}
	var out appointmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	cleanupAppointment(t, pool, out.OrderNumber, normalized, firstEmail)

	// A different spelling of the same plate must be rejected with 409.
	second := base
	second.Email = secondEmail
	second.Plate = strings.ToLower(strings.ReplaceAll(plate, "-", " "))
	resp2 := postAppointment(t, srv.URL+"/api/public/appointments", second)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate appointment: status %d, want 409", resp2.StatusCode)
	}
	if eb := decodeErrorBody(t, resp2); eb.Error.Code != model.CodeDuplicatePlate {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeDuplicatePlate)
	}

	// The rejected request created neither a second customer nor a second vehicle.
	var customerExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM customers WHERE email = $1)`, secondEmail).Scan(&customerExists); err != nil {
		t.Fatalf("check customer: %v", err)
	}
	if customerExists {
		t.Error("the rejected duplicate request created a customer")
	}
	var vehicleCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM vehicles WHERE plate_normalized = $1`, normalized).Scan(&vehicleCount); err != nil {
		t.Fatalf("count vehicles: %v", err)
	}
	if vehicleCount != 1 {
		t.Fatalf("vehicle count for the plate = %d, want 1", vehicleCount)
	}
}

func TestCreateAppointmentMalformedBodyIsValidationError(t *testing.T) {
	pool := testPool(t)
	srv := newBookingServer(t, pool)

	resp, err := http.Post(srv.URL+"/api/public/appointments", "application/json", strings.NewReader("{not json"))
	if err != nil {
		t.Fatalf("POST malformed body: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", resp.StatusCode)
	}
	if eb := decodeErrorBody(t, resp); eb.Error.Code != model.CodeValidation {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeValidation)
	}
}

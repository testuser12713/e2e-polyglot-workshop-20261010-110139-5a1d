package vehicle

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
)

// doRequest runs one handler call in-process and returns the recorder.
func doRequest(t *testing.T, h *Handler, method, target, id, body string, fn http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

func decodeVehicleResponse(t *testing.T, rec *httptest.ResponseRecorder) model.Vehicle {
	t.Helper()
	var v model.Vehicle
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode vehicle: %v (%s)", err, rec.Body.String())
	}
	return v
}

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()
	var eb httpx.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &eb); err != nil {
		t.Fatalf("decode error body: %v (%s)", err, rec.Body.String())
	}
	return eb
}

func TestHandlerCreateReturns201(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	body := `{"customerId":` + strconv.Itoa(customerID) + `,"plate":"b-xy 123","brand":"VW","model":"Golf","mileageKm":120000}`
	rec := doRequest(t, h, http.MethodPost, "/api/workshop/vehicles", "", body, h.Create)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	v := decodeVehicleResponse(t, rec)
	if v.ID == 0 || v.CustomerID != customerID || v.Plate != "BXY123" {
		t.Fatalf("unexpected vehicle: %+v", v)
	}
	if v.Brand != "VW" || v.Model != "Golf" || v.MileageKm != 120000 {
		t.Fatalf("unexpected vehicle fields: %+v", v)
	}
}

func TestHandlerCreateDuplicatePlateReturns409(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	first := `{"customerId":` + strconv.Itoa(customerID) + `,"plate":"B-XY 123","brand":"VW","model":"Golf","mileageKm":1000}`
	if rec := doRequest(t, h, http.MethodPost, "/api/workshop/vehicles", "", first, h.Create); rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}

	second := `{"customerId":` + strconv.Itoa(customerID) + `,"plate":"bxy123","brand":"Audi","model":"A4","mileageKm":2000}`
	rec := doRequest(t, h, http.MethodPost, "/api/workshop/vehicles", "", second, h.Create)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorBody(t, rec).Error.Code; code != model.CodeDuplicatePlate {
		t.Fatalf("code = %q, want %q", code, model.CodeDuplicatePlate)
	}
}

func TestHandlerCreateUnknownCustomerReturns404(t *testing.T) {
	pool := testPool(t)
	h := NewHandler(pool)

	body := `{"customerId":999999999,"plate":"AB-123","brand":"VW","model":"Golf","mileageKm":1000}`
	rec := doRequest(t, h, http.MethodPost, "/api/workshop/vehicles", "", body, h.Create)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorBody(t, rec).Error.Code; code != model.CodeNotFound {
		t.Fatalf("code = %q, want %q", code, model.CodeNotFound)
	}
}

func TestHandlerCreateValidationReturns422(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	body := `{"customerId":` + strconv.Itoa(customerID) + `,"plate":"","brand":"","model":"","mileageKm":0}`
	rec := doRequest(t, h, http.MethodPost, "/api/workshop/vehicles", "", body, h.Create)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (%s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorBody(t, rec).Error.Code; code != model.CodeValidation {
		t.Fatalf("code = %q, want %q", code, model.CodeValidation)
	}
}

func TestHandlerGetReturns200(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	created, err := Create(context.Background(), pool, customerID, "AB-123", "VW", "Golf", 1000)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rec := doRequest(t, h, http.MethodGet, "/api/workshop/vehicles/"+strconv.Itoa(created.ID), strconv.Itoa(created.ID), "", h.Get)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	v := decodeVehicleResponse(t, rec)
	if v != created {
		t.Fatalf("got %+v, want %+v", v, created)
	}
}

func TestHandlerGetUnknownReturns404(t *testing.T) {
	pool := testPool(t)
	h := NewHandler(pool)

	rec := doRequest(t, h, http.MethodGet, "/api/workshop/vehicles/999999999", "999999999", "", h.Get)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorBody(t, rec).Error.Code; code != model.CodeNotFound {
		t.Fatalf("code = %q, want %q", code, model.CodeNotFound)
	}
}

func TestHandlerGetInvalidIDReturns404(t *testing.T) {
	pool := testPool(t)
	h := NewHandler(pool)

	rec := doRequest(t, h, http.MethodGet, "/api/workshop/vehicles/abc", "abc", "", h.Get)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerUpdateReturns200(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	created, err := Create(context.Background(), pool, customerID, "AB-123", "VW", "Golf", 1000)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := `{"customerId":` + strconv.Itoa(customerID) + `,"plate":"cd 456","brand":"Audi","model":"A4","mileageKm":5000}`
	rec := doRequest(t, h, http.MethodPut, "/api/workshop/vehicles/"+strconv.Itoa(created.ID), strconv.Itoa(created.ID), body, h.Update)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	v := decodeVehicleResponse(t, rec)
	if v.ID != created.ID || v.Plate != "CD456" || v.Brand != "Audi" || v.Model != "A4" || v.MileageKm != 5000 {
		t.Fatalf("unexpected vehicle: %+v", v)
	}
}

func TestHandlerUpdateDuplicateReturns409(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	if _, err := Create(context.Background(), pool, customerID, "AB-123", "VW", "Golf", 1000); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second, err := Create(context.Background(), pool, customerID, "CD-456", "Audi", "A4", 2000)
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	body := `{"customerId":` + strconv.Itoa(customerID) + `,"plate":"ab123","brand":"Audi","model":"A4","mileageKm":2000}`
	rec := doRequest(t, h, http.MethodPut, "/api/workshop/vehicles/"+strconv.Itoa(second.ID), strconv.Itoa(second.ID), body, h.Update)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorBody(t, rec).Error.Code; code != model.CodeDuplicatePlate {
		t.Fatalf("code = %q, want %q", code, model.CodeDuplicatePlate)
	}
}

func TestHandlerUpdateUnknownCustomerReturns404(t *testing.T) {
	pool := testPool(t)
	customerID := createCustomer(t, pool)
	h := NewHandler(pool)

	created, err := Create(context.Background(), pool, customerID, "AB-123", "VW", "Golf", 1000)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := `{"customerId":999999999,"plate":"AB-123","brand":"VW","model":"Golf","mileageKm":1000}`
	rec := doRequest(t, h, http.MethodPut, "/api/workshop/vehicles/"+strconv.Itoa(created.ID), strconv.Itoa(created.ID), body, h.Update)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (%s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorBody(t, rec).Error.Code; code != model.CodeNotFound {
		t.Fatalf("code = %q, want %q", code, model.CodeNotFound)
	}
}

package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// newOrderServer mounts the order handler on the three routes this ticket owns,
// without the auth guard, so the handler and its service are exercised against
// the real PostgreSQL database.
func newOrderServer(t *testing.T, pool store.Pool) *httptest.Server {
	t.Helper()
	h := NewHandler(pool)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/workshop/orders", h.Create)
	mux.HandleFunc("GET /api/workshop/orders/{id}", h.Get)
	mux.HandleFunc("POST /api/workshop/orders/{id}/items", h.AddItems)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decodeOrder(t *testing.T, resp *http.Response) model.Order {
	t.Helper()
	var ord model.Order
	if err := json.NewDecoder(resp.Body).Decode(&ord); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	return ord
}

func decodeError(t *testing.T, resp *http.Response) httpx.ErrorBody {
	t.Helper()
	var eb httpx.ErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&eb); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return eb
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestHandlerCreateThenGet(t *testing.T) {
	pool := testPool(t)
	vehicleID := seedVehicle(t, pool)
	srv := newOrderServer(t, pool)

	resp := postJSON(t, srv.URL+"/api/workshop/orders",
		`{"vehicleId":`+itoa(vehicleID)+`,"requestedDate":"2026-11-05","problemDescription":"Bremsen"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST create: status %d, want 201", resp.StatusCode)
	}
	created := decodeOrder(t, resp)
	if created.ID == 0 || created.OrderNumber == "" || created.Status != model.StatusRequested {
		t.Fatalf("created order = %+v", created)
	}

	getResp, err := http.Get(srv.URL + "/api/workshop/orders/" + itoa(created.ID))
	if err != nil {
		t.Fatalf("GET order: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET order: status %d, want 200", getResp.StatusCode)
	}
	got := decodeOrder(t, getResp)
	if got.OrderNumber != created.OrderNumber || got.VehicleID != vehicleID {
		t.Fatalf("got = %+v, want order %q for vehicle %d", got, created.OrderNumber, vehicleID)
	}
}

func TestHandlerCreateUnknownVehicle(t *testing.T) {
	pool := testPool(t)
	srv := newOrderServer(t, pool)

	resp := postJSON(t, srv.URL+"/api/workshop/orders",
		`{"vehicleId":999999999,"requestedDate":"2026-11-05","problemDescription":"Bremsen"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
	if eb := decodeError(t, resp); eb.Error.Code != model.CodeNotFound {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeNotFound)
	}
}

func TestHandlerCreateInvalidBody(t *testing.T) {
	pool := testPool(t)
	srv := newOrderServer(t, pool)

	cases := []string{
		`{"vehicleId":0,"requestedDate":"2026-11-05","problemDescription":"x"}`,
		`{"vehicleId":1,"requestedDate":"nope","problemDescription":"x"}`,
		`{"vehicleId":1,"requestedDate":"2026-11-05","problemDescription":""}`,
		`{not json`,
	}
	for _, body := range cases {
		resp := postJSON(t, srv.URL+"/api/workshop/orders", body)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("body %q: status %d, want 422", body, resp.StatusCode)
		}
		if eb := decodeError(t, resp); eb.Error.Code != model.CodeValidation {
			t.Fatalf("body %q: code = %q, want %q", body, eb.Error.Code, model.CodeValidation)
		}
	}
}

func TestHandlerGetUnknown(t *testing.T) {
	pool := testPool(t)
	srv := newOrderServer(t, pool)

	resp, err := http.Get(srv.URL + "/api/workshop/orders/999999999")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

func TestHandlerAddItems(t *testing.T) {
	pool := testPool(t)
	vehicleID := seedVehicle(t, pool)
	srv := newOrderServer(t, pool)

	createResp := postJSON(t, srv.URL+"/api/workshop/orders",
		`{"vehicleId":`+itoa(vehicleID)+`,"requestedDate":"2026-11-05","problemDescription":"Inspektion"}`)
	created := decodeOrder(t, createResp)

	body := `{"items":[
		{"kind":"labor","description":"Arbeitszeit","hours":1.5,"unitPriceCents":8000},
		{"kind":"part","description":"Ölfilter","quantity":2,"unitPriceCents":1250}
	]}`
	resp := postJSON(t, srv.URL+"/api/workshop/orders/"+itoa(created.ID)+"/items", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST items: status %d, want 200", resp.StatusCode)
	}
	updated := decodeOrder(t, resp)
	if len(updated.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(updated.Items))
	}
	if updated.Items[0].AmountCents != 12000 || updated.Items[1].AmountCents != 2500 {
		t.Fatalf("amounts = %d/%d, want 12000/2500", updated.Items[0].AmountCents, updated.Items[1].AmountCents)
	}
}

func TestHandlerAddItemsUnknownOrder(t *testing.T) {
	pool := testPool(t)
	srv := newOrderServer(t, pool)

	body := `{"items":[{"kind":"labor","description":"Arbeit","hours":1}]}`
	url := srv.URL + "/api/workshop/orders/999999999/items"
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST items: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
	if eb := decodeError(t, resp); eb.Error.Code != model.CodeNotFound {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeNotFound)
	}
}

func TestHandlerAddItemsInvalid(t *testing.T) {
	pool := testPool(t)
	vehicleID := seedVehicle(t, pool)
	srv := newOrderServer(t, pool)

	createResp := postJSON(t, srv.URL+"/api/workshop/orders",
		`{"vehicleId":`+itoa(vehicleID)+`,"requestedDate":"2026-11-05","problemDescription":"Inspektion"}`)
	created := decodeOrder(t, createResp)

	resp := postJSON(t, srv.URL+"/api/workshop/orders/"+itoa(created.ID)+"/items", `{"items":[]}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", resp.StatusCode)
	}
	if eb := decodeError(t, resp); eb.Error.Code != model.CodeValidation {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeValidation)
	}
}

package customer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// newCustomerServer mounts the three customer routes under their declared paths
// and verbs on a real PostgreSQL pool. The routes are mounted directly rather
// than through the shared router, so this test is unaffected by the auth guard
// another ticket fills in — it proves the handler's own behaviour.
func newCustomerServer(t *testing.T) (*httptest.Server, store.Pool) {
	t.Helper()
	pool := testPool(t)
	h := NewHandler(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/workshop/customers", h.Create)
	mux.HandleFunc("GET /api/workshop/customers/{id}", h.Get)
	mux.HandleFunc("PUT /api/workshop/customers/{id}", h.Update)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, pool
}

func decodeErrorBody(t *testing.T, body []byte) httpx.ErrorBody {
	t.Helper()
	var eb httpx.ErrorBody
	if err := json.Unmarshal(body, &eb); err != nil {
		t.Fatalf("error body is not JSON: %v (%s)", err, body)
	}
	return eb
}

func doJSON(t *testing.T, method, url string, payload any) *http.Response {
	t.Helper()
	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func TestHandlerCreateGetUpdateRoundTrip(t *testing.T) {
	srv, _ := newCustomerServer(t)
	email := uniqueEmail("handler-roundtrip")

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/workshop/customers",
		map[string]string{"name": "Anna Muster", "email": email, "phone": "+49 170 1234567"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST status = %d, want 201 (%s)", resp.StatusCode, body)
	}
	var created model.Customer
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == 0 || created.Name != "Anna Muster" || created.Email != email {
		t.Fatalf("created = %+v, want the stored customer", created)
	}

	getURL := srv.URL + "/api/workshop/customers/" + strconv.Itoa(created.ID)
	respGet := doJSON(t, http.MethodGet, getURL, nil)
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", respGet.StatusCode)
	}
	var fetched model.Customer
	if err := json.NewDecoder(respGet.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	if fetched != created {
		t.Fatalf("GET returned %+v, want %+v", fetched, created)
	}

	respPut := doJSON(t, http.MethodPut, getURL,
		map[string]string{"name": "Anna Beispiel", "email": email, "phone": "0151 999888"})
	defer respPut.Body.Close()
	if respPut.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", respPut.StatusCode)
	}
	var updated model.Customer
	if err := json.NewDecoder(respPut.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	if updated.Name != "Anna Beispiel" || updated.Phone != "0151 999888" {
		t.Fatalf("PUT returned %+v, want the changed values", updated)
	}
}

func TestHandlerValidationErrorIs422(t *testing.T) {
	srv, _ := newCustomerServer(t)

	cases := []struct {
		name    string
		payload map[string]string
		want    string
	}{
		{"missing name", map[string]string{"email": "valid@example.com", "phone": "0170"}, "name is required"},
		{"missing email", map[string]string{"name": "Anna", "phone": "0170"}, "email is required"},
		{"unusable email", map[string]string{"name": "Anna", "email": "nope", "phone": "0170"}, "email is invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, http.MethodPost, srv.URL+"/api/workshop/customers", tc.payload)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", resp.StatusCode)
			}
			raw, _ := io.ReadAll(resp.Body)
			eb := decodeErrorBody(t, raw)
			if eb.Error.Code != model.CodeValidation {
				t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeValidation)
			}
			if eb.Error.Message != tc.want {
				t.Fatalf("message = %q, want %q", eb.Error.Message, tc.want)
			}
		})
	}
}

func TestHandlerEmptyBodyIsNever500(t *testing.T) {
	srv, _ := newCustomerServer(t)

	resp := doJSON(t, http.MethodPost, srv.URL+"/api/workshop/customers", map[string]string{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestHandlerUnknownIDIs404(t *testing.T) {
	srv, _ := newCustomerServer(t)

	respGet := doJSON(t, http.MethodGet, srv.URL+"/api/workshop/customers/2147480000", nil)
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusNotFound {
		t.Fatalf("GET status = %d, want 404", respGet.StatusCode)
	}
	raw, _ := io.ReadAll(respGet.Body)
	if eb := decodeErrorBody(t, raw); eb.Error.Code != model.CodeNotFound {
		t.Fatalf("GET code = %q, want %q", eb.Error.Code, model.CodeNotFound)
	}

	respPut := doJSON(t, http.MethodPut, srv.URL+"/api/workshop/customers/2147480000",
		map[string]string{"name": "Anna", "email": "valid@example.com", "phone": "0170"})
	defer respPut.Body.Close()
	if respPut.StatusCode != http.StatusNotFound {
		t.Fatalf("PUT status = %d, want 404", respPut.StatusCode)
	}
}

// Package api wires the HTTP surface of the product: every route of the
// contract, its handler and the middleware stack around it.
package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"workshop/backend/internal/auth"
	"workshop/backend/internal/config"
	"workshop/backend/internal/customer"
	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/order"
	"workshop/backend/internal/public"
	"workshop/backend/internal/store"
	"workshop/backend/internal/vehicle"
	"workshop/backend/internal/workshop"
)

// Deps is what the router is built from.
type Deps struct {
	Config *config.Config
	Pool   store.Pool
}

// NewRouter builds the complete HTTP handler, with every route of the contract
// registered, the auth guard applied to the workshop area and the login rate
// limiter applied to the login route.
func NewRouter(deps Deps) http.Handler {
	origin := ""
	if deps.Config != nil {
		origin = deps.Config.CORSOrigin
	}

	customers := customer.NewHandler(deps.Pool)
	vehicles := vehicle.NewHandler(deps.Pool)
	orders := order.NewHandler(deps.Pool)
	pub := public.NewHandler(deps.Pool)
	ws := workshop.NewHandler(deps.Pool)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", health)

	mux.HandleFunc("POST /api/public/appointments", jsonBody(pub.CreateAppointment))
	mux.HandleFunc("GET /api/public/orders/{orderNumber}", pub.GetOrderStatus)

	mux.Handle("POST /api/auth/login", auth.RateLimit(jsonBody(auth.Login)))

	guard := func(h http.HandlerFunc) http.Handler {
		return auth.RequireAuth(jsonBody(h))
	}
	mux.Handle("POST /api/workshop/customers", guard(customers.Create))
	mux.Handle("GET /api/workshop/customers/{id}", auth.RequireAuth(http.HandlerFunc(customers.Get)))
	mux.Handle("PUT /api/workshop/customers/{id}", guard(customers.Update))

	mux.Handle("POST /api/workshop/vehicles", guard(vehicles.Create))
	mux.Handle("GET /api/workshop/vehicles/{id}", auth.RequireAuth(http.HandlerFunc(vehicles.Get)))
	mux.Handle("PUT /api/workshop/vehicles/{id}", guard(vehicles.Update))

	mux.Handle("POST /api/workshop/orders", guard(orders.Create))
	mux.Handle("GET /api/workshop/orders", auth.RequireAuth(http.HandlerFunc(ws.ListOrders)))
	mux.Handle("GET /api/workshop/orders/{id}", auth.RequireAuth(http.HandlerFunc(orders.Get)))
	mux.Handle("POST /api/workshop/orders/{id}/items", guard(orders.AddItems))
	mux.Handle("POST /api/workshop/orders/{id}/status", guard(orders.ChangeStatus))

	mux.Handle("GET /api/workshop/dashboard", auth.RequireAuth(http.HandlerFunc(ws.Dashboard)))

	mux.HandleFunc("/", notFound)

	handler := httpx.CORS(origin)(httpx.Logging()(httpx.Recover()(mux)))
	return handler
}

// health is the only fully implemented route of the skeleton.
func health(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// notFound answers every unrouted request with the unified error body.
func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, model.CodeNotFound, http.StatusNotFound, "resource not found")
}

// jsonBody rejects a malformed JSON body with 422 before it reaches the
// handler, so every write route shares one validation behaviour. It restores
// the body so the handler can read it again.
func jsonBody(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
			next(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, "could not read request body")
			return
		}
		// An empty body is not malformed JSON: the handler itself decides
		// whether a body is required. Only a body that IS present and cannot be
		// parsed is rejected here with 422.
		if trimmed := bytes.TrimSpace(body); len(trimmed) > 0 && !json.Valid(trimmed) {
			httpx.Error(w, model.CodeValidation, http.StatusUnprocessableEntity, "malformed JSON body")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next(w, r)
	}
}

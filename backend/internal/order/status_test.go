package order

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// seedEmployee inserts an employee to attribute a status change to and returns
// its id. It writes only to the employees table this flow reads.
func seedEmployee(t *testing.T, pool store.Pool) (int, string) {
	t.Helper()
	name := "Meister " + time.Now().Format("150405.000000000")
	var id int
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO employees (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		name, name+"@example.test", "x").Scan(&id); err != nil {
		t.Fatalf("insert employee: %v", err)
	}
	return id, name
}

// recordingNotifier records the completed-order notifications it receives so a
// test can assert that exactly one is sent for the move to completed and none
// for the other transitions.
type recordingNotifier struct {
	mu    sync.Mutex
	calls []notifyCall
}

type notifyCall struct {
	orderID     int
	orderNumber string
}

func (r *recordingNotifier) NotifyOrderCompleted(_ context.Context, orderID int, orderNumber string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, notifyCall{orderID: orderID, orderNumber: orderNumber})
	return nil
}

func (r *recordingNotifier) snapshot() []notifyCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]notifyCall(nil), r.calls...)
}

// useNotifier installs a notifier for the duration of the test and restores the
// package default afterwards.
func useNotifier(t *testing.T, n Notifier) {
	t.Helper()
	SetNotifier(n)
	t.Cleanup(func() { SetNotifier(nil) })
}

// changeOrFail calls ChangeStatus and requires it to succeed.
func changeOrFail(t *testing.T, ctx context.Context, pool store.Pool, id int, status string, employeeID int) model.Order {
	t.Helper()
	ord, err := ChangeStatus(ctx, pool, id, status, employeeID)
	if err != nil {
		t.Fatalf("ChangeStatus(%d, %q): %v", id, status, err)
	}
	return ord
}

func TestChangeStatusWalksTheAllowedChain(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)
	employeeID, employeeName := seedEmployee(t, pool)
	useNotifier(t, &recordingNotifier{})

	ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Bremsen")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if ord.Status != model.StatusRequested {
		t.Fatalf("initial status = %q, want %q", ord.Status, model.StatusRequested)
	}

	chain := []string{model.StatusConfirmed, model.StatusInProgress, model.StatusCompleted, model.StatusPickedUp}
	wantHistory := 1 // the initial requested entry

	for _, next := range chain {
		got := changeOrFail(t, ctx, pool, ord.ID, next, employeeID)
		wantHistory++

		if got.Status != next {
			t.Fatalf("status = %q, want %q", got.Status, next)
		}
		if len(got.History) != wantHistory {
			t.Fatalf("history length = %d, want %d", len(got.History), wantHistory)
		}

		// Every step appends one entry naming the new status, a UTC timestamp
		// and the acting employee.
		last := got.History[len(got.History)-1]
		if last.Status != next {
			t.Fatalf("last history status = %q, want %q", last.Status, next)
		}
		if _, err := time.Parse(time.RFC3339, last.ChangedAt); err != nil {
			t.Fatalf("history timestamp %q is not RFC3339: %v", last.ChangedAt, err)
		}
		if last.EmployeeName != employeeName {
			t.Fatalf("history employeeName = %q, want %q", last.EmployeeName, employeeName)
		}

		// The previous entry's status is the old status of this transition.
		if wantHistory >= 2 {
			previous := got.History[len(got.History)-2].Status
			if previous == last.Status {
				t.Fatalf("history did not record a change: %q -> %q", previous, last.Status)
			}
		}
	}
}

func TestChangeStatusRejectsEveryForbiddenTransition(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)
	employeeID, _ := seedEmployee(t, pool)
	useNotifier(t, &recordingNotifier{})

	for i, current := range model.StatusOrder {
		// Bring a fresh order to `current` by walking the allowed chain.
		ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Übergänge")
		if err != nil {
			t.Fatalf("CreateOrder: %v", err)
		}
		for _, step := range model.StatusOrder[1 : i+1] {
			changeOrFail(t, ctx, pool, ord.ID, step, employeeID)
		}

		for _, target := range model.StatusOrder {
			if model.CanTransition(current, target) {
				continue
			}
			t.Run(current+"_to_"+target, func(t *testing.T) {
				if _, err := ChangeStatus(ctx, pool, ord.ID, target, employeeID); !errors.Is(err, ErrInvalidTransition) {
					t.Fatalf("err = %v, want ErrInvalidTransition", err)
				}
				reloaded, err := LoadOrder(ctx, pool, ord.ID)
				if err != nil {
					t.Fatalf("LoadOrder: %v", err)
				}
				if reloaded.Status != current {
					t.Fatalf("status changed on a forbidden transition: %q, want %q", reloaded.Status, current)
				}
			})
		}
	}
}

func TestChangeStatusRejectsUnknownStatus(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)
	employeeID, _ := seedEmployee(t, pool)
	useNotifier(t, &recordingNotifier{})

	ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Unbekannt")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	for _, bad := range []string{"", "done", "COMPLETED", "fertig"} {
		if _, err := ChangeStatus(ctx, pool, ord.ID, bad, employeeID); !errors.Is(err, ErrValidation) {
			t.Fatalf("ChangeStatus(%q): err = %v, want ErrValidation", bad, err)
		}
	}
}

func TestChangeStatusUnknownOrder(t *testing.T) {
	pool := testPool(t)
	useNotifier(t, &recordingNotifier{})

	if _, err := ChangeStatus(context.Background(), pool, 999999999, model.StatusConfirmed, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestChangeStatusNotifiesExactlyOnceOnCompleted(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	vehicleID := seedVehicle(t, pool)
	employeeID, _ := seedEmployee(t, pool)

	rec := &recordingNotifier{}
	useNotifier(t, rec)

	ord, err := CreateOrder(ctx, pool, vehicleID, "2026-11-05", "Rechnung")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	changeOrFail(t, ctx, pool, ord.ID, model.StatusConfirmed, employeeID)
	if got := rec.snapshot(); len(got) != 0 {
		t.Fatalf("notified %d times for confirmed, want 0", len(got))
	}
	changeOrFail(t, ctx, pool, ord.ID, model.StatusInProgress, employeeID)
	if got := rec.snapshot(); len(got) != 0 {
		t.Fatalf("notified %d times for in_progress, want 0", len(got))
	}

	completed := changeOrFail(t, ctx, pool, ord.ID, model.StatusCompleted, employeeID)
	calls := rec.snapshot()
	if len(calls) != 1 {
		t.Fatalf("notified %d times for completed, want exactly 1", len(calls))
	}
	if calls[0].orderID != ord.ID || calls[0].orderNumber != completed.OrderNumber {
		t.Fatalf("notification = %+v, want order %d / %q", calls[0], ord.ID, completed.OrderNumber)
	}

	// completed -> completed is not an allowed transition, so it must not add a
	// second message.
	if _, err := ChangeStatus(ctx, pool, ord.ID, model.StatusCompleted, employeeID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("re-complete: err = %v, want ErrInvalidTransition", err)
	}
	if got := rec.snapshot(); len(got) != 1 {
		t.Fatalf("notified %d times after a rejected re-complete, want 1", len(got))
	}
}

// newStatusServer mounts the order handler including the status route, without
// the auth guard, so the HTTP mapping of the workflow is exercised against the
// real database.
func newStatusServer(t *testing.T, pool store.Pool) *httptest.Server {
	t.Helper()
	h := NewHandler(pool)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/workshop/orders", h.Create)
	mux.HandleFunc("GET /api/workshop/orders/{id}", h.Get)
	mux.HandleFunc("POST /api/workshop/orders/{id}/status", h.ChangeStatus)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestHandlerChangeStatus(t *testing.T) {
	pool := testPool(t)
	vehicleID := seedVehicle(t, pool)
	useNotifier(t, &recordingNotifier{})
	srv := newStatusServer(t, pool)

	createResp := postJSON(t, srv.URL+"/api/workshop/orders",
		`{"vehicleId":`+itoa(vehicleID)+`,"requestedDate":"2026-11-05","problemDescription":"Status"}`)
	created := decodeOrder(t, createResp)

	statusURL := srv.URL + "/api/workshop/orders/" + itoa(created.ID) + "/status"

	ok := postJSON(t, statusURL, `{"status":"confirmed"}`)
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("confirmed: status %d, want 200", ok.StatusCode)
	}
	if got := decodeOrder(t, ok); got.Status != model.StatusConfirmed {
		t.Fatalf("status = %q, want %q", got.Status, model.StatusConfirmed)
	}

	conflict := postJSON(t, statusURL, `{"status":"picked_up"}`)
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("forbidden transition: status %d, want 409", conflict.StatusCode)
	}
	if eb := decodeError(t, conflict); eb.Error.Code != model.CodeInvalidTransition {
		t.Fatalf("code = %q, want %q", eb.Error.Code, model.CodeInvalidTransition)
	}

	invalid := postJSON(t, statusURL, `{"status":"done"}`)
	if invalid.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unknown status: status %d, want 422", invalid.StatusCode)
	}

	missing := postJSON(t, srv.URL+"/api/workshop/orders/999999999/status", `{"status":"confirmed"}`)
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown order: status %d, want 404", missing.StatusCode)
	}
}

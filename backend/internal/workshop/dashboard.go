// Package workshop serves the authenticated workshop area: the order list and
// the dashboard. The dashboard aggregates the open-order counts, the orders
// completed today and the current month's invoice revenue in one statement.
package workshop

import (
	"net/http"
	"time"

	"workshop/backend/internal/httpx"
	"workshop/backend/internal/model"
	"workshop/backend/internal/store"
)

// Handler serves the workshop overview routes.
type Handler struct {
	Pool store.Pool
}

// NewHandler builds a workshop handler on the given pool.
func NewHandler(pool store.Pool) *Handler {
	return &Handler{Pool: pool}
}

// OpenCounts is the number of open orders per workflow status.
type OpenCounts struct {
	Requested  int `json:"requested"`
	Confirmed  int `json:"confirmed"`
	InProgress int `json:"in_progress"`
}

// DashboardResponse is the body of GET /api/workshop/dashboard. Every amount is
// a whole number of cents.
type DashboardResponse struct {
	OpenCounts             OpenCounts `json:"openCounts"`
	CompletedToday         int        `json:"completedToday"`
	MonthRevenueGrossCents int        `json:"monthRevenueGrossCents"`
}

// dashboardSQL reads all five figures in a single statement. A single statement
// sees one snapshot of the database, so the numbers are consistent with each
// other. The open counts read the denormalised status column of orders; the
// "completed today" count reads the status log (the only place a completion
// timestamp exists) and counts distinct orders; the revenue sums the gross
// amounts of the invoices created within the given window.
const dashboardSQL = `
SELECT
	(SELECT count(*) FROM orders WHERE status = $1),
	(SELECT count(*) FROM orders WHERE status = $2),
	(SELECT count(*) FROM orders WHERE status = $3),
	(SELECT count(DISTINCT order_id) FROM order_status_log
		WHERE status = $4 AND changed_at >= $5 AND changed_at < $6),
	(SELECT COALESCE(sum(gross_cents), 0) FROM invoices
		WHERE created_at >= $7 AND created_at < $8)`

// Dashboard handles GET /api/workshop/dashboard. It reports the open-order
// counts per status, the number of orders that reached completed within the
// current UTC day and the sum of the gross amounts of all invoices created in
// the current UTC month, in whole cents.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "database unavailable")
		return
	}

	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonthStart := monthStart.AddDate(0, 1, 0)

	var resp DashboardResponse
	err := h.Pool.QueryRow(r.Context(), dashboardSQL,
		model.StatusRequested,
		model.StatusConfirmed,
		model.StatusInProgress,
		model.StatusCompleted,
		todayStart,
		tomorrowStart,
		monthStart,
		nextMonthStart,
	).Scan(
		&resp.OpenCounts.Requested,
		&resp.OpenCounts.Confirmed,
		&resp.OpenCounts.InProgress,
		&resp.CompletedToday,
		&resp.MonthRevenueGrossCents,
	)
	if err != nil {
		httpx.Error(w, model.CodeInternal, http.StatusInternalServerError, "could not read dashboard metrics")
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

// Package model holds every shared type of the API contract: the data shapes
// exchanged over HTTP and stored in PostgreSQL, the order-status constants and
// the error codes. It imports nothing but the standard library so that every
// other package can depend on it without an import cycle.
package model

// Error codes. Every failure response carries exactly one of these in
// {"error":{"code":...,"message":...}}. httpx re-exports them so the HTTP layer
// and the domain layer never disagree about a spelling.
const (
	CodeValidation        = "validation_error"
	CodeNotFound          = "not_found"
	CodeInvalidTransition = "invalid_transition"
	CodeDuplicatePlate    = "duplicate_plate"
	CodeUnauthorized      = "unauthorized"
	CodeRateLimited       = "rate_limited"
	CodeInternal          = "internal_error"
)

// Order statuses. The workflow is a strict chain in this order:
// requested -> confirmed -> in_progress -> completed -> picked_up.
const (
	StatusRequested  = "requested"
	StatusConfirmed  = "confirmed"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusPickedUp   = "picked_up"
)

// StatusOrder is the ordered list of the allowed workflow statuses.
var StatusOrder = []string{
	StatusRequested,
	StatusConfirmed,
	StatusInProgress,
	StatusCompleted,
	StatusPickedUp,
}

// ValidStatus reports whether s is one of the known statuses.
func ValidStatus(s string) bool {
	for _, v := range StatusOrder {
		if v == s {
			return true
		}
	}
	return false
}

// NextStatus returns the status that follows current in the workflow, or
// false when current is unknown or already terminal.
func NextStatus(current string) (string, bool) {
	for i, v := range StatusOrder {
		if v == current {
			if i+1 < len(StatusOrder) {
				return StatusOrder[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

// CanTransition reports whether moving from one status to another is allowed.
// Only the single forward step in StatusOrder is permitted.
func CanTransition(from, to string) bool {
	next, ok := NextStatus(from)
	return ok && next == to
}

// Customer is a workshop customer. It never carries a password.
type Customer struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Vehicle is a customer's vehicle. Plate is the normalised representation.
type Vehicle struct {
	ID         int    `json:"id"`
	CustomerID int    `json:"customerId"`
	Plate      string `json:"plate"`
	Brand      string `json:"brand"`
	Model      string `json:"model"`
	MileageKm  int    `json:"mileageKm"`
}

// OrderItemKindLabor and OrderItemKindPart are the only item kinds.
const (
	OrderItemKindLabor = "labor"
	OrderItemKindPart  = "part"
)

// OrderItem is one position of an order. Hours/Quantity/UnitPriceCents are
// pointers because a labor position has no quantity and a part has no hours.
type OrderItem struct {
	ID             int      `json:"id"`
	Kind           string   `json:"kind"`
	Description    string   `json:"description"`
	Hours          *float64 `json:"hours"`
	Quantity       *int     `json:"quantity"`
	UnitPriceCents *int     `json:"unitPriceCents"`
	AmountCents    int      `json:"amountCents"`
}

// OrderItemInput is the request shape for POST /orders/{id}/items.
type OrderItemInput struct {
	Kind           string   `json:"kind"`
	Description    string   `json:"description"`
	Hours          *float64 `json:"hours"`
	Quantity       *int     `json:"quantity"`
	UnitPriceCents *int     `json:"unitPriceCents"`
}

// StatusEntry is one entry of an order's status history as shown in the
// workshop area. The public status response omits EmployeeName.
type StatusEntry struct {
	Status       string `json:"status"`
	ChangedAt    string `json:"changedAt"`
	EmployeeName string `json:"employeeName"`
}

// Order is an order with its positions and status history.
type Order struct {
	ID                 int           `json:"id"`
	OrderNumber        string        `json:"orderNumber"`
	VehicleID          int           `json:"vehicleId"`
	CustomerID         int           `json:"customerId"`
	Plate              string        `json:"plate"`
	Brand              string        `json:"brand"`
	Model              string        `json:"model"`
	Status             string        `json:"status"`
	RequestedDate      string        `json:"requestedDate"`
	ProblemDescription string        `json:"problemDescription"`
	CreatedAt          string        `json:"createdAt"`
	Items              []OrderItem   `json:"items"`
	History            []StatusEntry `json:"history"`
}

// OrderSummary is the compact shape used by the workshop order list.
type OrderSummary struct {
	ID            int    `json:"id"`
	OrderNumber   string `json:"orderNumber"`
	Plate         string `json:"plate"`
	Brand         string `json:"brand"`
	Model         string `json:"model"`
	Status        string `json:"status"`
	RequestedDate string `json:"requestedDate"`
	CreatedAt     string `json:"createdAt"`
}

// Employee is a workshop employee. The password hash is never part of it.
type Employee struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Claims is the payload of the issued JWT.
type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

// InvoiceItem is one position of an invoice.
type InvoiceItem struct {
	Kind           string   `json:"kind"`
	Description    string   `json:"description"`
	Hours          *float64 `json:"hours"`
	Quantity       *int     `json:"quantity"`
	UnitPriceCents *int     `json:"unitPriceCents"`
	AmountCents    int      `json:"amountCents"`
}

// Invoice is a generated invoice. All amounts are integer cents.
type Invoice struct {
	InvoiceNumber  string        `json:"invoiceNumber"`
	NetCents       int           `json:"netCents"`
	TaxRatePercent int           `json:"taxRatePercent"`
	TaxCents       int           `json:"taxCents"`
	GrossCents     int           `json:"grossCents"`
	Items          []InvoiceItem `json:"items"`
}

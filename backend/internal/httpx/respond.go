// Package httpx is the single place that knows how an HTTP response looks:
// the JSON helpers, the one error body every failure shares, and the middleware
// stack (CORS, request logging, panic recovery).
package httpx

import (
	"encoding/json"
	"log"
	"net/http"

	"workshop/backend/internal/model"
)

// Error codes re-exported from model so callers of httpx and model never have
// to spell a code twice.
const (
	CodeValidation        = model.CodeValidation
	CodeNotFound          = model.CodeNotFound
	CodeInvalidTransition = model.CodeInvalidTransition
	CodeDuplicatePlate    = model.CodeDuplicatePlate
	CodeUnauthorized      = model.CodeUnauthorized
	CodeRateLimited       = model.CodeRateLimited
	CodeInternal          = model.CodeInternal

	// CodeNotImplemented marks a route the skeleton has declared and wired but
	// whose feature a later ticket still owes. It answers 501 and is the only
	// code that is expected to disappear from the product.
	CodeNotImplemented = "not_implemented"
)

// ErrorDetail is the inner object of every error response.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorBody is the unified error response body of the whole API.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// JSON writes v as a JSON body with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: failed to encode response: %v", err)
	}
}

// Error writes the unified error body {"error":{"code":...,"message":...}} with
// the given status code. Every failure of the API goes through here.
func Error(w http.ResponseWriter, code string, status int, message string) {
	JSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// NotImplemented answers 501 for a route the skeleton declares but whose
// feature a later ticket implements. A stub answers 501, never 500, so a
// mid-sprint build is reportable rather than looking crashed.
func NotImplemented(w http.ResponseWriter, what string) {
	Error(w, CodeNotImplemented, http.StatusNotImplemented, what+" is not implemented yet")
}

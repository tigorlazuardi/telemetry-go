package terror

import (
	"fmt"
	"net/http"
)

// Code describes the category of an error with an HTTP status code and a
// human-readable string form.
type Code interface {
	// HTTPStatusCode returns the HTTP status code associated with this error code.
	HTTPStatusCode() int

	// String returns a human-readable description of the error code.
	String() string
}

// StatusCode is the built-in implementation of [Code] backed by an HTTP status
// integer.
type StatusCode int

// HTTPStatusCode implements [Code].
func (s StatusCode) HTTPStatusCode() int { return int(s) }

// String implements [Code].
func (s StatusCode) String() string {
	if text := http.StatusText(int(s)); text != "" {
		return text
	}
	return fmt.Sprintf("StatusCode(%d)", s)
}

// Predefined [StatusCode] values covering the most common HTTP error statuses.
const (
	CodeBadRequest          StatusCode = http.StatusBadRequest          // 400
	CodeUnauthorized        StatusCode = http.StatusUnauthorized        // 401
	CodeForbidden           StatusCode = http.StatusForbidden           // 403
	CodeNotFound            StatusCode = http.StatusNotFound            // 404
	CodeConflict            StatusCode = http.StatusConflict            // 409
	CodeUnprocessableEntity StatusCode = http.StatusUnprocessableEntity // 422
	CodeTooManyRequests     StatusCode = http.StatusTooManyRequests     // 429
	CodeInternal            StatusCode = http.StatusInternalServerError // 500
	CodeServiceUnavailable  StatusCode = http.StatusServiceUnavailable  // 503
	CodeGatewayTimeout      StatusCode = http.StatusGatewayTimeout      // 504
)

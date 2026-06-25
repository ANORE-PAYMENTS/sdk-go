package anore

import "fmt"

// APIError is returned when the API responds with a non-2xx status. Inspect Kind
// to branch on the specific failure, or use errors.As with this type.
type APIError struct {
	Status    int    // HTTP status code
	Kind      string // "validation" | "authentication" | "forbidden" | "not_found" | "server" | "api"
	Message   string // human-readable message from the API
	RequestID string // X-Request-Id header, if present
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("anore: HTTP %d — %s (request_id=%s)", e.Status, e.Message, e.RequestID)
	}
	return fmt.Sprintf("anore: HTTP %d — %s", e.Status, e.Message)
}

// Convenience predicates so callers don't have to compare Kind strings.
func (e *APIError) IsValidation() bool     { return e.Kind == "validation" }
func (e *APIError) IsAuthentication() bool { return e.Kind == "authentication" }
func (e *APIError) IsForbidden() bool      { return e.Kind == "forbidden" }
func (e *APIError) IsNotFound() bool       { return e.Kind == "not_found" }
func (e *APIError) IsServer() bool         { return e.Kind == "server" }

func apiErrorForStatus(status int, message, requestID string) *APIError {
	kind := "api"
	switch {
	case status == 400:
		kind = "validation"
	case status == 401:
		kind = "authentication"
	case status == 403:
		kind = "forbidden"
	case status == 404:
		kind = "not_found"
	case status >= 500:
		kind = "server"
	}
	return &APIError{Status: status, Kind: kind, Message: message, RequestID: requestID}
}

// ConnectionError wraps a network failure / timeout (surfaced after retries).
type ConnectionError struct{ Err error }

func (e *ConnectionError) Error() string { return "anore: could not reach API: " + e.Err.Error() }
func (e *ConnectionError) Unwrap() error { return e.Err }

// SignatureError is returned by ParseWebhook when the signature does not match.
type SignatureError struct{}

func (e *SignatureError) Error() string { return "anore: webhook signature verification failed" }

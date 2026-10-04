package anore

import "fmt"

type APIError struct {
	Status    int
	Kind      string
	Message   string
	RequestID string
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("anore: HTTP %d — %s (request_id=%s)", e.Status, e.Message, e.RequestID)
	}
	return fmt.Sprintf("anore: HTTP %d — %s", e.Status, e.Message)
}

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

type ConnectionError struct{ Err error }

func (e *ConnectionError) Error() string { return "anore: could not reach API: " + e.Err.Error() }
func (e *ConnectionError) Unwrap() error { return e.Err }

type SignatureError struct{}

func (e *SignatureError) Error() string { return "anore: webhook signature verification failed" }

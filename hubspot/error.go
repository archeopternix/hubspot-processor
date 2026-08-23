package hubspot

import (
	"fmt"
	"net/http"
)

// ErrorCategory identifies the broad class of a HubSpot API failure.
type ErrorCategory string

const (
	// ErrorCategoryValidation indicates that HubSpot rejected the request data.
	ErrorCategoryValidation ErrorCategory = "validation"
	// ErrorCategoryAuthentication indicates missing or invalid authentication.
	ErrorCategoryAuthentication ErrorCategory = "authentication"
	// ErrorCategoryAuthorization indicates insufficient permissions.
	ErrorCategoryAuthorization ErrorCategory = "authorization"
	// ErrorCategoryNotFound indicates that the requested HubSpot resource was not found.
	ErrorCategoryNotFound ErrorCategory = "not_found"
	// ErrorCategoryConflict indicates that the request conflicts with current HubSpot state.
	ErrorCategoryConflict ErrorCategory = "conflict"
	// ErrorCategoryRateLimit indicates that HubSpot rate-limited the request.
	ErrorCategoryRateLimit ErrorCategory = "rate_limit"
	// ErrorCategoryTimeout indicates that HubSpot returned a timeout response.
	ErrorCategoryTimeout ErrorCategory = "timeout"
	// ErrorCategoryServer indicates a HubSpot server-side failure.
	ErrorCategoryServer ErrorCategory = "server"
	// ErrorCategoryUnexpected indicates an unclassified non-successful HTTP response.
	ErrorCategoryUnexpected ErrorCategory = "unexpected"
)

// APIError describes a non-successful response from the HubSpot API.
type APIError struct {
	// Category is the normalized class of failure.
	Category ErrorCategory
	// Operation identifies the operation and resource that failed.
	Operation string
	// StatusCode is the HTTP status code returned by HubSpot.
	StatusCode int
	// Body contains at most 64 KiB of the trimmed error response body.
	Body string
}

// Error returns the caller-facing description of the API failure.
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	message := fmt.Sprintf(
		"HubSpot %s: returned %d %s",
		e.Operation,
		e.StatusCode,
		http.StatusText(e.StatusCode),
	)
	if e.Body != "" {
		message += ": " + e.Body
	}
	return message
}

func classifyHTTPStatus(statusCode int) ErrorCategory {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ErrorCategoryValidation
	case http.StatusUnauthorized:
		return ErrorCategoryAuthentication
	case http.StatusForbidden:
		return ErrorCategoryAuthorization
	case http.StatusNotFound:
		return ErrorCategoryNotFound
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return ErrorCategoryTimeout
	case http.StatusConflict:
		return ErrorCategoryConflict
	case http.StatusTooManyRequests:
		return ErrorCategoryRateLimit
	default:
		if statusCode >= http.StatusInternalServerError {
			return ErrorCategoryServer
		}
		return ErrorCategoryUnexpected
	}
}

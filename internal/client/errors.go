package client

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Error is a non-2xx response from the Coolify API.
//
// Coolify is a Laravel application, so error bodies are consistently
// {"message": "..."} with an extra {"errors": {"field": ["..."]}} map on 422.
type Error struct {
	Method     string
	Path       string
	StatusCode int
	Message    string
	// Validation holds per-field messages returned on HTTP 422.
	Validation map[string][]string
	// RetryAfter is populated from the Retry-After header on HTTP 429.
	RetryAfter time.Duration
	// Body is the raw response body, truncated, for diagnostics.
	Body string
}

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, e.Message)

	if len(e.Validation) > 0 {
		fields := make([]string, 0, len(e.Validation))
		for field := range e.Validation {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, field := range fields {
			fmt.Fprintf(&b, "\n  - %s: %s", field, strings.Join(e.Validation[field], "; "))
		}
	}

	return b.String()
}

// Retryable reports whether re-sending the same request could succeed. Only
// statuses that guarantee the request had no effect are retryable.
func (e *Error) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// IsNotFound reports whether err is a 404 from the API. Resources use this to
// detect drift (the object was deleted outside Terraform) and remove themselves
// from state instead of failing the run.
func IsNotFound(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// IsUnauthorized reports whether err is a 401/400 authentication failure. Coolify
// returns 400 "Invalid token." for a malformed token and 401 for a revoked one.
func IsUnauthorized(err error) bool {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusUnauthorized ||
		(apiErr.StatusCode == http.StatusBadRequest && strings.Contains(apiErr.Message, "token"))
}

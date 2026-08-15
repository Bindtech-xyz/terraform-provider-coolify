package client

import (
	"strings"
	"testing"
)

// TestErrorIncludesRawBody locks in a regression reported live by a real
// consumer: Error() only ever surfaced the decoded {"message": "..."}
// field, silently dropping the raw response whenever the actual culprit sat
// in front of Coolify (a proxy, an edge auth layer, or Coolify's own
// allowed_ips gate) and returned something the {message, errors} shape
// didn't capture. Diagnosing it required re-running the request with curl
// to see the body the provider already had, but never showed.
func TestErrorIncludesRawBody(t *testing.T) {
	e := &Error{
		Method:     "GET",
		Path:       "/version",
		StatusCode: 403,
		Message:    "You are not allowed to access the API.",
		Body:       `{"success":true,"message":"You are not allowed to access the API."}`,
	}
	got := e.Error()
	if !strings.Contains(got, e.Body) {
		t.Errorf("Error() = %q, want it to contain the raw body %q", got, e.Body)
	}
}

// TestErrorOmitsBodyWhenEmpty covers the ordinary case (no body captured,
// e.g. a transport-level failure) — nothing to append, no trailing "body:"
// line with nothing after it.
func TestErrorOmitsBodyWhenEmpty(t *testing.T) {
	e := &Error{Method: "GET", Path: "/version", StatusCode: 500, Message: "Internal Server Error"}
	got := e.Error()
	if strings.Contains(got, "body:") {
		t.Errorf("Error() = %q, want no body section when Body is empty", got)
	}
}

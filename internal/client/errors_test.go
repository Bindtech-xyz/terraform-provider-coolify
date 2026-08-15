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

// TestIsBlockedByAllowedIPsAndIsAPIDisabled locks in the two distinct 403s
// Coolify's ApiAllowed middleware can return — same status code, same
// {"success":true, "message":...} shape, disambiguated only by the exact
// message text. Both are 403s that IsUnauthorized (401/400) must not match,
// and each other's message must not match either — a consumer reported live
// that a generic "Unable to reach Coolify" for the allowed_ips case forced
// them to re-diagnose with curl; these let the provider name the layer.
func TestIsBlockedByAllowedIPsAndIsAPIDisabled(t *testing.T) {
	allowedIPs := &Error{StatusCode: 403, Message: "You are not allowed to access the API."}
	apiDisabled := &Error{StatusCode: 403, Message: "API is disabled."}
	other403 := &Error{StatusCode: 403, Message: "Forbidden."}
	unauthorized := &Error{StatusCode: 401, Message: "Unauthenticated."}

	if !IsBlockedByAllowedIPs(allowedIPs) {
		t.Error("IsBlockedByAllowedIPs should match the allowed_ips 403")
	}
	if IsBlockedByAllowedIPs(apiDisabled) || IsBlockedByAllowedIPs(other403) || IsBlockedByAllowedIPs(unauthorized) {
		t.Error("IsBlockedByAllowedIPs should only match its exact message")
	}

	if !IsAPIDisabled(apiDisabled) {
		t.Error("IsAPIDisabled should match the API-disabled 403")
	}
	if IsAPIDisabled(allowedIPs) || IsAPIDisabled(other403) || IsAPIDisabled(unauthorized) {
		t.Error("IsAPIDisabled should only match its exact message")
	}

	if IsUnauthorized(allowedIPs) || IsUnauthorized(apiDisabled) {
		t.Error("IsUnauthorized must not match either 403 — they're a different gate entirely")
	}
}

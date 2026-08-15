package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient spins up an httptest server and a Client pointed at it.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	if handler == nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unexpected call", http.StatusTeapot)
		})
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// newJSONServer serves a fixed JSON body on every request and returns its URL.
func newJSONServer(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestNewNormalizesEndpoint(t *testing.T) {
	for _, in := range []string{
		"https://coolify.example.com",
		"https://coolify.example.com/",
		"https://coolify.example.com/api/v1",
		"https://coolify.example.com/api/v1/",
	} {
		c, err := New(in, "tok")
		if err != nil {
			t.Fatalf("New(%q): %v", in, err)
		}
		if got, want := c.Endpoint(), "https://coolify.example.com/api/v1"; got != want {
			t.Errorf("New(%q).Endpoint() = %q, want %q", in, got, want)
		}
	}
}

func TestNewRejectsEmptyToken(t *testing.T) {
	if _, err := New("https://coolify.example.com", ""); err == nil {
		t.Fatal("New with empty token: expected error, got nil")
	}
}

func TestExtraHeadersAreSentAndCannotOverrideAuthorization(t *testing.T) {
	var gotHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		_, _ = w.Write([]byte(`"4.0.0"`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "real-token", WithExtraHeaders(map[string]string{
		"CF-Access-Client-Id":     "id123",
		"CF-Access-Client-Secret": "secret456",
		// A conflicting Authorization must not survive: the client's own
		// bearer token always wins, applied after extra headers.
		"Authorization": "Bearer attacker-supplied",
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Version(context.Background()); err != nil {
		t.Fatalf("Version: %v", err)
	}

	if got := gotHeaders.Get("CF-Access-Client-Id"); got != "id123" {
		t.Errorf("CF-Access-Client-Id = %q, want id123", got)
	}
	if got := gotHeaders.Get("CF-Access-Client-Secret"); got != "secret456" {
		t.Errorf("CF-Access-Client-Secret = %q, want secret456", got)
	}
	if want := "Bearer real-token"; gotHeaders.Get("Authorization") != want {
		t.Errorf("Authorization = %q, want %q (extra headers must not override it)", gotHeaders.Get("Authorization"), want)
	}
}

func TestAuthorizationHeaderIsSent(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`"4.0.0"`))
	}))

	if _, err := c.Version(context.Background()); err != nil {
		t.Fatalf("Version: %v", err)
	}
	if want := "Bearer test-token"; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
}

func TestCreateProjectFollowsUpWithRead(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/projects":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"abc123"}`))
		case "GET /api/v1/projects/abc123":
			_, _ = w.Write([]byte(`{"id":1,"uuid":"abc123","name":"demo","description":"d"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	name := "demo"
	p, err := c.CreateProject(context.Background(), ProjectRequest{Name: &name})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.UUID != "abc123" || p.Name != "demo" {
		t.Errorf("CreateProject = %+v, want uuid=abc123 name=demo", p)
	}
}

func TestNotFoundIsDetectable(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Resource not found."}`))
	}))

	_, err := c.GetProject(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetProject: expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound(%v) = false, want true", err)
	}
}

func TestValidationErrorsAreSurfaced(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation error.","errors":{"name":["The name field is required."]}}`))
	}))

	_, err := c.CreateProject(context.Background(), ProjectRequest{})
	if err == nil {
		t.Fatal("CreateProject: expected error, got nil")
	}
	want := "The name field is required."
	if got := err.Error(); !strings.Contains(got, want) {
		t.Errorf("error %q does not contain %q", got, want)
	}
}

func TestServerErrorsAreRetried(t *testing.T) {
	var calls int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`"4.0.0"`))
	}))

	if _, err := c.Version(context.Background()); err != nil {
		t.Fatalf("Version after retry: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

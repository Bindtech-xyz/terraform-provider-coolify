package client

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateServerFollowsUpWithRead(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/servers":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"srv1"}`))
		case "GET /api/v1/servers/srv1":
			_, _ = w.Write([]byte(`{"uuid":"srv1","name":"worker","ip":"203.0.113.10","port":22,"user":"root",
				"settings":{"is_reachable":true,"is_usable":true}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	name, ip := "worker", "203.0.113.10"
	s, err := c.CreateServer(context.Background(), ServerCreateRequest{Name: &name, IP: &ip})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	if s.UUID != "srv1" || !s.Settings.IsUsable {
		t.Errorf("CreateServer = %+v, want srv1 usable", s)
	}
}

func TestValidateServerPath(t *testing.T) {
	var gotPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"message":"queued"}`))
	}))
	if err := c.ValidateServer(context.Background(), "srv1"); err != nil {
		t.Fatalf("ValidateServer: %v", err)
	}
	if want := "/api/v1/servers/srv1/validate"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestListServers(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"uuid":"a"},{"uuid":"b"}]`))
	}))
	servers, err := c.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(servers) != 2 {
		t.Errorf("len = %d, want 2", len(servers))
	}
}

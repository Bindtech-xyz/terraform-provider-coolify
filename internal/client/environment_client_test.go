package client

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateEnvironmentReadsDetailsEndpoint(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"env1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"uuid":"env1","name":"staging"}`))
	}))

	name := "staging"
	env, err := c.CreateEnvironment(context.Background(), "proj1", EnvironmentRequest{Name: &name})
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	if env.Name != "staging" {
		t.Errorf("name = %q", env.Name)
	}
	// Details endpoint is /projects/{uuid}/{env}, not .../environments/{env}.
	if want := "GET /api/v1/projects/proj1/env1"; paths[1] != want {
		t.Errorf("read path = %q, want %q", paths[1], want)
	}
}

func TestDeleteEnvironmentPath(t *testing.T) {
	var gotPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"message":"deleted"}`))
	}))
	if err := c.DeleteEnvironment(context.Background(), "proj1", "staging"); err != nil {
		t.Fatalf("DeleteEnvironment: %v", err)
	}
	if want := "/api/v1/projects/proj1/environments/staging"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestDuplicateEnvironmentSurfaces409(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"Environment with this name already exists."}`))
	}))
	name := "prod"
	if _, err := c.CreateEnvironment(context.Background(), "proj1", EnvironmentRequest{Name: &name}); err == nil {
		t.Fatal("expected 409 error")
	}
}

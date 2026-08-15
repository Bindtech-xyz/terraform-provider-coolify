package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateApplicationSelectsEndpointPerType(t *testing.T) {
	cases := map[ApplicationType]string{
		ApplicationTypePublic:           "/api/v1/applications/public",
		ApplicationTypePrivateGithubApp: "/api/v1/applications/private-github-app",
		ApplicationTypePrivateDeployKey: "/api/v1/applications/private-deploy-key",
		ApplicationTypeDockerfile:       "/api/v1/applications/dockerfile",
		ApplicationTypeDockerImage:      "/api/v1/applications/dockerimage",
	}
	for appType, wantPath := range cases {
		var gotPath string
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"uuid":"app1"}`))
				return
			}
			_, _ = w.Write([]byte(`{"uuid":"app1","name":"web"}`))
		}))
		if _, err := c.CreateApplication(context.Background(), appType, ApplicationRequest{}); err != nil {
			t.Fatalf("CreateApplication(%s): %v", appType, err)
		}
		if gotPath != wantPath {
			t.Errorf("type %s → path %q, want %q", appType, gotPath, wantPath)
		}
	}
}

func TestCreateApplicationRejectsUnknownType(t *testing.T) {
	c := newTestClient(t, nil)
	if _, err := c.CreateApplication(context.Background(), "wat", ApplicationRequest{}); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestApplicationRequestOmitsUnsetFields(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"app1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"uuid":"app1"}`))
	}))

	repo := "https://github.com/acme/api"
	if _, err := c.CreateApplication(context.Background(), ApplicationTypePublic, ApplicationRequest{
		GitRepository: &repo,
	}); err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}
	if len(body) != 1 || body["git_repository"] != repo {
		t.Errorf("body = %v, want only git_repository", body)
	}
}

func TestApplicationLifecycleActions(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	ctx := context.Background()
	_ = c.StartApplication(ctx, "a")
	_ = c.StopApplication(ctx, "a")
	_ = c.RestartApplication(ctx, "a")
	want := []string{"/api/v1/applications/a/start", "/api/v1/applications/a/stop", "/api/v1/applications/a/restart"}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("action %d path = %q, want %q", i, paths[i], want[i])
		}
	}
}

package client

import (
	"context"
	"net/http"
	"testing"
)

func TestGithubAppAddressedByNumericID(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":42,"uuid":"gh1","name":"deployer","app_id":1,"installation_id":2}`))
		case http.MethodDelete:
			_, _ = w.Write([]byte(`{"message":"deleted"}`))
		default:
			_, _ = w.Write([]byte(`[{"id":42,"uuid":"gh1","name":"deployer"}]`))
		}
	}))

	ctx := context.Background()
	name := "deployer"
	app, err := c.CreateGithubApp(ctx, GithubAppRequest{Name: &name})
	if err != nil {
		t.Fatalf("CreateGithubApp: %v", err)
	}
	if app.ID != 42 {
		t.Errorf("id = %d", app.ID)
	}
	if _, err := c.GetGithubApp(ctx, 42); err != nil {
		t.Fatalf("GetGithubApp: %v", err)
	}
	if _, err := c.GetGithubApp(ctx, 99); !IsNotFound(err) {
		t.Errorf("missing app: IsNotFound = false (%v)", err)
	}
	if err := c.DeleteGithubApp(ctx, 42); err != nil {
		t.Fatalf("DeleteGithubApp: %v", err)
	}
	if last := paths[len(paths)-1]; last != "DELETE /api/v1/github-apps/42" {
		t.Errorf("delete path = %q", last)
	}
}

func TestGitlabAppList(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":7,"uuid":"gl1","name":"gitlab","html_url":"https://gitlab.com"}]`))
	}))
	app, err := c.GetGitlabApp(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetGitlabApp: %v", err)
	}
	if app.HTMLURL != "https://gitlab.com" {
		t.Errorf("html_url = %q", app.HTMLURL)
	}
}

func TestGithubRepositoriesUnwrapped(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/github-apps/42/repositories" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"repositories":[{"id":1,"name":"api","full_name":"acme/api","private":true}]}`))
	}))
	repos, err := c.ListGithubAppRepositories(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListGithubAppRepositories: %v", err)
	}
	if len(repos) != 1 || repos[0].FullName != "acme/api" {
		t.Errorf("repos = %+v", repos)
	}
}

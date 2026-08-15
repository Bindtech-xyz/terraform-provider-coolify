package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestCreateDatabaseUsesEnginePath(t *testing.T) {
	var gotPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"uuid":"db1","internal_db_url":"postgres://..."}`))
		default:
			w.Write([]byte(`{"uuid":"db1","name":"pg","image":"postgres:16"}`))
		}
	}))

	if _, err := c.CreateDatabase(context.Background(), DatabasePostgreSQL, DatabaseRequest{}); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if want := "/api/v1/databases/postgresql"; gotPath != want {
		t.Errorf("create path = %q, want %q", gotPath, want)
	}
}

func TestDeleteApplicationQueryFlags(t *testing.T) {
	var gotQuery string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Post-delete polling: report the teardown as finished.
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Resource not found."}`))
			return
		}
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"message":"deleted"}`))
	}))

	f := false
	if err := c.DeleteApplication(context.Background(), "app1", nil, &f, nil, nil); err != nil {
		t.Fatalf("DeleteApplication: %v", err)
	}
	if want := "delete_volumes=false"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
}

func TestEnvVarLifecyclePaths(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"uuid":"env1"}`))
		default:
			w.Write([]byte(`[{"uuid":"env1","key":"FOO","value":"bar"}]`))
		}
	}))

	key, value := "FOO", "bar"
	v, err := c.CreateEnvVar(context.Background(), EnvVarParentApplication, "app1", EnvVarRequest{Key: &key, Value: &value})
	if err != nil {
		t.Fatalf("CreateEnvVar: %v", err)
	}
	if v.Key != "FOO" {
		t.Errorf("key = %q, want FOO", v.Key)
	}
	if paths[0] != "POST /api/v1/applications/app1/envs" {
		t.Errorf("create path = %q", paths[0])
	}
}

func TestSharedEnvScopePaths(t *testing.T) {
	cases := map[string]SharedEnvScope{
		"/team/envs":                          {Kind: "team"},
		"/projects/p1/envs":                   {Kind: "project", ProjectUUID: "p1"},
		"/projects/p1/environments/prod/envs": {Kind: "environment", ProjectUUID: "p1", Environment: "prod"},
		"/servers/s1/envs":                    {Kind: "server", ServerUUID: "s1"},
	}
	for want, scope := range cases {
		got, err := scope.base()
		if err != nil {
			t.Fatalf("base(%v): %v", scope, err)
		}
		if got != want {
			t.Errorf("base(%v) = %q, want %q", scope, got, want)
		}
	}
	if _, err := (SharedEnvScope{Kind: "environment"}).base(); err == nil {
		t.Error("expected error for environment scope without project")
	}
}

func TestFetchServiceTemplates(t *testing.T) {
	c := newTestClient(t, nil)
	// Serve the catalog from a second httptest server (distinct from the API).
	catalog := `{"plausible":{"slogan":"Analytics","category":"analytics","port":"8000"},
	            "gitea":{"slogan":"Git hosting","category":"git","port":"3000"}}`
	srv := newJSONServer(t, catalog)

	templates, err := c.FetchServiceTemplates(context.Background(), srv)
	if err != nil {
		t.Fatalf("FetchServiceTemplates: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("len = %d, want 2", len(templates))
	}
	// Sorted by type.
	if templates[0].Type != "gitea" || templates[1].Type != "plausible" {
		t.Errorf("order = %s,%s; want gitea,plausible", templates[0].Type, templates[1].Type)
	}
	if templates[1].Category != "analytics" {
		t.Errorf("category = %q, want analytics", templates[1].Category)
	}
}

func TestUpdateEnvironmentReadsBackByNewName(t *testing.T) {
	var gets []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gets = append(gets, r.URL.Path)
		}
		if strings.Contains(r.URL.Path, "/environments/") && r.Method == http.MethodPatch {
			w.Write([]byte(`{}`))
			return
		}
		w.Write([]byte(`{"uuid":"e1","name":"staging2"}`))
	}))

	name := "staging2"
	env, err := c.UpdateEnvironment(context.Background(), "p1", "staging", EnvironmentRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateEnvironment: %v", err)
	}
	if env.Name != "staging2" {
		t.Errorf("name = %q, want staging2", env.Name)
	}
	if len(gets) != 1 || gets[0] != "/api/v1/projects/p1/staging2" {
		t.Errorf("read-back path = %v, want /api/v1/projects/p1/staging2", gets)
	}
}

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdateEnvVarsBulkWrapsData(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			if r.URL.Path != "/api/v1/applications/app1/envs/bulk" {
				t.Errorf("path = %s", r.URL.Path)
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`[{"uuid":"e1","key":"FOO","value":"bar"}]`))
	}))

	key, value := "FOO", "bar"
	vars, err := c.UpdateEnvVarsBulk(context.Background(), EnvVarParentApplication, "app1",
		[]EnvVarRequest{{Key: &key, Value: &value}})
	if err != nil {
		t.Fatalf("UpdateEnvVarsBulk: %v", err)
	}
	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("body = %v, want data array", body)
	}
	if len(vars) != 1 || vars[0].Key != "FOO" {
		t.Errorf("vars = %+v", vars)
	}
}

func TestDeployByTagAndUUID(t *testing.T) {
	var got string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.String()
		_, _ = w.Write([]byte(`{"deployments":[]}`))
	}))
	if err := c.Deploy(context.Background(), "app1,app2", "", true); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if want := "/api/v1/deploy?force=true&uuid=app1%2Capp2"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestListApplicationDeploymentsWrapped(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"deployments":[{"deployment_uuid":"d1","status":"finished"}]}`))
	}))
	deployments, err := c.ListApplicationDeployments(context.Background(), "app1")
	if err != nil {
		t.Fatalf("ListApplicationDeployments: %v", err)
	}
	if len(deployments) != 1 || deployments[0].Status != "finished" {
		t.Errorf("deployments = %+v", deployments)
	}
}

func TestBackupExecutionsWrapped(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"executions":[{"uuid":"x1","status":"success","size":1024}]}`))
	}))
	execs, err := c.ListBackupExecutions(context.Background(), "db1", "bk1")
	if err != nil {
		t.Fatalf("ListBackupExecutions: %v", err)
	}
	if len(execs) != 1 || execs[0].Size != 1024 {
		t.Errorf("executions = %+v", execs)
	}
}

func TestServerReads(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/servers/s1/domains":
			_, _ = w.Write([]byte(`[{"ip":"1.2.3.4","domains":["a.example.com","b.example.com"]}]`))
		case "/api/v1/servers/s1/resources":
			_, _ = w.Write([]byte(`[{"uuid":"r1","name":"api","type":"application","status":"running"}]`))
		case "/api/v1/health":
			_, _ = w.Write([]byte(`OK`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	ctx := context.Background()
	domains, err := c.GetServerDomains(ctx, "s1")
	if err != nil || len(domains) != 1 || len(domains[0].Domains) != 2 {
		t.Errorf("domains = %+v (%v)", domains, err)
	}
	resources, err := c.GetServerResources(ctx, "s1")
	if err != nil || len(resources) != 1 || resources[0].Type != "application" {
		t.Errorf("resources = %+v (%v)", resources, err)
	}
	if !c.Healthy(ctx) {
		t.Error("Healthy = false, want true")
	}
}

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
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
		_, _ = w.Write([]byte(`{"deployments":[{"message":"Deployment queued.","resource_uuid":"app1","deployment_uuid":"d1"}]}`))
	}))
	results, err := c.Deploy(context.Background(), "app1,app2", "", true)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if want := "/api/v1/deploy?force=true&uuid=app1%2Capp2"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
	if len(results) != 1 || results[0].DeploymentUUID != "d1" {
		t.Errorf("results = %+v, want one entry with deployment_uuid d1", results)
	}
}

// TestDeployByTagUsesDetailsEnvelope locks in a regression found live:
// deploying by tag returns {"details": [...], "message": [...]} instead of
// the by-uuid path's {"deployments": [...]} — decoding only "deployments"
// silently produced zero results for every by-tag deploy, no error at all.
func TestDeployByTagUsesDetailsEnvelope(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"details":[{"resource_uuid":"app1","deployment_uuid":"d1"}],"message":["Application app1 deployment queued."]}`))
	}))
	results, err := c.Deploy(context.Background(), "", "production", false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 || results[0].DeploymentUUID != "d1" || results[0].ResourceUUID != "app1" {
		t.Errorf("results = %+v, want one entry with resource_uuid app1, deployment_uuid d1", results)
	}
}

// TestDeployResolvesSkippedDeploymentUUID locks in a real upstream bug found
// live: coolify_application's instant_deploy already starts a deployment at
// create time, so a coolify_deployment triggered right after it hits
// queue_application_deployment's "skipped" branch (a deployment for the same
// application+commit is already queued/in_progress). Confirmed against the
// controller source (bootstrap/helpers/applications.php +
// DeployController::deploy_resource): the "skipped" result carries the REAL
// existing deployment's uuid, but deploy_resource discards it and returns a
// deployment_uuid generated *before* the queue check instead -- one that was
// never written to ApplicationDeploymentQueue and 404s on every GET forever,
// not a transient race. Deploy must detect the skip message and resolve the
// real uuid via the application's own deployment list instead of trusting
// what /deploy returned.
func TestDeployResolvesSkippedDeploymentUUID(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deploy":
			_, _ = w.Write([]byte(`{"deployments":[{"message":"Deployment already queued for this commit.","resource_uuid":"app1","deployment_uuid":"phantom-never-persisted"}]}`))
		case "/api/v1/deployments/applications/app1":
			_, _ = w.Write([]byte(`{"deployments":[{"deployment_uuid":"real-in-progress","status":"in_progress"},{"deployment_uuid":"older","status":"finished"}]}`))
		}
	}))
	results, err := c.Deploy(context.Background(), "app1", "", false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 || results[0].DeploymentUUID != "real-in-progress" {
		t.Errorf("results = %+v, want the resolved real-in-progress uuid, not the phantom one", results)
	}
}

// TestDeployResolvesSkippedDeploymentUUIDFallsBackToNewest covers the race
// window already having closed by the time we resolve: no in_progress/queued
// entry remains, so the most recent deployment overall (newest-first per
// Application::deployments()'s default sort) is still a better answer than
// a uuid that was never persisted.
func TestDeployResolvesSkippedDeploymentUUIDFallsBackToNewest(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deploy":
			_, _ = w.Write([]byte(`{"deployments":[{"message":"Deployment already queued for this commit.","resource_uuid":"app1","deployment_uuid":"phantom"}]}`))
		case "/api/v1/deployments/applications/app1":
			_, _ = w.Write([]byte(`{"deployments":[{"deployment_uuid":"finished-one","status":"finished"}]}`))
		}
	}))
	results, err := c.Deploy(context.Background(), "app1", "", false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 || results[0].DeploymentUUID != "finished-one" {
		t.Errorf("results = %+v, want fallback to the newest deployment", results)
	}
}

// TestDeployLeavesOrdinaryResultsAlone confirms the resolution only ever
// triggers on the exact skip message — an ordinary successful queue result
// keeps its own deployment_uuid untouched.
func TestDeployLeavesOrdinaryResultsAlone(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"deployments":[{"message":"Application app1 deployment queued.","resource_uuid":"app1","deployment_uuid":"d1"}]}`))
	}))
	results, err := c.Deploy(context.Background(), "app1", "", false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 || results[0].DeploymentUUID != "d1" {
		t.Errorf("results = %+v, want the original deployment_uuid untouched", results)
	}
}

// TestWaitForDeploymentCompletionTreatsCancelledAsTerminal confirms
// "cancelled-by-user" (ApplicationDeploymentStatus's fifth value, alongside
// queued/in_progress/finished/failed) is terminal — without this, a
// cancelled deployment polled forever until the timeout instead of failing
// immediately with a clear reason.
func TestWaitForDeploymentCompletionTreatsCancelledAsTerminal(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"deployment_uuid":"d1","status":"cancelled-by-user"}`))
	}))
	_, err := c.WaitForDeploymentCompletion(context.Background(), "d1", 5*time.Second)
	if err == nil {
		t.Fatal("expected an error for a cancelled deployment")
	}
}

func TestGetDeployment(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/api/v1/deployments/d1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_, _ = w.Write([]byte(`{"deployment_uuid":"d1","status":"finished"}`))
	}))
	d, err := c.GetDeployment(context.Background(), "d1")
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if d.Status != "finished" {
		t.Errorf("status = %q, want finished", d.Status)
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

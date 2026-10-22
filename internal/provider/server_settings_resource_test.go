package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestApplyProxyConfigurationSkipsRestartWhenUnchanged locks in the reason
// applyProxyConfiguration takes a prior model at all: a restart briefly
// interrupts every ingress request through the proxy, so it must not run on
// an apply that only touched an unrelated proxy.* field (e.g.
// generate_exact_labels) while `configuration` itself stayed the same.
func TestApplyProxyConfigurationSkipsRestartWhenUnchanged(t *testing.T) {
	var saveCalled, restartCalled bool
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/servers/srv1/proxy/configuration":
			saveCalled = true
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/srv1/proxy/restart":
			restartCalled = true
		}
	}))
	r := &serverSettingsResource{client: c}

	same := types.StringValue("services:\n  traefik:\n    container_name: coolify-proxy\n")
	plan := &serverProxySettingsModel{Configuration: same}
	prior := &serverProxySettingsModel{Configuration: same}

	var diags diag.Diagnostics
	r.applyProxyConfiguration(context.Background(), "srv1", plan, prior, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if saveCalled || restartCalled {
		t.Errorf("unchanged configuration must not save or restart, got save=%v restart=%v", saveCalled, restartCalled)
	}
}

// TestApplyProxyConfigurationSavesAndRestartsWhenChanged covers the normal
// path: a genuinely new configuration must be saved AND followed by a
// restart, since SaveProxyConfiguration only rewrites docker-compose.yml on
// disk and never restarts the container itself.
func TestApplyProxyConfigurationSavesAndRestartsWhenChanged(t *testing.T) {
	var saveCalled, restartCalled bool
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/servers/srv1/proxy/configuration":
			saveCalled = true
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/srv1/proxy/restart":
			restartCalled = true
		}
	}))
	r := &serverSettingsResource{client: c}

	plan := &serverProxySettingsModel{Configuration: types.StringValue("new config")}
	prior := &serverProxySettingsModel{Configuration: types.StringValue("old config")}

	var diags diag.Diagnostics
	r.applyProxyConfiguration(context.Background(), "srv1", plan, prior, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !saveCalled || !restartCalled {
		t.Errorf("changed configuration must save and restart, got save=%v restart=%v", saveCalled, restartCalled)
	}
}

// TestApplyProxyConfigurationCreateAlwaysApplies covers Create, where prior
// is nil: a configured value must always be applied since there is nothing
// to compare it against yet.
func TestApplyProxyConfigurationCreateAlwaysApplies(t *testing.T) {
	var saveCalled, restartCalled bool
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/servers/srv1/proxy/configuration":
			saveCalled = true
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/srv1/proxy/restart":
			restartCalled = true
		}
	}))
	r := &serverSettingsResource{client: c}

	plan := &serverProxySettingsModel{Configuration: types.StringValue("first config")}

	var diags diag.Diagnostics
	r.applyProxyConfiguration(context.Background(), "srv1", plan, nil, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !saveCalled || !restartCalled {
		t.Errorf("a first-time configured value must save and restart, got save=%v restart=%v", saveCalled, restartCalled)
	}
}

// TestApplyProxyConfigurationRestartFailureIsWarningNotError locks in that a
// failed restart downgrades to a warning: the configuration is already
// persisted to the database and disk, so the resource must not be left
// untracked in state over a container restart that can simply be retried.
func TestApplyProxyConfigurationRestartFailureIsWarningNotError(t *testing.T) {
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/servers/srv1/proxy/configuration":
			return
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/srv1/proxy/restart":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"Internal Server Error"}`))
		}
	}))
	r := &serverSettingsResource{client: c}

	plan := &serverProxySettingsModel{Configuration: types.StringValue("new config")}

	var diags diag.Diagnostics
	r.applyProxyConfiguration(context.Background(), "srv1", plan, nil, &diags)

	if diags.HasError() {
		t.Fatalf("a failed restart must be a warning, not an error: %v", diags)
	}
	if diags.WarningsCount() == 0 {
		t.Error("expected a warning naming the restart failure")
	}
}

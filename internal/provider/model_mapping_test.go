package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

func TestProjectToModelPreservesNullDescription(t *testing.T) {
	api := &client.Project{UUID: "p1", Name: "shop", Description: ""}

	// Never configured: stays null (no permanent diff).
	m := projectToModel(api, projectResourceModel{Description: types.StringNull()})
	if !m.Description.IsNull() {
		t.Error("description must stay null when never configured")
	}

	// Configured then cleared server-side: adopts "".
	m = projectToModel(api, projectResourceModel{Description: types.StringValue("old")})
	if m.Description.IsNull() {
		t.Error("configured description must not become null")
	}
}

func TestServerToModelCarriesWriteOnlyFields(t *testing.T) {
	api := &client.Server{
		UUID: "s1", Name: "worker", IP: "1.2.3.4", Port: 22, User: "root", ProxyType: "traefik",
		Settings: &client.ServerSetting{IsReachable: true, IsUsable: true, IsBuildServer: false},
	}
	prior := serverResourceModel{
		PrivateKeyUUID:  types.StringValue("key1"),
		InstantValidate: types.BoolValue(true),
	}
	m := serverToModel(api, prior)
	// The API never echoes these back; state must keep the configured values.
	if m.PrivateKeyUUID.ValueString() != "key1" || !m.InstantValidate.ValueBool() {
		t.Errorf("write-only fields lost: %+v", m)
	}
	if !m.IsReachable.ValueBool() || !m.IsUsable.ValueBool() {
		t.Error("settings flags must be adopted")
	}
}

func TestEnvVarToModelKeepsHiddenValue(t *testing.T) {
	api := &client.EnvVar{UUID: "e1", Key: "TOKEN", Value: "", IsRuntime: true, IsBuildtime: true}
	prior := envVarResourceModel{Value: types.StringValue("configured-secret")}
	m := envVarToModel(api, prior)
	if m.Value.ValueString() != "configured-secret" {
		t.Error("hidden value must keep the configured secret")
	}
	if !m.IsRuntime.ValueBool() || !m.IsBuildtime.ValueBool() {
		t.Error("flags must be adopted from the API")
	}
}

func TestSharedEnvScopeMapping(t *testing.T) {
	m := sharedEnvVarResourceModel{
		Scope:       types.StringValue("environment"),
		ProjectUUID: types.StringValue("p1"),
		Environment: types.StringValue("prod"),
	}
	scope := sharedEnvScope(m)
	if scope.Kind != "environment" || scope.ProjectUUID != "p1" || scope.Environment != "prod" {
		t.Errorf("scope = %+v", scope)
	}
}

func TestGithubAppToModelKeepsSecrets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":7,"uuid":"key-uuid-7"}]`))
	}))
	t.Cleanup(srv.Close)
	c, err := client.New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	api := &client.GithubApp{ID: 42, UUID: "gh1", Name: "deployer", ClientID: "", PrivateKeyID: 7}
	prior := githubAppResourceModel{
		ClientID:      types.StringValue("Iv1.abc"),
		ClientSecret:  types.StringValue("s3cret"),
		WebhookSecret: types.StringValue("hook"),
	}
	r := &githubAppResource{client: c}
	m, diags := r.githubAppToModel(context.Background(), api, prior)
	if diags.HasError() {
		t.Fatalf("githubAppToModel: %v", diags)
	}
	if m.ClientID.ValueString() != "Iv1.abc" {
		t.Error("hidden client_id must keep the configured value")
	}
	if m.ClientSecret.ValueString() != "s3cret" || m.WebhookSecret.ValueString() != "hook" {
		t.Error("secrets must survive the round trip")
	}
	if m.ID.ValueInt64() != 42 {
		t.Error("numeric id must be adopted")
	}
	if m.PrivateKeyUUID.ValueString() != "key-uuid-7" {
		t.Errorf("private_key_uuid must be resolved from private_key_id, got %q", m.PrivateKeyUUID.ValueString())
	}
}

// TestGithubAppToModelAdoptsSecretsWhenReadable covers the other half of
// keepPriorIfHidden: a token with the read:sensitive ability gets real
// secrets back from Coolify (confirmed live: GithubApp.$hidden is overridden
// per-request by removeSensitiveData()'s makeVisible() call, so this isn't
// speculative), and state must adopt them — this is what makes import
// producible without stashing client_secret/webhook_secret anywhere first.
func TestGithubAppToModelAdoptsSecretsWhenReadable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":7,"uuid":"key-uuid-7"}]`))
	}))
	t.Cleanup(srv.Close)
	c, err := client.New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	api := &client.GithubApp{
		ID: 42, UUID: "gh1", Name: "deployer",
		ClientSecret: "real-secret", WebhookSecret: "real-hook", PrivateKeyID: 7,
	}
	// Freshly imported: no prior configuration to fall back to.
	r := &githubAppResource{client: c}
	m, diags := r.githubAppToModel(context.Background(), api, githubAppResourceModel{})
	if diags.HasError() {
		t.Fatalf("githubAppToModel: %v", diags)
	}
	if m.ClientSecret.ValueString() != "real-secret" {
		t.Errorf("client_secret = %q, want it adopted from the API", m.ClientSecret.ValueString())
	}
	if m.WebhookSecret.ValueString() != "real-hook" {
		t.Errorf("webhook_secret = %q, want it adopted from the API", m.WebhookSecret.ValueString())
	}
}

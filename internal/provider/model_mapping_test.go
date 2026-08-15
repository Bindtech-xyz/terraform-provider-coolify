package provider

import (
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
	api := &client.GithubApp{ID: 42, UUID: "gh1", Name: "deployer", ClientID: ""}
	prior := githubAppResourceModel{
		ClientID:      types.StringValue("Iv1.abc"),
		ClientSecret:  types.StringValue("s3cret"),
		WebhookSecret: types.StringValue("hook"),
	}
	m := githubAppToModel(api, prior)
	if m.ClientID.ValueString() != "Iv1.abc" {
		t.Error("hidden client_id must keep the configured value")
	}
	if m.ClientSecret.ValueString() != "s3cret" || m.WebhookSecret.ValueString() != "hook" {
		t.Error("secrets must survive the round trip")
	}
	if m.ID.ValueInt64() != 42 {
		t.Error("numeric id must be adopted")
	}
}

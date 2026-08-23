package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestApplicationTypeDerivation locks in the source-mode discrimination rules:
// which attributes select which Coolify create endpoint.
func TestApplicationTypeDerivation(t *testing.T) {
	str := types.StringValue

	cases := []struct {
		name  string
		model applicationResourceModel
		want  client.ApplicationType
	}{
		{
			name:  "public git",
			model: applicationResourceModel{GitRepository: str("https://github.com/a/b")},
			want:  client.ApplicationTypePublic,
		},
		{
			name: "deploy key wins over public",
			model: applicationResourceModel{
				GitRepository:  str("git@github.com:a/b.git"),
				PrivateKeyUUID: str("key1"),
			},
			want: client.ApplicationTypePrivateDeployKey,
		},
		{
			name: "github app wins over public",
			model: applicationResourceModel{
				GitRepository: str("a/b"),
				GithubAppUUID: str("gh1"),
			},
			want: client.ApplicationTypePrivateGithubApp,
		},
		{
			name:  "dockerfile",
			model: applicationResourceModel{Dockerfile: str("RlJPTQ==")},
			want:  client.ApplicationTypeDockerfile,
		},
		{
			name:  "docker image",
			model: applicationResourceModel{DockerRegistryImageName: str("nginx")},
			want:  client.ApplicationTypeDockerImage,
		},
	}

	for _, tc := range cases {
		if got := applicationType(tc.model); got != tc.want {
			t.Errorf("%s: applicationType = %s, want %s", tc.name, got, tc.want)
		}
	}
}

// TestApplicationToRequestPlacementGating ensures placement/source identifiers
// are only sent on create, never on update (the API rejects them there).
func TestApplicationToRequestPlacementGating(t *testing.T) {
	model := applicationResourceModel{
		ProjectUUID:    types.StringValue("proj1"),
		ServerUUID:     types.StringValue("srv1"),
		PrivateKeyUUID: types.StringValue("key1"),
		GitRepository:  types.StringValue("git@github.com:a/b.git"),
		Name:           types.StringValue("api"),
	}

	create := applicationToRequest(model, true)
	if create.ProjectUUID == nil || create.ServerUUID == nil || create.PrivateKeyUUID == nil {
		t.Error("create request must include placement fields")
	}

	update := applicationToRequest(model, false)
	if update.ProjectUUID != nil || update.ServerUUID != nil || update.PrivateKeyUUID != nil {
		t.Error("update request must not include placement fields")
	}
	if update.Name == nil || *update.Name != "api" {
		t.Error("update request must keep mutable fields")
	}
}

// TestApplicationToRequestOmitsDockerfileOnUpdate locks in a regression
// reported live against Coolify 4.3.2: ApplicationsController::update_by_uuid's
// allowed-fields whitelist omits `dockerfile` entirely (dockerfile_location and
// dockerfile_target_build are there, dockerfile itself isn't — confirmed
// against the controller source, and store()'s own whitelist does include it),
// so any PATCH carrying it 422s with "This field is not allowed.", even when
// the value is unchanged. dockerfile must only ever be sent on create; the
// schema's RequiresReplace plan modifier means a real content change goes
// through Create+Delete, never Update, so it's safe to omit unconditionally.
func TestApplicationToRequestOmitsDockerfileOnUpdate(t *testing.T) {
	model := applicationResourceModel{
		Name:       types.StringValue("api"),
		Dockerfile: types.StringValue("RlJPTQ=="),
	}

	create := applicationToRequest(model, true)
	if create.Dockerfile == nil || *create.Dockerfile != "RlJPTQ==" {
		t.Error("create request must include dockerfile")
	}

	update := applicationToRequest(model, false)
	if update.Dockerfile != nil {
		t.Error("update request must never include dockerfile")
	}
}

// TestApplicationToRequestDockerfileLocationSentBothWays confirms
// dockerfile_location has no create/update whitelist mismatch (unlike
// dockerfile and dockerfile_target_build) — Coolify accepts it on both
// endpoints, so it belongs in the base request, sent unconditionally.
func TestApplicationToRequestDockerfileLocationSentBothWays(t *testing.T) {
	model := applicationResourceModel{
		Name:               types.StringValue("api"),
		DockerfileLocation: types.StringValue("/docker/Dockerfile.web"),
	}

	for _, create := range []bool{true, false} {
		req := applicationToRequest(model, create)
		if req.DockerfileLocation == nil || *req.DockerfileLocation != "/docker/Dockerfile.web" {
			t.Errorf("dockerfile_location must be sent regardless of create=%v", create)
		}
	}
}

// TestApplicationToRequestDockerfileTargetBuildUpdateOnly locks in the
// inverse whitelist mismatch from dockerfile: Coolify's create endpoints all
// reject dockerfile_target_build (confirmed against the controller source —
// absent from every create allowed-fields list, present only in
// update_by_uuid's), so it must only ever be sent on update. Create() applies
// a configured value via a follow-up UpdateApplication call instead.
func TestApplicationToRequestDockerfileTargetBuildUpdateOnly(t *testing.T) {
	model := applicationResourceModel{
		Name:                  types.StringValue("api"),
		DockerfileTargetBuild: types.StringValue("release"),
	}

	create := applicationToRequest(model, true)
	if create.DockerfileTargetBuild != nil {
		t.Error("create request must never include dockerfile_target_build")
	}

	update := applicationToRequest(model, false)
	if update.DockerfileTargetBuild == nil || *update.DockerfileTargetBuild != "release" {
		t.Error("update request must include dockerfile_target_build")
	}
}

// TestApplicationToRequestManualWebhookSecretsSentBothWays confirms the four
// manual_webhook_secret_* fields have no create/update whitelist mismatch
// (confirmed against the controller source — present in both allowed-fields
// lists), unlike dockerfile/dockerfile_target_build, so they belong in the
// base request, sent unconditionally.
func TestApplicationToRequestManualWebhookSecretsSentBothWays(t *testing.T) {
	model := applicationResourceModel{
		Name:                         types.StringValue("api"),
		ManualWebhookSecretGithub:    types.StringValue("gh-secret"),
		ManualWebhookSecretGitlab:    types.StringValue("gl-secret"),
		ManualWebhookSecretBitbucket: types.StringValue("bb-secret"),
		ManualWebhookSecretGitea:     types.StringValue("ge-secret"),
	}

	for _, create := range []bool{true, false} {
		req := applicationToRequest(model, create)
		if req.ManualWebhookSecretGithub == nil || *req.ManualWebhookSecretGithub != "gh-secret" ||
			req.ManualWebhookSecretGitlab == nil || *req.ManualWebhookSecretGitlab != "gl-secret" ||
			req.ManualWebhookSecretBitbucket == nil || *req.ManualWebhookSecretBitbucket != "bb-secret" ||
			req.ManualWebhookSecretGitea == nil || *req.ManualWebhookSecretGitea != "ge-secret" {
			t.Errorf("manual webhook secrets must be sent regardless of create=%v, got %+v", create, req)
		}
	}
}

// TestApplicationToRequestHealthCheckSentBothWays confirms all 14
// health_check_* fields have no create/update whitelist mismatch (confirmed
// against the controller source — present in both allowed-fields lists), so
// they belong in the base request, sent unconditionally.
func TestApplicationToRequestHealthCheckSentBothWays(t *testing.T) {
	model := applicationResourceModel{
		Name:                    types.StringValue("api"),
		HealthCheckEnabled:      types.BoolValue(true),
		HealthCheckPath:         types.StringValue("/health"),
		HealthCheckPort:         types.StringValue("8080"),
		HealthCheckHost:         types.StringValue("127.0.0.1"),
		HealthCheckMethod:       types.StringValue("GET"),
		HealthCheckReturnCode:   types.Int64Value(200),
		HealthCheckScheme:       types.StringValue("http"),
		HealthCheckResponseText: types.StringValue("ok"),
		HealthCheckInterval:     types.Int64Value(30),
		HealthCheckTimeout:      types.Int64Value(10),
		HealthCheckRetries:      types.Int64Value(3),
		HealthCheckStartPeriod:  types.Int64Value(120),
		HealthCheckType:         types.StringValue("cmd"),
		HealthCheckCommand:      types.StringValue("wget -q -O- localhost/health"),
	}

	for _, create := range []bool{true, false} {
		req := applicationToRequest(model, create)
		if req.HealthCheckEnabled == nil || !*req.HealthCheckEnabled ||
			req.HealthCheckPath == nil || *req.HealthCheckPath != "/health" ||
			req.HealthCheckPort == nil || *req.HealthCheckPort != "8080" ||
			req.HealthCheckHost == nil || *req.HealthCheckHost != "127.0.0.1" ||
			req.HealthCheckMethod == nil || *req.HealthCheckMethod != "GET" ||
			req.HealthCheckReturnCode == nil || *req.HealthCheckReturnCode != 200 ||
			req.HealthCheckScheme == nil || *req.HealthCheckScheme != "http" ||
			req.HealthCheckResponseText == nil || *req.HealthCheckResponseText != "ok" ||
			req.HealthCheckInterval == nil || *req.HealthCheckInterval != 30 ||
			req.HealthCheckTimeout == nil || *req.HealthCheckTimeout != 10 ||
			req.HealthCheckRetries == nil || *req.HealthCheckRetries != 3 ||
			req.HealthCheckStartPeriod == nil || *req.HealthCheckStartPeriod != 120 ||
			req.HealthCheckType == nil || *req.HealthCheckType != "cmd" ||
			req.HealthCheckCommand == nil || *req.HealthCheckCommand != "wget -q -O- localhost/health" {
			t.Errorf("health check fields must be sent regardless of create=%v, got %+v", create, req)
		}
	}
}

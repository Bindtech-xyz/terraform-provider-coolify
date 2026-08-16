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

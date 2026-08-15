package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestCloudInitScriptToModelEchoesConfigScript locks in a regression found by
// a real apply: cloudInitScriptToModel used to adopt `script` straight from
// the API response, but Coolify normalizes stored content (a trailing
// newline is stripped). `script` is Required, not Computed (Required+Computed
// is not a valid attribute combination), so the planned value is always
// known — any byte-for-byte difference from what Create/Update returns is a
// hard "provider produced inconsistent result" error, on essentially every
// script written with a trailing newline (i.e. nearly all of them, since
// that's how every text editor and HCL heredoc writes files). Config must be
// echoed back on create/read/update; only import (no prior script) adopts
// the API's value.
func TestCloudInitScriptToModelEchoesConfigScript(t *testing.T) {
	prior := cloudInitScriptResourceModel{
		Script: types.StringValue("#cloud-config\npackage_update: true\n"),
	}
	api := &client.CloudInitScript{
		UUID:   "ci1",
		Name:   "sweep",
		Script: "#cloud-config\npackage_update: true", // Coolify strips the trailing newline
	}

	got := cloudInitScriptToModel(api, prior)

	if got.Script.ValueString() != "#cloud-config\npackage_update: true\n" {
		t.Errorf("script = %q, want the configured value unchanged", got.Script.ValueString())
	}
}

// TestCloudInitScriptToModelAdoptsAPIScriptOnImport covers import, where the
// prior model has no script yet — the API's (normalized) value must be
// adopted since there is no config to echo.
func TestCloudInitScriptToModelAdoptsAPIScriptOnImport(t *testing.T) {
	prior := cloudInitScriptResourceModel{} // Script is null: zero value
	api := &client.CloudInitScript{
		UUID:   "ci1",
		Name:   "sweep",
		Script: "#cloud-config\npackage_update: true",
	}

	got := cloudInitScriptToModel(api, prior)

	if got.Script.ValueString() != "#cloud-config\npackage_update: true" {
		t.Errorf("script = %q, want the API's value adopted on import", got.Script.ValueString())
	}
}

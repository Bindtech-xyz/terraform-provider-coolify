package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// TestStorageToModelEchoesConfigNotAPI locks in a regression found by a real
// deployment: storageToModel used to adopt type/name/content directly from
// the API response, which crashed or silently corrupted state because none
// of the three round-trip faithfully —
//
//   - type has no response field at all (Coolify has no discriminator
//     column; the API always decodes to "").
//   - name comes back prefixed with the parent's UUID
//     ("<parent-uuid>-<name>"), not the configured value.
//   - content is unconditionally hidden server-side; the API never returns
//     it, under any token ability.
//
// Only uuid and the new volume_name (the real, prefixed name) may be
// adopted from the API; everything else must be an exact echo of prior
// state/config, by construction immune to "provider produced inconsistent
// result".
func TestStorageToModelEchoesConfigNotAPI(t *testing.T) {
	prior := storageResourceModel{
		Type:      types.StringValue("persistent"),
		Name:      types.StringValue("data"),
		MountPath: types.StringValue("/data"),
		Content:   types.StringNull(),
	}
	api := &client.Storage{
		UUID: "st1",
		Name: "app123-data", // the real, prefixed name Coolify assigns
		// Type and Content are never present in a real response; a zero
		// Go string models that.
	}

	got := storageToModel(api, prior)

	if got.Type.ValueString() != "persistent" {
		t.Errorf("type = %q, want the configured value unchanged", got.Type.ValueString())
	}
	if got.Name.ValueString() != "data" {
		t.Errorf("name = %q, want the configured value unchanged, not the API's prefixed name", got.Name.ValueString())
	}
	if !got.Content.IsNull() {
		t.Errorf("content = %v, want null (never adopted from the API)", got.Content)
	}
	if got.VolumeName.ValueString() != "app123-data" {
		t.Errorf("volume_name = %q, want the API's real prefixed name", got.VolumeName.ValueString())
	}
	if got.UUID.ValueString() != "st1" {
		t.Errorf("uuid = %q, want st1", got.UUID.ValueString())
	}
}

// TestStorageToModelFileMountHasNullVolumeName covers the file-mount case,
// where name (and so volume_name) has no underlying API field at all.
func TestStorageToModelFileMountHasNullVolumeName(t *testing.T) {
	prior := storageResourceModel{
		Type:      types.StringValue("file"),
		MountPath: types.StringValue("/etc/app.conf"),
	}
	api := &client.Storage{UUID: "st2"} // Name is always "" for file mounts

	got := storageToModel(api, prior)
	if !got.VolumeName.IsNull() {
		t.Errorf("volume_name = %v, want null for a file mount", got.VolumeName)
	}
}

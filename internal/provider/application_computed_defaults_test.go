package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestApplicationServerDefaultedAttributesAreComputed locks in a regression
// found by a real deployment (a docker-image-mode application): Coolify
// assigns a non-empty default to these attributes whenever they are left
// unset — base_directory → "/", limits_memory/limits_cpus → "0",
// git_branch → "main", git_commit_sha → "HEAD", build_pack → the effective
// build strategy (e.g. "dockerimage"), static_image → the effective image —
// never "". An Optional-only (non-Computed) string attribute whose final
// state diverges from a planned null value is a framework-level "provider
// produced inconsistent result" error, which aborted every application
// create/update that left any of these unset (i.e. nearly all of them).
// Computed (+ UseStateForUnknown, so the server-assigned value doesn't
// perpetually diff) is required for any attribute where the API's "unset"
// sentinel is not the empty string.
func TestApplicationServerDefaultedAttributesAreComputed(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&applicationResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	for _, name := range []string{
		"base_directory", "limits_memory", "limits_cpus",
		"git_branch", "git_commit_sha", "build_pack", "static_image",
	} {
		attr, ok := schemaResp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("schema is missing %q", name)
		}
		if !attr.IsComputed() {
			t.Errorf("%q must be Computed: Coolify assigns a non-empty default when unset, "+
				"and an Optional-only attribute whose state diverges from a planned null "+
				"value fails the apply with \"provider produced inconsistent result\"", name)
		}
	}
}

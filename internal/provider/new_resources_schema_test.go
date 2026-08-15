package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestDeploymentResourceSchema locks in the Computed/default invariants
// coolify_deployment relies on: force/wait_for_completion/timeout_seconds
// must be Computed (Coolify has sane defaults for none of these — they're
// pure provider-side conveniences), and results must be Computed-only
// (it's the API's response, never user-supplied).
func TestDeploymentResourceSchema(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&deploymentResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	for _, name := range []string{"force", "wait_for_completion", "timeout_seconds"} {
		attr, ok := schemaResp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("schema is missing %q", name)
		}
		if !attr.IsComputed() {
			t.Errorf("%q must be Computed (has a provider-side default)", name)
		}
	}

	results, ok := schemaResp.Schema.Attributes["results"]
	if !ok {
		t.Fatal("schema is missing \"results\"")
	}
	if !results.IsComputed() || results.IsOptional() || results.IsRequired() {
		t.Error("\"results\" must be Computed-only: it is Coolify's own response, never user-supplied")
	}
}

// TestAPISettingsResourceSchema locks in the same Computed invariant for
// coolify_api_settings — both flags need Optional+Computed+Default since
// there is no way to distinguish "user didn't set this" from "user wants
// the current value" without a GET, so a default must always apply.
func TestAPISettingsResourceSchema(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&apiSettingsResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	for _, name := range []string{"api_enabled", "mcp_enabled"} {
		attr, ok := schemaResp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("schema is missing %q", name)
		}
		if !attr.IsComputed() || !attr.IsOptional() {
			t.Errorf("%q must be Optional+Computed", name)
		}
	}
}

// TestResourceTagSchemaTagNameNotComputed documents, via the schema itself,
// the deliberate choice behind resource_tag_mapping_test.go's
// TestFindTagIsCaseInsensitive: tag_name stays plain Required (never
// Computed) because the resource always echoes the configured value back
// into state instead of adopting Coolify's lowercased one.
func TestResourceTagSchemaTagNameNotComputed(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&resourceTagResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	attr, ok := schemaResp.Schema.Attributes["tag_name"]
	if !ok {
		t.Fatal("schema is missing \"tag_name\"")
	}
	if attr.IsComputed() {
		t.Error("\"tag_name\" must not be Computed: state always echoes config, never Coolify's normalized value")
	}
	if !attr.IsRequired() {
		t.Error("\"tag_name\" must be Required")
	}
}

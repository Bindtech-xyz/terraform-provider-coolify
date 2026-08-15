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

// TestDatabaseLimitsAreComputed is coolify_database's twin of the
// application check above: limits_memory/limits_cpus default to "0"
// (unlimited) whenever unset, never "" — the same class of bug, just never
// surfaced as a crash here because the client didn't even read the fields
// back from the API (so state simply kept the planned null, no divergence).
// That silent gap — a configured limit applied once at create and never
// verified again — is the actual defect; Computed is what makes drift on
// these two attributes detectable at all.
func TestDatabaseLimitsAreComputed(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&databaseResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	for _, name := range []string{"limits_memory", "limits_cpus"} {
		attr, ok := schemaResp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("schema is missing %q", name)
		}
		if !attr.IsComputed() {
			t.Errorf("%q must be Computed: Coolify defaults it to \"0\" when unset, "+
				"never \"\" — without Computed, a configured value is applied once and "+
				"never read back, so drift on it goes undetected", name)
		}
	}
}

// TestDatabaseBackupDatabasesToBackupIsComputed locks in a regression found
// by a real apply: databases_to_backup was Optional-only, but Coolify
// defaults it to the engine's own logical database name (e.g. "postgres")
// whenever left unset — the same "provider produced inconsistent result"
// class as the two checks above, just on coolify_database_backup instead.
func TestDatabaseBackupDatabasesToBackupIsComputed(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&databaseBackupResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	attr, ok := schemaResp.Schema.Attributes["databases_to_backup"]
	if !ok {
		t.Fatal("schema is missing \"databases_to_backup\"")
	}
	if !attr.IsComputed() {
		t.Error("\"databases_to_backup\" must be Computed: Coolify defaults it to the engine's " +
			"logical database name when unset, never \"\"")
	}
}

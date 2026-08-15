package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// newDatabaseImportStateResponse builds an ImportStateResponse with a State
// wired to the real schema — SetAttribute panics without one, since it needs
// the schema to resolve the attribute's type. The framework always provides
// this in production; a bare resource.ImportStateResponse{} (no Schema) does
// not, and is only valid for resources whose ImportState never calls
// SetAttribute (e.g. plain ImportStatePassthroughID).
func newDatabaseImportStateResponse(t *testing.T) *resource.ImportStateResponse {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&databaseResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	tfType := schemaResp.Schema.Type().TerraformType(ctx)
	return &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(tfType, nil),
		},
	}
}

// TestDatabaseImportStateRejectsBadIdentifiers locks in the "<engine>/<uuid>"
// contract: engine is Required+RequiresReplace, so ImportState must populate
// it explicitly or the first plan after import proposes a destroy/recreate.
func TestDatabaseImportStateRejectsBadIdentifiers(t *testing.T) {
	r := &databaseResource{}
	ctx := context.Background()

	cases := []struct {
		name string
		id   string
	}{
		{"missing slash", "abc123"},
		{"empty engine", "/abc123"},
		{"empty uuid", "postgresql/"},
		{"unknown engine", "oracle/abc123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ImportStateResponse{}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.id}, resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("ImportState(%q): expected error, got none", tc.id)
			}
		})
	}
}

func TestDatabaseImportStateAcceptsEveryEngine(t *testing.T) {
	r := &databaseResource{}
	ctx := context.Background()

	for _, engine := range []string{"postgresql", "mysql", "mariadb", "mongodb", "redis", "keydb", "dragonfly", "clickhouse"} {
		resp := newDatabaseImportStateResponse(t)
		r.ImportState(ctx, resource.ImportStateRequest{ID: engine + "/db1"}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ImportState(%s/db1): unexpected error: %v", engine, resp.Diagnostics)
		}
		var got databaseResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
		if got.Engine.ValueString() != engine {
			t.Errorf("engine = %q, want %q", got.Engine.ValueString(), engine)
		}
		if got.UUID.ValueString() != "db1" {
			t.Errorf("uuid = %q, want db1", got.UUID.ValueString())
		}
	}
}

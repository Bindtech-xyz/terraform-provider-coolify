package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestDataSourcesReturnEmptyListNotNull locks in a regression found by a real
// apply: several list data sources initialized their list field as a bare Go
// zero-value struct, so an empty API result produced a nil Go slice — which
// the framework marshals as a Terraform null, not an empty list. Terraform
// Core's `length()` (and anything else expecting a collection, e.g.
// `for_each`) rejects null outright, breaking any consuming configuration the
// moment the underlying Coolify team happens to have zero of that object —
// not a hypothetical, it is what a disposable or newly bootstrapped instance
// looks like. Every data source below is exercised against a real empty API
// response; the fix is to always `make([]T, 0, ...)` regardless of count.
func TestDataSourcesReturnEmptyListNotNull(t *testing.T) {
	emptyArray := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(emptyArray.Close)
	c, err := client.New(emptyArray.URL, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	ctx := context.Background()

	t.Run("tags", func(t *testing.T) {
		d := &tagsDataSource{client: c}
		resp := readEmpty(t, ctx, d)
		var state tagsDataSourceModel
		resp.State.Get(ctx, &state)
		if state.Tags == nil {
			t.Error("Tags is nil (null), want an empty non-nil slice")
		}
	})

	t.Run("s3_storages", func(t *testing.T) {
		d := &s3StoragesDataSource{client: c}
		resp := readEmpty(t, ctx, d)
		var state s3StoragesDataSourceModel
		resp.State.Get(ctx, &state)
		if state.S3Storages == nil {
			t.Error("S3Storages is nil (null), want an empty non-nil slice")
		}
	})

	t.Run("teams", func(t *testing.T) {
		d := &teamsDataSource{client: c}
		resp := readEmpty(t, ctx, d)
		var state teamsDataSourceModel
		resp.State.Get(ctx, &state)
		if state.Teams == nil {
			t.Error("Teams is nil (null), want an empty non-nil slice")
		}
	})
}

// readEmpty invokes Read on a schema-having data source with a Config built
// from its own zero-value schema (i.e. no required attributes to fill —
// every data source exercised here takes no config), and returns the response.
func readEmpty(t *testing.T, ctx context.Context, d interface {
	datasource.DataSource
	datasource.DataSourceWithConfigure
}) *datasource.ReadResponse {
	t.Helper()
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}

	tfType := schemaResp.Schema.Type().TerraformType(ctx)
	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(tfType, nil),
		},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tfType, nil)},
	}
	d.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	return resp
}

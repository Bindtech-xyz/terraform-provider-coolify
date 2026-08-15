package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestAllResourceSchemasValidate instantiates every resource, requests its
// schema and metadata, and fails on any diagnostic or duplicate type name.
func TestAllResourceSchemasValidate(t *testing.T) {
	ctx := context.Background()
	seen := map[string]bool{}

	for _, newResource := range New("test")().Resources(ctx) {
		res := newResource()

		var metaResp resource.MetadataResponse
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "coolify"}, &metaResp)
		if metaResp.TypeName == "" {
			t.Fatalf("%T: empty type name", res)
		}
		if seen[metaResp.TypeName] {
			t.Errorf("duplicate resource type %q", metaResp.TypeName)
		}
		seen[metaResp.TypeName] = true

		var schemaResp resource.SchemaResponse
		res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema diagnostics: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if len(schemaResp.Schema.Attributes) == 0 {
			t.Errorf("%s: schema has no attributes", metaResp.TypeName)
		}
	}

	if len(seen) < 25 {
		t.Errorf("expected at least 25 resources, got %d", len(seen))
	}
}

// TestAllDataSourceSchemasValidate does the same for data sources.
func TestAllDataSourceSchemasValidate(t *testing.T) {
	ctx := context.Background()
	seen := map[string]bool{}

	for _, newDataSource := range New("test")().DataSources(ctx) {
		ds := newDataSource()

		var metaResp datasource.MetadataResponse
		ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "coolify"}, &metaResp)
		if metaResp.TypeName == "" {
			t.Fatalf("%T: empty type name", ds)
		}
		if seen[metaResp.TypeName] {
			t.Errorf("duplicate data source type %q", metaResp.TypeName)
		}
		seen[metaResp.TypeName] = true

		var schemaResp datasource.SchemaResponse
		ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema diagnostics: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
	}

	if len(seen) < 20 {
		t.Errorf("expected at least 20 data sources, got %d", len(seen))
	}
}

// TestProviderSchemaValidates covers the provider-level schema itself.
func TestProviderSchemaValidates(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	var metaResp provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &metaResp)
	if metaResp.TypeName != "coolify" {
		t.Errorf("provider type name = %q, want coolify", metaResp.TypeName)
	}

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Errorf("provider schema diagnostics: %v", schemaResp.Diagnostics)
	}
	for _, attr := range []string{"endpoint", "token", "insecure"} {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("provider schema missing %q", attr)
		}
	}
}

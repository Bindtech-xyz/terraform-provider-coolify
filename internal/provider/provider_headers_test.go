package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tfConfigFromModel builds a tfsdk.Config for a provider schema from a Go
// model. tfsdk.Config has no Set method (only State and Plan do, since Config
// is practitioner-authored and read-only to the framework) — so this
// marshals through a throwaway State sharing the same schema/type, then
// rewraps its Raw value into a Config. Test-only; production code never
// needs this.
func tfConfigFromModel(t *testing.T, ctx context.Context, sch pschema.Schema, model any) tfsdk.Config {
	t.Helper()
	state := tfsdk.State{Schema: sch}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("building test config: %v", diags)
	}
	return tfsdk.Config{Schema: sch, Raw: state.Raw}
}

// TestProviderSchemaHasHeadersAttribute locks in the generic escape hatch for
// anything gating on HTTP headers (Cloudflare Access, oauth2-proxy, a
// header-based mTLS gateway, ...). Deliberately not named after any specific
// one of these — the provider has no built-in notion of Cloudflare.
func TestProviderSchemaHasHeadersAttribute(t *testing.T) {
	ctx := context.Background()
	var schemaResp provider.SchemaResponse
	New("test")().Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	attr, ok := schemaResp.Schema.Attributes["headers"]
	if !ok {
		t.Fatal(`provider schema is missing "headers"`)
	}
	if !attr.IsSensitive() {
		t.Error(`"headers" must be Sensitive — header values are commonly secrets (service token, API key)`)
	}
	if attr.IsRequired() {
		t.Error(`"headers" must be Optional — most Coolify instances sit behind nothing`)
	}
}

// TestUnknownHeadersProducesActionableError covers the exact scenario a
// Cloudflare Access service token creates: headers built from a resource's
// attributes are Unknown until that resource is applied, and a provider
// block can never depend on a resource created in the same apply. Configure
// must fail with a diagnostic that explains why and how to fix it, not
// surface a confusing connectivity error deep inside client.New.
func TestUnknownHeadersProducesActionableError(t *testing.T) {
	ctx := context.Background()
	p := &coolifyProvider{version: "test"}

	config := coolifyProviderModel{
		Endpoint: types.StringValue("https://coolify.example.com"),
		Token:    types.StringValue("tok"),
		Headers:  types.MapUnknown(types.StringType),
	}

	schemaResp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

	rawState := tfConfigFromModel(t, ctx, schemaResp.Schema, config)
	resp := &provider.ConfigureResponse{}
	p.Configure(ctx, provider.ConfigureRequest{Config: rawState}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Configure with unknown headers: expected an error, got none")
	}
	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Unknown Coolify provider headers" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the \"Unknown Coolify provider headers\" diagnostic, got: %v", resp.Diagnostics)
	}
}

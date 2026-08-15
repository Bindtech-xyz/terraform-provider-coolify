package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// mapRequiresReplace forces replacement when a map attribute changes.
func mapRequiresReplace() planmodifier.Map {
	return mapplanmodifier.RequiresReplace()
}

// pathRoot is a tiny alias so provider.go reads without importing path directly.
func pathRoot(name string) path.Path { return path.Root(name) }

// resourceClient extracts the *client.Client stored by the provider's Configure.
// It returns nil (with a diagnostic already added) when ProviderData is not yet
// available — the framework calls Configure before the provider is configured.
func resourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource Configure type",
			"Expected *client.Client. This is a bug in the provider; please report it.",
		)
		return nil
	}
	return c
}

// dataSourceClient is the data-source twin of resourceClient.
func dataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source Configure type",
			"Expected *client.Client. This is a bug in the provider; please report it.",
		)
		return nil
	}
	return c
}

// stringOrNil converts a framework string to the *string the API client expects:
// nil when null/unknown so the field is omitted from the JSON body.
func stringOrNil(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// int64OrNil is the int64 twin of stringOrNil.
func int64OrNil(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// keepNullIfEmpty adopts the API value but keeps the attribute null when the
// API returns "" and the practitioner never configured it — Coolify normalises
// absent strings to "", which would otherwise cause permanent diffs.
func keepNullIfEmpty(apiValue string, prior types.String) types.String {
	if apiValue == "" && prior.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(apiValue)
}

// keepPriorIfHidden adopts the API value unless the API hid it (empty string)
// while the prior state holds a value — used for sensitive fields the API only
// echoes to tokens with the read:sensitive ability. An Optional+Computed
// attribute that is unknown in the plan resolves to empty when the API hides it
// and nothing was configured.
func keepPriorIfHidden(apiValue string, prior types.String) types.String {
	if apiValue == "" && !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return types.StringValue(apiValue)
}

// boolOrNil is the bool twin of stringOrNil.
func boolOrNil(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

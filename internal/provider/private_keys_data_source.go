package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*privateKeysDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*privateKeysDataSource)(nil)
)

// NewPrivateKeysDataSource is registered in provider.go.
func NewPrivateKeysDataSource() datasource.DataSource {
	return &privateKeysDataSource{}
}

type privateKeysDataSource struct {
	client *client.Client
}

type privateKeyListEntryModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

type privateKeysDataSourceModel struct {
	PrivateKeys []privateKeyListEntryModel `tfsdk:"private_keys"`
}

func (d *privateKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_keys"
}

func (d *privateKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the private keys of the token's team (metadata only, no key material).",
		Attributes: map[string]schema.Attribute{
			"private_keys": schema.ListNestedAttribute{
				MarkdownDescription: "All private keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":        schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Description."},
						"fingerprint": schema.StringAttribute{Computed: true, MarkdownDescription: "SSH fingerprint."},
					},
				},
			},
		},
	}
}

func (d *privateKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *privateKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListPrivateKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify private keys", err.Error())
		return
	}

	state := privateKeysDataSourceModel{PrivateKeys: make([]privateKeyListEntryModel, 0, len(keys))}
	for _, k := range keys {
		state.PrivateKeys = append(state.PrivateKeys, privateKeyListEntryModel{
			UUID:        types.StringValue(k.UUID),
			Name:        types.StringValue(k.Name),
			Description: types.StringValue(k.Description),
			Fingerprint: types.StringValue(k.Fingerprint),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

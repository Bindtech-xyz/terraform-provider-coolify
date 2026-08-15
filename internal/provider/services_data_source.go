package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*servicesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*servicesDataSource)(nil)
)

// NewServicesDataSource is registered in provider.go.
func NewServicesDataSource() datasource.DataSource {
	return &servicesDataSource{}
}

type servicesDataSource struct {
	client *client.Client
}

type serviceListEntryModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Status types.String `tfsdk:"status"`
}

type servicesDataSourceModel struct {
	Services []serviceListEntryModel `tfsdk:"services"`
}

func (d *servicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *servicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every service of the token's team.",
		Attributes: map[string]schema.Attribute{
			"services": schema.ListNestedAttribute{
				MarkdownDescription: "All services.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":   schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"type":   schema.StringAttribute{Computed: true, MarkdownDescription: "One-click service type (empty for compose services)."},
						"status": schema.StringAttribute{Computed: true, MarkdownDescription: "Runtime status."},
					},
				},
			},
		},
	}
}

func (d *servicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *servicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	services, err := d.client.ListServices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify services", err.Error())
		return
	}

	state := servicesDataSourceModel{Services: make([]serviceListEntryModel, 0, len(services))}
	for _, s := range services {
		state.Services = append(state.Services, serviceListEntryModel{
			UUID:   types.StringValue(s.UUID),
			Name:   types.StringValue(s.Name),
			Type:   types.StringValue(s.Type),
			Status: types.StringValue(s.Status),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

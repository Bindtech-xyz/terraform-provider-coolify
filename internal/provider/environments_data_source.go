package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*environmentsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*environmentsDataSource)(nil)
)

// NewEnvironmentsDataSource is registered in provider.go.
func NewEnvironmentsDataSource() datasource.DataSource {
	return &environmentsDataSource{}
}

type environmentsDataSource struct {
	client *client.Client
}

type environmentListEntryModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type environmentsDataSourceModel struct {
	ProjectUUID  types.String                `tfsdk:"project_uuid"`
	Environments []environmentListEntryModel `tfsdk:"environments"`
}

func (d *environmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environments"
}

func (d *environmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the environments of a project.",
		Attributes: map[string]schema.Attribute{
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the project to inspect.",
				Required:            true,
			},
			"environments": schema.ListNestedAttribute{
				MarkdownDescription: "Environments of the project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":        schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Description."},
					},
				},
			},
		},
	}
}

func (d *environmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *environmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config environmentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envs, err := d.client.ListEnvironments(ctx, config.ProjectUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to list environments of project %s", config.ProjectUUID.ValueString()),
			err.Error(),
		)
		return
	}

	state := environmentsDataSourceModel{
		ProjectUUID:  config.ProjectUUID,
		Environments: make([]environmentListEntryModel, 0, len(envs)),
	}
	for _, e := range envs {
		state.Environments = append(state.Environments, environmentListEntryModel{
			UUID:        types.StringValue(e.UUID),
			Name:        types.StringValue(e.Name),
			Description: types.StringValue(e.Description),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

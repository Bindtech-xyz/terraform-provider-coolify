package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*projectsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*projectsDataSource)(nil)
)

// NewProjectsDataSource is registered in provider.go.
func NewProjectsDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

type projectsDataSource struct {
	client *client.Client
}

type projectsDataSourceModel struct {
	Projects []projectDataSourceModel `tfsdk:"projects"`
}

func (d *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every Coolify project visible to the configured API token.",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				MarkdownDescription: "All projects on the instance.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "UUID of the project.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the project.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of the project.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *projectsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	projects, err := d.client.ListProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify projects", err.Error())
		return
	}

	state := projectsDataSourceModel{
		Projects: make([]projectDataSourceModel, 0, len(projects)),
	}
	for _, p := range projects {
		state.Projects = append(state.Projects, projectDataSourceModel{
			UUID:        types.StringValue(p.UUID),
			Name:        types.StringValue(p.Name),
			Description: types.StringValue(p.Description),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

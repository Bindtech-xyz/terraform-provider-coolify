package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*applicationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*applicationsDataSource)(nil)
)

// NewApplicationsDataSource is registered in provider.go.
func NewApplicationsDataSource() datasource.DataSource {
	return &applicationsDataSource{}
}

type applicationsDataSource struct {
	client *client.Client
}

type applicationListEntryModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	FQDN      types.String `tfsdk:"fqdn"`
	GitRepo   types.String `tfsdk:"git_repository"`
	GitBranch types.String `tfsdk:"git_branch"`
	BuildPack types.String `tfsdk:"build_pack"`
	Status    types.String `tfsdk:"status"`
}

type applicationsDataSourceModel struct {
	Applications []applicationListEntryModel `tfsdk:"applications"`
}

func (d *applicationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

func (d *applicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every application of the token's team.",
		Attributes: map[string]schema.Attribute{
			"applications": schema.ListNestedAttribute{
				MarkdownDescription: "All applications.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":           schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"fqdn":           schema.StringAttribute{Computed: true, MarkdownDescription: "Domain(s)."},
						"git_repository": schema.StringAttribute{Computed: true, MarkdownDescription: "Git repository."},
						"git_branch":     schema.StringAttribute{Computed: true, MarkdownDescription: "Git branch."},
						"build_pack":     schema.StringAttribute{Computed: true, MarkdownDescription: "Build pack."},
						"status":         schema.StringAttribute{Computed: true, MarkdownDescription: "Runtime status."},
					},
				},
			},
		},
	}
}

func (d *applicationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *applicationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apps, err := d.client.ListApplications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify applications", err.Error())
		return
	}

	state := applicationsDataSourceModel{Applications: make([]applicationListEntryModel, 0, len(apps))}
	for _, a := range apps {
		state.Applications = append(state.Applications, applicationListEntryModel{
			UUID:      types.StringValue(a.UUID),
			Name:      types.StringValue(a.Name),
			FQDN:      types.StringValue(a.FQDN),
			GitRepo:   types.StringValue(a.GitRepository),
			GitBranch: types.StringValue(a.GitBranch),
			BuildPack: types.StringValue(a.BuildPack),
			Status:    types.StringValue(a.Status),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

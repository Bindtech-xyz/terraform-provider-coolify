package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*deploymentsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*deploymentsDataSource)(nil)
)

// NewDeploymentsDataSource is registered in provider.go.
func NewDeploymentsDataSource() datasource.DataSource {
	return &deploymentsDataSource{}
}

type deploymentsDataSource struct {
	client *client.Client
}

type deploymentEntryModel struct {
	DeploymentUUID  types.String `tfsdk:"deployment_uuid"`
	ApplicationName types.String `tfsdk:"application_name"`
	Status          types.String `tfsdk:"status"`
	Commit          types.String `tfsdk:"commit"`
}

type deploymentsDataSourceModel struct {
	ApplicationUUID types.String           `tfsdk:"application_uuid"`
	Deployments     []deploymentEntryModel `tfsdk:"deployments"`
}

func (d *deploymentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployments"
}

func (d *deploymentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deployments: without `application_uuid`, the currently " +
			"queued/running deployments of the instance; with it, the deployment history of " +
			"that application.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "Restrict to one application's deployment history.",
				Optional:            true,
			},
			"deployments": schema.ListNestedAttribute{
				MarkdownDescription: "Deployments, newest first.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"deployment_uuid":  schema.StringAttribute{Computed: true, MarkdownDescription: "Deployment UUID."},
						"application_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Application name."},
						"status":           schema.StringAttribute{Computed: true, MarkdownDescription: "queued, in_progress, finished, failed…"},
						"commit":           schema.StringAttribute{Computed: true, MarkdownDescription: "Deployed commit SHA."},
					},
				},
			},
		},
	}
}

func (d *deploymentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *deploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config deploymentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		deployments []client.Deployment
		err         error
	)
	if config.ApplicationUUID.IsNull() {
		deployments, err = d.client.ListDeployments(ctx)
	} else {
		deployments, err = d.client.ListApplicationDeployments(ctx, config.ApplicationUUID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify deployments", err.Error())
		return
	}

	state := deploymentsDataSourceModel{
		ApplicationUUID: config.ApplicationUUID,
		Deployments:     make([]deploymentEntryModel, 0, len(deployments)),
	}
	for _, dep := range deployments {
		state.Deployments = append(state.Deployments, deploymentEntryModel{
			DeploymentUUID:  types.StringValue(dep.DeploymentUUID),
			ApplicationName: types.StringValue(dep.ApplicationName),
			Status:          types.StringValue(dep.Status),
			Commit:          types.StringValue(dep.Commit),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*teamDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*teamDataSource)(nil)
)

// NewTeamDataSource is registered in provider.go.
func NewTeamDataSource() datasource.DataSource {
	return &teamDataSource{}
}

type teamDataSource struct {
	client *client.Client
}

type teamMemberModel struct {
	ID    types.Int64  `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Email types.String `tfsdk:"email"`
}

type teamDataSourceModel struct {
	ID          types.Int64       `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	Description types.String      `tfsdk:"description"`
	Members     []teamMemberModel `tfsdk:"members"`
}

func (d *teamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *teamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The team the configured API token belongs to, with its members.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric team id."},
			"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Team name."},
			"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Team description."},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "Team members.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.Int64Attribute{Computed: true, MarkdownDescription: "User id."},
						"name":  schema.StringAttribute{Computed: true, MarkdownDescription: "User name."},
						"email": schema.StringAttribute{Computed: true, MarkdownDescription: "User email."},
					},
				},
			},
		},
	}
}

func (d *teamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *teamDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	team, err := d.client.CurrentTeam(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the token's Coolify team", err.Error())
		return
	}
	members, err := d.client.CurrentTeamMembers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list the token's Coolify team members", err.Error())
		return
	}

	state := teamDataSourceModel{
		ID:          types.Int64Value(team.ID),
		Name:        types.StringValue(team.Name),
		Description: types.StringValue(team.Description),
		Members:     make([]teamMemberModel, 0, len(members)),
	}
	for _, m := range members {
		state.Members = append(state.Members, teamMemberModel{
			ID:    types.Int64Value(m.ID),
			Name:  types.StringValue(m.Name),
			Email: types.StringValue(m.Email),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

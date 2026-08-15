package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*databasesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*databasesDataSource)(nil)
)

// NewDatabasesDataSource is registered in provider.go.
func NewDatabasesDataSource() datasource.DataSource {
	return &databasesDataSource{}
}

type databasesDataSource struct {
	client *client.Client
}

type databaseListEntryModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	Image         types.String `tfsdk:"image"`
	IsPublic      types.Bool   `tfsdk:"is_public"`
	Status        types.String `tfsdk:"status"`
	InternalDBURL types.String `tfsdk:"internal_db_url"`
}

type databasesDataSourceModel struct {
	Databases []databaseListEntryModel `tfsdk:"databases"`
}

func (d *databasesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_databases"
}

func (d *databasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every standalone database of the token's team.",
		Attributes: map[string]schema.Attribute{
			"databases": schema.ListNestedAttribute{
				MarkdownDescription: "All databases.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":            schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":            schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"image":           schema.StringAttribute{Computed: true, MarkdownDescription: "Docker image."},
						"is_public":       schema.BoolAttribute{Computed: true, MarkdownDescription: "Publicly exposed."},
						"status":          schema.StringAttribute{Computed: true, MarkdownDescription: "Runtime status."},
						"internal_db_url": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Internal connection URL."},
					},
				},
			},
		},
	}
}

func (d *databasesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *databasesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	dbs, err := d.client.ListDatabases(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify databases", err.Error())
		return
	}

	state := databasesDataSourceModel{Databases: make([]databaseListEntryModel, 0, len(dbs))}
	for _, db := range dbs {
		state.Databases = append(state.Databases, databaseListEntryModel{
			UUID:          types.StringValue(db.UUID),
			Name:          types.StringValue(db.Name),
			Image:         types.StringValue(db.Image),
			IsPublic:      types.BoolValue(db.IsPublic),
			Status:        types.StringValue(db.Status),
			InternalDBURL: types.StringValue(db.InternalDBURL),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

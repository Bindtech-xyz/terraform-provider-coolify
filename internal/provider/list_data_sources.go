package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// Compact list/single data sources for referencing objects that are not
// managed by this configuration.

// ---- coolify_server (single) ----

var (
	_ datasource.DataSource              = (*serverDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serverDataSource)(nil)
)

// NewServerDataSource is registered in provider.go.
func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

type serverDataSource struct {
	client *client.Client
}

type serverDataSourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IP          types.String `tfsdk:"ip"`
	Port        types.Int64  `tfsdk:"port"`
	User        types.String `tfsdk:"user"`
	ProxyType   types.String `tfsdk:"proxy_type"`
	IsReachable types.Bool   `tfsdk:"is_reachable"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single Coolify server by UUID.",
		Attributes: map[string]schema.Attribute{
			"uuid":         schema.StringAttribute{Required: true, MarkdownDescription: "UUID of the server."},
			"name":         schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
			"description":  schema.StringAttribute{Computed: true, MarkdownDescription: "Description."},
			"ip":           schema.StringAttribute{Computed: true, MarkdownDescription: "IP address or hostname."},
			"port":         schema.Int64Attribute{Computed: true, MarkdownDescription: "SSH port."},
			"user":         schema.StringAttribute{Computed: true, MarkdownDescription: "SSH user."},
			"proxy_type":   schema.StringAttribute{Computed: true, MarkdownDescription: "Reverse proxy type."},
			"is_reachable": schema.BoolAttribute{Computed: true, MarkdownDescription: "Reachability on last check."},
			"is_usable":    schema.BoolAttribute{Computed: true, MarkdownDescription: "Fully validated and usable."},
		},
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := d.client.GetServer(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify server %s", config.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	state := serverDataSourceModel{
		UUID:        types.StringValue(server.UUID),
		Name:        types.StringValue(server.Name),
		Description: types.StringValue(server.Description),
		IP:          types.StringValue(server.IP),
		Port:        types.Int64Value(server.Port),
		User:        types.StringValue(server.User),
		ProxyType:   types.StringValue(server.ProxyType),
		IsReachable: types.BoolValue(false),
		IsUsable:    types.BoolValue(false),
	}
	if server.Settings != nil {
		state.IsReachable = types.BoolValue(server.Settings.IsReachable)
		state.IsUsable = types.BoolValue(server.Settings.IsUsable)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_tags ----

var (
	_ datasource.DataSource              = (*tagsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*tagsDataSource)(nil)
)

// NewTagsDataSource is registered in provider.go.
func NewTagsDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

type tagsDataSource struct {
	client *client.Client
}

type tagEntryModel struct {
	UUID types.String `tfsdk:"uuid"`
	Name types.String `tfsdk:"name"`
}

type tagsDataSourceModel struct {
	Tags []tagEntryModel `tfsdk:"tags"`
}

func (d *tagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every tag of the token's team.",
		Attributes: map[string]schema.Attribute{
			"tags": schema.ListNestedAttribute{
				MarkdownDescription: "All tags.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
					},
				},
			},
		},
	}
}

func (d *tagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *tagsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tags, err := d.client.ListTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify tags", err.Error())
		return
	}
	state := tagsDataSourceModel{Tags: make([]tagEntryModel, 0, len(tags))}
	for _, t := range tags {
		state.Tags = append(state.Tags, tagEntryModel{
			UUID: types.StringValue(t.UUID),
			Name: types.StringValue(t.Name),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_destinations ----

var (
	_ datasource.DataSource              = (*destinationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*destinationsDataSource)(nil)
)

// NewDestinationsDataSource is registered in provider.go.
func NewDestinationsDataSource() datasource.DataSource {
	return &destinationsDataSource{}
}

type destinationsDataSource struct {
	client *client.Client
}

type destinationEntryModel struct {
	UUID    types.String `tfsdk:"uuid"`
	Name    types.String `tfsdk:"name"`
	Network types.String `tfsdk:"network"`
}

type destinationsDataSourceModel struct {
	ServerUUID   types.String            `tfsdk:"server_uuid"`
	Destinations []destinationEntryModel `tfsdk:"destinations"`
}

func (d *destinationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destinations"
}

func (d *destinationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists destinations (Docker networks), optionally restricted to one server.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "Restrict to one server's destinations.",
				Optional:            true,
			},
			"destinations": schema.ListNestedAttribute{
				MarkdownDescription: "Destinations.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":    schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":    schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"network": schema.StringAttribute{Computed: true, MarkdownDescription: "Docker network."},
					},
				},
			},
		},
	}
}

func (d *destinationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *destinationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config destinationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		destinations []client.Destination
		err          error
	)
	if config.ServerUUID.IsNull() {
		destinations, err = d.client.ListDestinations(ctx)
	} else {
		destinations, err = d.client.ListServerDestinations(ctx, config.ServerUUID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify destinations", err.Error())
		return
	}

	state := destinationsDataSourceModel{
		ServerUUID:   config.ServerUUID,
		Destinations: make([]destinationEntryModel, 0, len(destinations)),
	}
	for _, dest := range destinations {
		state.Destinations = append(state.Destinations, destinationEntryModel{
			UUID:    types.StringValue(dest.UUID),
			Name:    types.StringValue(dest.Name),
			Network: types.StringValue(dest.Network),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_s3_storages ----

var (
	_ datasource.DataSource              = (*s3StoragesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*s3StoragesDataSource)(nil)
)

// NewS3StoragesDataSource is registered in provider.go.
func NewS3StoragesDataSource() datasource.DataSource {
	return &s3StoragesDataSource{}
}

type s3StoragesDataSource struct {
	client *client.Client
}

type s3StorageEntryModel struct {
	UUID     types.String `tfsdk:"uuid"`
	Name     types.String `tfsdk:"name"`
	Endpoint types.String `tfsdk:"endpoint"`
	Bucket   types.String `tfsdk:"bucket"`
	IsUsable types.Bool   `tfsdk:"is_usable"`
}

type s3StoragesDataSourceModel struct {
	S3Storages []s3StorageEntryModel `tfsdk:"s3_storages"`
}

func (d *s3StoragesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storages"
}

func (d *s3StoragesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every S3 storage of the token's team (no credentials).",
		Attributes: map[string]schema.Attribute{
			"s3_storages": schema.ListNestedAttribute{
				MarkdownDescription: "All S3 storages.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":      schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":      schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"endpoint":  schema.StringAttribute{Computed: true, MarkdownDescription: "S3 endpoint."},
						"bucket":    schema.StringAttribute{Computed: true, MarkdownDescription: "Bucket."},
						"is_usable": schema.BoolAttribute{Computed: true, MarkdownDescription: "Connectivity validated."},
					},
				},
			},
		},
	}
}

func (d *s3StoragesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *s3StoragesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	storages, err := d.client.ListS3Storages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify S3 storages", err.Error())
		return
	}
	state := s3StoragesDataSourceModel{S3Storages: make([]s3StorageEntryModel, 0, len(storages))}
	for _, s := range storages {
		state.S3Storages = append(state.S3Storages, s3StorageEntryModel{
			UUID:     types.StringValue(s.UUID),
			Name:     types.StringValue(s.Name),
			Endpoint: types.StringValue(s.Endpoint),
			Bucket:   types.StringValue(s.Bucket),
			IsUsable: types.BoolValue(s.IsUsable),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_teams ----

var (
	_ datasource.DataSource              = (*teamsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*teamsDataSource)(nil)
)

// NewTeamsDataSource is registered in provider.go.
func NewTeamsDataSource() datasource.DataSource {
	return &teamsDataSource{}
}

type teamsDataSource struct {
	client *client.Client
}

type teamEntryModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type teamsDataSourceModel struct {
	Teams []teamEntryModel `tfsdk:"teams"`
}

func (d *teamsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teams"
}

func (d *teamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every team the API token's user belongs to.",
		Attributes: map[string]schema.Attribute{
			"teams": schema.ListNestedAttribute{
				MarkdownDescription: "Teams.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric id."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Description."},
					},
				},
			},
		},
	}
}

func (d *teamsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *teamsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	teams, err := d.client.ListTeams(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify teams", err.Error())
		return
	}
	state := teamsDataSourceModel{Teams: make([]teamEntryModel, 0, len(teams))}
	for _, t := range teams {
		state.Teams = append(state.Teams, teamEntryModel{
			ID:          types.Int64Value(t.ID),
			Name:        types.StringValue(t.Name),
			Description: types.StringValue(t.Description),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// ---- coolify_server_domains ----

var (
	_ datasource.DataSource              = (*serverDomainsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serverDomainsDataSource)(nil)
)

// NewServerDomainsDataSource is registered in provider.go.
func NewServerDomainsDataSource() datasource.DataSource {
	return &serverDomainsDataSource{}
}

type serverDomainsDataSource struct {
	client *client.Client
}

type serverDomainEntryModel struct {
	IP      types.String   `tfsdk:"ip"`
	Domains []types.String `tfsdk:"domains"`
}

type serverDomainsDataSourceModel struct {
	ServerUUID types.String             `tfsdk:"server_uuid"`
	Domains    []serverDomainEntryModel `tfsdk:"domains"`
}

func (d *serverDomainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_domains"
}

func (d *serverDomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The domains served by a server, grouped by IP — useful for DNS automation.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server.",
				Required:            true,
			},
			"domains": schema.ListNestedAttribute{
				MarkdownDescription: "Domains grouped by IP.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{Computed: true, MarkdownDescription: "IP address."},
						"domains": schema.ListAttribute{
							Computed: true, ElementType: types.StringType,
							MarkdownDescription: "Domains served from this IP.",
						},
					},
				},
			},
		},
	}
}

func (d *serverDomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *serverDomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverDomainsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domains, err := d.client.GetServerDomains(ctx, config.ServerUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read domains of server %s", config.ServerUUID.ValueString()),
			err.Error(),
		)
		return
	}

	state := serverDomainsDataSourceModel{
		ServerUUID: config.ServerUUID,
		Domains:    make([]serverDomainEntryModel, 0, len(domains)),
	}
	for _, entry := range domains {
		row := serverDomainEntryModel{
			IP:      types.StringValue(entry.IP),
			Domains: make([]types.String, 0, len(entry.Domains)),
		}
		for _, domain := range entry.Domains {
			row.Domains = append(row.Domains, types.StringValue(domain))
		}
		state.Domains = append(state.Domains, row)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_server_resources ----

var (
	_ datasource.DataSource              = (*serverResourcesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serverResourcesDataSource)(nil)
)

// NewServerResourcesDataSource is registered in provider.go.
func NewServerResourcesDataSource() datasource.DataSource {
	return &serverResourcesDataSource{}
}

type serverResourcesDataSource struct {
	client *client.Client
}

type serverResourceEntryModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Status types.String `tfsdk:"status"`
}

type serverResourcesDataSourceModel struct {
	ServerUUID types.String               `tfsdk:"server_uuid"`
	Resources  []serverResourceEntryModel `tfsdk:"resources"`
}

func (d *serverResourcesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_resources"
}

func (d *serverResourcesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Every workload (application, database, service) defined on a server.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server.",
				Required:            true,
			},
			"resources": schema.ListNestedAttribute{
				MarkdownDescription: "Workloads on the server.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":   schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"type":   schema.StringAttribute{Computed: true, MarkdownDescription: "Workload type."},
						"status": schema.StringAttribute{Computed: true, MarkdownDescription: "Runtime status."},
					},
				},
			},
		},
	}
}

func (d *serverResourcesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *serverResourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverResourcesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resources, err := d.client.GetServerResources(ctx, config.ServerUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read resources of server %s", config.ServerUUID.ValueString()),
			err.Error(),
		)
		return
	}

	state := serverResourcesDataSourceModel{
		ServerUUID: config.ServerUUID,
		Resources:  make([]serverResourceEntryModel, 0, len(resources)),
	}
	for _, res := range resources {
		state.Resources = append(state.Resources, serverResourceEntryModel{
			UUID:   types.StringValue(res.UUID),
			Name:   types.StringValue(res.Name),
			Type:   types.StringValue(res.Type),
			Status: types.StringValue(res.Status),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

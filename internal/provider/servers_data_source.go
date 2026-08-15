package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*serversDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serversDataSource)(nil)
)

// NewServersDataSource is registered in provider.go.
func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

type serversDataSource struct {
	client *client.Client
}

type serverListEntryModel struct {
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

type serversDataSourceModel struct {
	Servers []serverListEntryModel `tfsdk:"servers"`
}

func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (d *serversDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every server registered on the Coolify instance.",
		Attributes: map[string]schema.Attribute{
			"servers": schema.ListNestedAttribute{
				MarkdownDescription: "All servers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":         schema.StringAttribute{Computed: true, MarkdownDescription: "UUID."},
						"name":         schema.StringAttribute{Computed: true, MarkdownDescription: "Name."},
						"description":  schema.StringAttribute{Computed: true, MarkdownDescription: "Description."},
						"ip":           schema.StringAttribute{Computed: true, MarkdownDescription: "IP address or hostname."},
						"port":         schema.Int64Attribute{Computed: true, MarkdownDescription: "SSH port."},
						"user":         schema.StringAttribute{Computed: true, MarkdownDescription: "SSH user."},
						"proxy_type":   schema.StringAttribute{Computed: true, MarkdownDescription: "Reverse proxy type."},
						"is_reachable": schema.BoolAttribute{Computed: true, MarkdownDescription: "Reachability on last check."},
						"is_usable":    schema.BoolAttribute{Computed: true, MarkdownDescription: "Fully validated and usable."},
					},
				},
			},
		},
	}
}

func (d *serversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *serversDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	servers, err := d.client.ListServers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coolify servers", err.Error())
		return
	}

	state := serversDataSourceModel{Servers: make([]serverListEntryModel, 0, len(servers))}
	for _, s := range servers {
		entry := serverListEntryModel{
			UUID:        types.StringValue(s.UUID),
			Name:        types.StringValue(s.Name),
			Description: types.StringValue(s.Description),
			IP:          types.StringValue(s.IP),
			Port:        types.Int64Value(s.Port),
			User:        types.StringValue(s.User),
			ProxyType:   types.StringValue(s.ProxyType),
			IsReachable: types.BoolValue(false),
			IsUsable:    types.BoolValue(false),
		}
		if s.Settings != nil {
			entry.IsReachable = types.BoolValue(s.Settings.IsReachable)
			entry.IsUsable = types.BoolValue(s.Settings.IsUsable)
		}
		state.Servers = append(state.Servers, entry)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

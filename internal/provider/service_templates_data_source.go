package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ datasource.DataSource              = (*serviceTemplatesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serviceTemplatesDataSource)(nil)
)

// NewServiceTemplatesDataSource is registered in provider.go.
func NewServiceTemplatesDataSource() datasource.DataSource {
	return &serviceTemplatesDataSource{}
}

type serviceTemplatesDataSource struct {
	client *client.Client
}

type serviceTemplateModel struct {
	Slogan        types.String `tfsdk:"slogan"`
	Category      types.String `tfsdk:"category"`
	Documentation types.String `tfsdk:"documentation"`
	Port          types.String `tfsdk:"port"`
}

type serviceTemplatesDataSourceModel struct {
	URL       types.String                    `tfsdk:"url"`
	Category  types.String                    `tfsdk:"category"`
	Types     []types.String                  `tfsdk:"types"`
	Templates map[string]serviceTemplateModel `tfsdk:"templates"`
}

func (d *serviceTemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_templates"
}

func (d *serviceTemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The live catalog of Coolify one-click services (300+ templates), " +
			"fetched from the same CDN feed Coolify instances use " +
			"(see [coolify.io/docs/services/all](https://coolify.io/docs/services/all)). " +
			"Use it to validate or discover `coolify_service.type` values dynamically — the " +
			"catalog follows Coolify releases, never this provider's.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Override the catalog URL. Defaults to the official CDN feed " +
					"(`" + client.DefaultServiceTemplatesURL + "`).",
				Optional: true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Only return templates of this category (e.g. `automation`, `analytics`).",
				Optional:            true,
			},
			"types": schema.ListAttribute{
				MarkdownDescription: "Sorted list of valid `coolify_service.type` values.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"templates": schema.MapNestedAttribute{
				MarkdownDescription: "Full catalog keyed by service type.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slogan":        schema.StringAttribute{Computed: true, MarkdownDescription: "Short description."},
						"category":      schema.StringAttribute{Computed: true, MarkdownDescription: "Category."},
						"documentation": schema.StringAttribute{Computed: true, MarkdownDescription: "Documentation URL."},
						"port":          schema.StringAttribute{Computed: true, MarkdownDescription: "Default exposed port."},
					},
				},
			},
		},
	}
}

func (d *serviceTemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *serviceTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serviceTemplatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templates, err := d.client.FetchServiceTemplates(ctx, config.URL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to fetch the Coolify service catalog", err.Error())
		return
	}

	category := config.Category.ValueString()
	state := serviceTemplatesDataSourceModel{
		URL:       config.URL,
		Category:  config.Category,
		Types:     make([]types.String, 0, len(templates)),
		Templates: make(map[string]serviceTemplateModel, len(templates)),
	}
	for _, tpl := range templates {
		if category != "" && tpl.Category != category {
			continue
		}
		state.Types = append(state.Types, types.StringValue(tpl.Type))
		state.Templates[tpl.Type] = serviceTemplateModel{
			Slogan:        types.StringValue(tpl.Slogan),
			Category:      types.StringValue(tpl.Category),
			Documentation: types.StringValue(tpl.Documentation),
			Port:          types.StringValue(tpl.Port),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

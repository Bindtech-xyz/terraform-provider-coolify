package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// ---- coolify_instance (health + version) ----

var (
	_ datasource.DataSource              = (*instanceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*instanceDataSource)(nil)
)

// NewInstanceDataSource is registered in provider.go.
func NewInstanceDataSource() datasource.DataSource {
	return &instanceDataSource{}
}

type instanceDataSource struct {
	client *client.Client
}

type instanceDataSourceModel struct {
	Version types.String `tfsdk:"version"`
	Healthy types.Bool   `tfsdk:"healthy"`
}

func (d *instanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *instanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Health and version of the Coolify instance.",
		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{Computed: true, MarkdownDescription: "Coolify version (e.g. `4.3.3`)."},
			"healthy": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether `/health` answers OK."},
		},
	}
}

func (d *instanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *instanceDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	version, err := d.client.Version(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coolify version", err.Error())
		return
	}
	state := instanceDataSourceModel{
		Version: types.StringValue(version),
		Healthy: types.BoolValue(d.client.Healthy(ctx)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_backup_executions ----

var (
	_ datasource.DataSource              = (*backupExecutionsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*backupExecutionsDataSource)(nil)
)

// NewBackupExecutionsDataSource is registered in provider.go.
func NewBackupExecutionsDataSource() datasource.DataSource {
	return &backupExecutionsDataSource{}
}

type backupExecutionsDataSource struct {
	client *client.Client
}

type backupExecutionEntryModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Status    types.String `tfsdk:"status"`
	Message   types.String `tfsdk:"message"`
	Size      types.Int64  `tfsdk:"size"`
	Filename  types.String `tfsdk:"filename"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type backupExecutionsDataSourceModel struct {
	DatabaseUUID types.String                `tfsdk:"database_uuid"`
	BackupUUID   types.String                `tfsdk:"backup_uuid"`
	Executions   []backupExecutionEntryModel `tfsdk:"executions"`
}

func (d *backupExecutionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_executions"
}

func (d *backupExecutionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The runs of a database backup schedule (`coolify_database_backup`) — " +
			"useful to alert on failed backups.",
		Attributes: map[string]schema.Attribute{
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the database.",
				Required:            true,
			},
			"backup_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the backup schedule.",
				Required:            true,
			},
			"executions": schema.ListNestedAttribute{
				MarkdownDescription: "Backup runs, newest first.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":       schema.StringAttribute{Computed: true, MarkdownDescription: "Run UUID."},
						"status":     schema.StringAttribute{Computed: true, MarkdownDescription: "success / failed / running."},
						"message":    schema.StringAttribute{Computed: true, MarkdownDescription: "Error message when failed."},
						"size":       schema.Int64Attribute{Computed: true, MarkdownDescription: "Backup size in bytes."},
						"filename":   schema.StringAttribute{Computed: true, MarkdownDescription: "Stored filename."},
						"created_at": schema.StringAttribute{Computed: true, MarkdownDescription: "Run timestamp."},
					},
				},
			},
		},
	}
}

func (d *backupExecutionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *backupExecutionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config backupExecutionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	executions, err := d.client.ListBackupExecutions(ctx, config.DatabaseUUID.ValueString(), config.BackupUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list backup executions", err.Error())
		return
	}

	state := backupExecutionsDataSourceModel{
		DatabaseUUID: config.DatabaseUUID,
		BackupUUID:   config.BackupUUID,
	}
	for _, e := range executions {
		state.Executions = append(state.Executions, backupExecutionEntryModel{
			UUID:      types.StringValue(e.UUID),
			Status:    types.StringValue(e.Status),
			Message:   types.StringValue(e.Message),
			Size:      types.Int64Value(e.Size),
			Filename:  types.StringValue(e.Filename),
			CreatedAt: types.StringValue(e.CreatedAt),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_github_app_repositories ----

var (
	_ datasource.DataSource              = (*githubRepositoriesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*githubRepositoriesDataSource)(nil)
)

// NewGithubRepositoriesDataSource is registered in provider.go.
func NewGithubRepositoriesDataSource() datasource.DataSource {
	return &githubRepositoriesDataSource{}
}

type githubRepositoriesDataSource struct {
	client *client.Client
}

type githubRepositoryEntryModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	FullName types.String `tfsdk:"full_name"`
	Private  types.Bool   `tfsdk:"private"`
}

type githubRepositoriesDataSourceModel struct {
	GithubAppID  types.Int64                  `tfsdk:"github_app_id"`
	Repositories []githubRepositoryEntryModel `tfsdk:"repositories"`
}

func (d *githubRepositoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_repositories"
}

func (d *githubRepositoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The repositories a GitHub App (`coolify_github_app`) can access.",
		Attributes: map[string]schema.Attribute{
			"github_app_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric id of the GitHub App.",
				Required:            true,
			},
			"repositories": schema.ListNestedAttribute{
				MarkdownDescription: "Accessible repositories.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":        schema.Int64Attribute{Computed: true, MarkdownDescription: "GitHub repository id."},
						"name":      schema.StringAttribute{Computed: true, MarkdownDescription: "Repository name."},
						"full_name": schema.StringAttribute{Computed: true, MarkdownDescription: "`owner/repo`."},
						"private":   schema.BoolAttribute{Computed: true, MarkdownDescription: "Private repository."},
					},
				},
			},
		},
	}
}

func (d *githubRepositoriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *githubRepositoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config githubRepositoriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repos, err := d.client.ListGithubAppRepositories(ctx, config.GithubAppID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to list repositories of GitHub App %d", config.GithubAppID.ValueInt64()),
			err.Error(),
		)
		return
	}

	state := githubRepositoriesDataSourceModel{GithubAppID: config.GithubAppID}
	for _, repo := range repos {
		state.Repositories = append(state.Repositories, githubRepositoryEntryModel{
			ID:       types.Int64Value(repo.ID),
			Name:     types.StringValue(repo.Name),
			FullName: types.StringValue(repo.FullName),
			Private:  types.BoolValue(repo.Private),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ---- coolify_cloud_catalog ----

var (
	_ datasource.DataSource              = (*cloudCatalogDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cloudCatalogDataSource)(nil)
)

// NewCloudCatalogDataSource is registered in provider.go.
func NewCloudCatalogDataSource() datasource.DataSource {
	return &cloudCatalogDataSource{}
}

type cloudCatalogDataSource struct {
	client *client.Client
}

type cloudCatalogDataSourceModel struct {
	Provider  types.String   `tfsdk:"provider_name"`
	Section   types.String   `tfsdk:"section"`
	TokenUUID types.String   `tfsdk:"cloud_token_uuid"`
	Items     []types.Map    `tfsdk:"items"`
	Names     []types.String `tfsdk:"names"`
}

func (d *cloudCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_catalog"
}

func (d *cloudCatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A provider catalog proxied by Coolify — regions, sizes, images, SSH " +
			"keys… — used to feed `coolify_cloud_server`. Sections per provider: hetzner " +
			"(`locations`, `server-types`, `images`, `ssh-keys`, `firewalls`, `networks`), " +
			"digitalocean (`regions`, `sizes`, `images`, `ssh-keys`), vultr (`regions`, `plans`, " +
			"`os`, `ssh-keys`).",
		Attributes: map[string]schema.Attribute{
			"provider_name": schema.StringAttribute{
				MarkdownDescription: "`hetzner`, `digitalocean` or `vultr`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(client.CloudProviders...)},
			},
			"section": schema.StringAttribute{
				MarkdownDescription: "Catalog section (see list above).",
				Required:            true,
			},
			"cloud_token_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_cloud_token` used to query the provider.",
				Required:            true,
			},
			"items": schema.ListAttribute{
				MarkdownDescription: "Raw catalog entries (each entry flattened to string values).",
				Computed:            true,
				ElementType:         types.MapType{ElemType: types.StringType},
			},
			"names": schema.ListAttribute{
				MarkdownDescription: "Convenience list of each entry's `name` (or `slug`/`id`).",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *cloudCatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(req, resp)
}

func (d *cloudCatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config cloudCatalogDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, err := d.client.CloudCatalog(ctx,
		config.Provider.ValueString(), config.Section.ValueString(), config.TokenUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to fetch %s %s catalog", config.Provider.ValueString(), config.Section.ValueString()),
			err.Error(),
		)
		return
	}

	state := config
	for _, entry := range entries {
		flat := map[string]string{}
		for key, value := range entry {
			flat[key] = fmt.Sprintf("%v", value)
		}
		item, diags := types.MapValueFrom(ctx, types.StringType, flat)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Items = append(state.Items, item)

		name := flat["name"]
		if name == "" {
			name = flat["slug"]
		}
		if name == "" {
			name = flat["id"]
		}
		state.Names = append(state.Names, types.StringValue(name))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

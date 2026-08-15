// Package provider implements the Terraform provider for Coolify on top of the
// terraform-plugin-framework (protocol v6, Terraform >= 1.0 not supported —
// requires Terraform >= 1.1 for protocol 6).
//
// Layout conventions for this package:
//   - one file per resource  (<name>_resource.go)  + its acceptance tests
//   - one file per data source (<name>_data_source.go)
//   - resources/data sources receive a *client.Client through Configure; the
//     framework calls Configure twice, first with a nil ProviderData — always
//     guard for that.
package provider

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// Environment variables honoured when the corresponding attribute is not set in
// configuration. Keeping the token out of .tf files is the recommended usage.
const (
	envEndpoint = "COOLIFY_ENDPOINT"
	envToken    = "COOLIFY_TOKEN"
)

// Interface conformance guards: a compile error here is cheaper than a runtime
// protocol error.
var _ provider.Provider = (*coolifyProvider)(nil)
var _ provider.ProviderWithFunctions = (*coolifyProvider)(nil)

type coolifyProvider struct {
	// version is "dev" for local builds and the release tag for published ones.
	version string
}

// coolifyProviderModel maps the provider block schema to Go values.
type coolifyProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
	Insecure types.Bool   `tfsdk:"insecure"`
	Headers  types.Map    `tfsdk:"headers"`
}

// New returns the provider constructor consumed by providerserver.Serve and by
// acceptance tests.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &coolifyProvider{version: version}
	}
}

func (p *coolifyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "coolify"
	resp.Version = p.version
}

func (p *coolifyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Interact with a [Coolify](https://coolify.io) v4 instance " +
			"(cloud or self-hosted) through its REST API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "URL of the Coolify instance, e.g. `https://coolify.example.com`. " +
					"`/api/v1` is appended automatically. May also be set with the `" + envEndpoint + "` " +
					"environment variable. Defaults to Coolify Cloud (`" + client.DefaultEndpoint + "`).",
				Optional: true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "API token, created in Coolify under **Keys & Tokens → API tokens**. " +
					"May also be set with the `" + envToken + "` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification. Only for self-hosted " +
					"instances with self-signed certificates. Defaults to `false`.",
				Optional: true,
			},
			"headers": schema.MapAttribute{
				MarkdownDescription: "Fixed HTTP headers sent with every request, applied before the " +
					"provider's own `Authorization` header (which always wins on conflict). This is a " +
					"deliberately generic escape hatch — the provider has no built-in notion of any " +
					"particular reverse proxy — for reaching a Coolify instance placed behind an " +
					"authenticating edge, e.g. a Cloudflare Access application gated by a service token:\n\n" +
					"```terraform\n" +
					"provider \"coolify\" {\n" +
					"  headers = {\n" +
					"    \"CF-Access-Client-Id\"     = var.cf_access_client_id\n" +
					"    \"CF-Access-Client-Secret\" = var.cf_access_client_secret\n" +
					"  }\n" +
					"}\n" +
					"```",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
		},
	}
}

func (p *coolifyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config coolifyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown values mean the practitioner wired another resource's output into
	// the provider block and it is not resolvable yet.
	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			pathRoot("endpoint"),
			"Unknown Coolify endpoint",
			"The provider cannot connect: \"endpoint\" is derived from a value that is not known until apply. "+
				"Set it statically or via the "+envEndpoint+" environment variable.",
		)
	}
	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			pathRoot("token"),
			"Unknown Coolify API token",
			"The provider cannot connect: \"token\" is derived from a value that is not known until apply. "+
				"Set it statically or via the "+envToken+" environment variable.",
		)
	}
	if config.Headers.IsUnknown() {
		// A common shape for this: headers built from a resource this same
		// configuration also creates (e.g. a Cloudflare Access service token).
		// Terraform cannot configure a provider from a value produced by a
		// resource in the same apply — the provider must exist before any
		// resource can be planned. Point at the fix instead of failing
		// opaquely deep inside Configure.
		resp.Diagnostics.AddAttributeError(
			pathRoot("headers"),
			"Unknown Coolify provider headers",
			"The provider cannot connect: \"headers\" is derived from a value that is not known until "+
				"apply — often because it references an attribute of a resource this same configuration "+
				"creates. A provider block is evaluated before any resource, so it cannot depend on one "+
				"created in the same apply; create that resource in a separate apply (or a separate "+
				"Terraform run/state) first, then reference its output here.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv(envEndpoint)
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	token := os.Getenv(envToken)
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("token"),
			"Missing Coolify API token",
			"Set the \"token\" attribute in the provider block or the "+envToken+" environment variable. "+
				"Create a token in Coolify under Keys & Tokens → API tokens.",
		)
		return
	}

	opts := []client.Option{
		client.WithUserAgent("terraform-provider-coolify/" + p.version),
	}
	if !config.Headers.IsNull() {
		headers := make(map[string]string, len(config.Headers.Elements()))
		resp.Diagnostics.Append(config.Headers.ElementsAs(ctx, &headers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, client.WithExtraHeaders(headers))
	}
	if config.Insecure.ValueBool() {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- explicit opt-in
		opts = append(opts, client.WithHTTPClient(&http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		}))
	}

	apiClient, err := client.New(endpoint, token, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}

	// Fail fast: one cheap call validates both the endpoint and the token, so
	// practitioners get a clear error at plan time instead of one per resource.
	version, err := apiClient.Version(ctx)
	if err != nil {
		if client.IsUnauthorized(err) {
			resp.Diagnostics.AddAttributeError(
				pathRoot("token"),
				"Invalid Coolify API token",
				"The Coolify instance at "+apiClient.Endpoint()+" rejected the token: "+err.Error(),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to reach Coolify",
				"Checking connectivity against "+apiClient.Endpoint()+" failed: "+err.Error(),
			)
		}
		return
	}

	tflog.Info(ctx, "configured Coolify client", map[string]any{
		"endpoint":        apiClient.Endpoint(),
		"coolify_version": version,
	})

	// The same value serves resources and data sources; both type-assert it back
	// to *client.Client in their own Configure methods.
	resp.ResourceData = apiClient
	resp.DataSourceData = apiClient
}

func (p *coolifyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Infrastructure.
		NewPrivateKeyResource,
		NewServerResource,
		NewDestinationResource,
		// Organisation.
		NewProjectResource,
		NewEnvironmentResource,
		NewTagResource,
		// Workloads.
		NewApplicationResource,
		NewDatabaseResource,
		NewServiceResource,
		// Configuration.
		NewEnvVarResource,
		NewSharedEnvVarResource,
		NewS3StorageResource,
		NewStorageResource,
		NewScheduledTaskResource,
		NewDatabaseBackupResource,
		NewNotificationSettingsResource,
		NewServerSettingsResource,
		NewGithubAppResource,
		NewGitlabAppResource,
		NewVolumeBackupResource,
		NewEnvVarsBulkResource,
		NewResourceActionResource,
		// Cloud provisioning.
		NewCloudTokenResource,
		NewCloudInitScriptResource,
		NewCloudServerResource,
	}
}

func (p *coolifyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
		NewProjectsDataSource,
		NewEnvironmentsDataSource,
		NewServersDataSource,
		NewPrivateKeysDataSource,
		NewApplicationsDataSource,
		NewServicesDataSource,
		NewDatabasesDataSource,
		NewTeamDataSource,
		NewTeamsDataSource,
		NewServiceTemplatesDataSource,
		NewDeploymentsDataSource,
		NewServerDataSource,
		NewServerDomainsDataSource,
		NewServerResourcesDataSource,
		NewTagsDataSource,
		NewDestinationsDataSource,
		NewS3StoragesDataSource,
		NewBackupExecutionsDataSource,
		NewGithubRepositoriesDataSource,
		NewInstanceDataSource,
		NewCloudCatalogDataSource,
	}
}

func (p *coolifyProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}

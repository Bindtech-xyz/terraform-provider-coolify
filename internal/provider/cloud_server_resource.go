package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*cloudServerResource)(nil)
	_ resource.ResourceWithConfigure   = (*cloudServerResource)(nil)
	_ resource.ResourceWithImportState = (*cloudServerResource)(nil)
)

// NewCloudServerResource is registered in provider.go.
func NewCloudServerResource() resource.Resource {
	return &cloudServerResource{}
}

type cloudServerResource struct {
	client *client.Client
}

type cloudServerResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Provider        types.String `tfsdk:"provider_name"`
	CloudTokenUUID  types.String `tfsdk:"cloud_token_uuid"`
	Name            types.String `tfsdk:"name"`
	PrivateKeyUUID  types.String `tfsdk:"private_key_uuid"`
	CloudInitScript types.String `tfsdk:"cloud_init_script"`
	InstantValidate types.Bool   `tfsdk:"instant_validate"`

	// Hetzner.
	Location     types.String `tfsdk:"location"`
	ServerType   types.String `tfsdk:"server_type"`
	HetznerImage types.Int64  `tfsdk:"hetzner_image_id"`
	EnableIPv4   types.Bool   `tfsdk:"enable_ipv4"`
	EnableIPv6   types.Bool   `tfsdk:"enable_ipv6"`

	// DigitalOcean.
	Region     types.String `tfsdk:"region"`
	Size       types.String `tfsdk:"size"`
	Image      types.String `tfsdk:"image"`
	Monitoring types.Bool   `tfsdk:"monitoring"`

	// Vultr.
	Plan types.String `tfsdk:"plan"`
	OSID types.Int64  `tfsdk:"os_id"`

	// Read-only.
	IP       types.String `tfsdk:"ip"`
	IsUsable types.Bool   `tfsdk:"is_usable"`
}

func (r *cloudServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_server"
}

func replaceString() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

func (r *cloudServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provisions a VM at a cloud provider (Hetzner, DigitalOcean or Vultr) " +
			"through Coolify and registers it as a server. Set the provider-specific attributes " +
			"matching `provider_name`. **Destroying this resource deregisters the server from " +
			"Coolify but does not destroy the VM at the provider.** Provisioning attributes force " +
			"replacement — but replacement only re-registers a new VM; clean up the old one at " +
			"the provider.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server UUID (usable wherever a `coolify_server` UUID is expected).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"provider_name": schema.StringAttribute{
				MarkdownDescription: "`hetzner`, `digitalocean` or `vultr`.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(client.CloudProviders...)},
				PlanModifiers:       replaceString(),
			},
			"cloud_token_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_cloud_token` for this provider.",
				Required:            true,
				PlanModifiers:       replaceString(),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Hostname of the VM. Generated when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `coolify_private_key` installed on the VM.",
				Required:            true,
				PlanModifiers:       replaceString(),
			},
			"cloud_init_script": schema.StringAttribute{
				MarkdownDescription: "Inline cloud-init YAML applied at first boot.",
				Optional:            true,
				PlanModifiers:       replaceString(),
			},
			"instant_validate": schema.BoolAttribute{
				MarkdownDescription: "Validate the server right after provisioning. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},

			// Hetzner.
			"location": schema.StringAttribute{
				MarkdownDescription: "Hetzner location (e.g. `fsn1`).",
				Optional:            true,
				PlanModifiers:       replaceString(),
			},
			"server_type": schema.StringAttribute{
				MarkdownDescription: "Hetzner server type (e.g. `cx22`).",
				Optional:            true,
				PlanModifiers:       replaceString(),
			},
			"hetzner_image_id": schema.Int64Attribute{
				MarkdownDescription: "Hetzner image id (integer).",
				Optional:            true,
			},
			"enable_ipv4": schema.BoolAttribute{
				MarkdownDescription: "Attach a public IPv4 (Hetzner).",
				Optional:            true,
			},
			"enable_ipv6": schema.BoolAttribute{
				MarkdownDescription: "Attach a public IPv6 (Hetzner, DigitalOcean).",
				Optional:            true,
			},

			// DigitalOcean.
			"region": schema.StringAttribute{
				MarkdownDescription: "Region slug (DigitalOcean `fra1`, Vultr `fra`).",
				Optional:            true,
				PlanModifiers:       replaceString(),
			},
			"size": schema.StringAttribute{
				MarkdownDescription: "DigitalOcean droplet size slug (e.g. `s-2vcpu-4gb`).",
				Optional:            true,
				PlanModifiers:       replaceString(),
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "DigitalOcean image slug or id (e.g. `ubuntu-24-04-x64`).",
				Optional:            true,
			},
			"monitoring": schema.BoolAttribute{
				MarkdownDescription: "Enable DigitalOcean monitoring.",
				Optional:            true,
			},

			// Vultr.
			"plan": schema.StringAttribute{
				MarkdownDescription: "Vultr plan id (e.g. `vc2-2c-4gb`).",
				Optional:            true,
				PlanModifiers:       replaceString(),
			},
			"os_id": schema.Int64Attribute{
				MarkdownDescription: "Vultr OS id (integer).",
				Optional:            true,
			},

			// Read-only.
			"ip": schema.StringAttribute{
				MarkdownDescription: "Public IP assigned to the VM.",
				Computed:            true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether the server is validated and usable.",
				Computed:            true,
			},
		},
	}
}

func (r *cloudServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *cloudServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	provider := plan.Provider.ValueString()
	server, err := r.client.CreateCloudServer(ctx, provider, client.CloudServerRequest{
		CloudProviderTokenUUID: stringOrNil(plan.CloudTokenUUID),
		Name:                   stringOrNil(plan.Name),
		PrivateKeyUUID:         stringOrNil(plan.PrivateKeyUUID),
		CloudInitScript:        stringOrNil(plan.CloudInitScript),
		InstantValidate:        boolOrNil(plan.InstantValidate),
		Location:               stringOrNil(plan.Location),
		ServerType:             stringOrNil(plan.ServerType),
		HetznerImage:           int64OrNil(plan.HetznerImage),
		EnableIPv4:             boolOrNil(plan.EnableIPv4),
		EnableIPv6:             boolOrNil(plan.EnableIPv6),
		Region:                 stringOrNil(plan.Region),
		Size:                   stringOrNil(plan.Size),
		DigitalOceanImage:      stringOrNil(plan.Image),
		Monitoring:             boolOrNil(plan.Monitoring),
		Plan:                   stringOrNil(plan.Plan),
		OSID:                   int64OrNil(plan.OSID),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to provision %s server through Coolify", provider),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudServerToModel(server, plan))...)
}

func (r *cloudServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.GetServer(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify cloud server %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudServerToModel(server, state))...)
}

// Update only supports renaming; provisioning attributes force replacement.
func (r *cloudServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.UpdateServer(ctx, plan.UUID.ValueString(), client.ServerUpdateRequest{
		ServerCreateRequest: client.ServerCreateRequest{
			Name: stringOrNil(plan.Name),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify cloud server %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, cloudServerToModel(server, plan))...)
}

// Delete deregisters the server from Coolify. The VM itself stays alive at the
// provider and must be cleaned up there.
func (r *cloudServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteServer(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to deregister Coolify cloud server %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.AddWarning(
		"VM not destroyed at the provider",
		"Coolify only deregisters provisioned servers; delete the VM at "+state.Provider.ValueString()+" if it is no longer needed.",
	)
}

func (r *cloudServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func cloudServerToModel(s *client.Server, prior cloudServerResourceModel) cloudServerResourceModel {
	m := prior
	m.UUID = types.StringValue(s.UUID)
	m.Name = types.StringValue(s.Name)
	m.IP = types.StringValue(s.IP)
	m.IsUsable = types.BoolValue(false)
	if s.Settings != nil {
		m.IsUsable = types.BoolValue(s.Settings.IsUsable)
	}
	return m
}

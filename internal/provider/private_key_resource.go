package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*privateKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*privateKeyResource)(nil)
	_ resource.ResourceWithImportState = (*privateKeyResource)(nil)
)

// NewPrivateKeyResource is registered in provider.go.
func NewPrivateKeyResource() resource.Resource {
	return &privateKeyResource{}
}

type privateKeyResource struct {
	client *client.Client
}

type privateKeyResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func (r *privateKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_key"
}

func (r *privateKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An SSH private key stored in Coolify, used to reach servers and " +
			"private git repositories. Coolify rejects keys whose fingerprint already exists.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the key. Coolify generates one when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description. Coolify defaults unset to " +
					"\"Created by Coolify via API\".",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "PEM private key material (or its base64 encoding).",
				Required:            true,
				Sensitive:           true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "SSH fingerprint computed by Coolify.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *privateKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *privateKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.CreatePrivateKey(ctx, client.PrivateKeyRequest{
		Name:        stringOrNil(plan.Name),
		Description: stringOrNil(plan.Description),
		PrivateKey:  stringOrNil(plan.PrivateKey),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify private key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, privateKeyToModel(key, plan))...)
}

func (r *privateKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetPrivateKey(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify private key %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, privateKeyToModel(key, state))...)
}

func (r *privateKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privateKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.UpdatePrivateKey(ctx, plan.UUID.ValueString(), client.PrivateKeyRequest{
		Name:        stringOrNil(plan.Name),
		Description: stringOrNil(plan.Description),
		PrivateKey:  stringOrNil(plan.PrivateKey),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify private key %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, privateKeyToModel(key, plan))...)
}

func (r *privateKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePrivateKey(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify private key %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *privateKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// privateKeyToModel maps the API object to state. The key material is not
// echoed back without the read:sensitive ability, so the configured value is
// preserved from prior state to avoid perpetual diffs.
func privateKeyToModel(k *client.PrivateKey, prior privateKeyResourceModel) privateKeyResourceModel {
	m := privateKeyResourceModel{
		UUID:        types.StringValue(k.UUID),
		Name:        types.StringValue(k.Name),
		Fingerprint: types.StringValue(k.Fingerprint),
		PrivateKey:  prior.PrivateKey,
	}
	if k.PrivateKey != "" && prior.PrivateKey.IsNull() {
		m.PrivateKey = types.StringValue(k.PrivateKey)
	}
	// Computed (Coolify defaults unset to "Created by Coolify via API", never
	// ""): adopt directly, no null-preservation — "" never legitimately occurs.
	m.Description = types.StringValue(k.Description)
	return m
}

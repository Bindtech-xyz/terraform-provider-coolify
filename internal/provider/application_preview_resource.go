package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*applicationPreviewResource)(nil)
	_ resource.ResourceWithConfigure   = (*applicationPreviewResource)(nil)
	_ resource.ResourceWithImportState = (*applicationPreviewResource)(nil)
)

// NewApplicationPreviewResource is registered in provider.go.
func NewApplicationPreviewResource() resource.Resource {
	return &applicationPreviewResource{}
}

type applicationPreviewResource struct {
	client *client.Client
}

type applicationPreviewResourceModel struct {
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	PullRequestID   types.Int64  `tfsdk:"pull_request_id"`
}

func (r *applicationPreviewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_preview"
}

func (r *applicationPreviewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Tracks the lifecycle of a PR preview deployment so it gets cleaned " +
			"up when the resource is destroyed. Coolify has **no create or read API for previews** " +
			"— it creates them automatically from GitHub App pull-request webhook events, never via " +
			"a direct API call — so `create` is state-only (nothing is called), and `destroy` is the " +
			"only real API interaction (`DELETE /applications/{uuid}/previews/{pull_request_id}`). " +
			"Use this to guarantee a preview environment doesn't outlive the `terraform destroy` of " +
			"whatever created it, rather than to provision the preview itself.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the application that owns the preview. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"pull_request_id": schema.Int64Attribute{
				MarkdownDescription: "The pull request number the preview was created for. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *applicationPreviewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

// Create makes no API call — see the schema's MarkdownDescription for why.
func (r *applicationPreviewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationPreviewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read is a no-op: Coolify has no GET for a single preview, so there is
// nothing to reconcile drift against.
func (r *applicationPreviewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationPreviewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update never runs: every attribute forces replacement.
func (r *applicationPreviewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationPreviewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *applicationPreviewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationPreviewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePreview(ctx, state.ApplicationUUID.ValueString(), state.PullRequestID.ValueInt64())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete preview for application %s", state.ApplicationUUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<application_uuid>/<pull_request_id>".
func (r *applicationPreviewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	pullRequestID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if len(parts) != 2 || parts[0] == "" || err != nil {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<application_uuid>/<pull_request_id>\", got %q.", req.ID),
		)
		return
	}

	state := applicationPreviewResourceModel{
		ApplicationUUID: types.StringValue(parts[0]),
		PullRequestID:   types.Int64Value(pullRequestID),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

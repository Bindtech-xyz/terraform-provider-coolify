package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                     = (*deploymentResource)(nil)
	_ resource.ResourceWithConfigure        = (*deploymentResource)(nil)
	_ resource.ResourceWithConfigValidators = (*deploymentResource)(nil)
)

// NewDeploymentResource is registered in provider.go.
func NewDeploymentResource() resource.Resource {
	return &deploymentResource{}
}

type deploymentResource struct {
	client *client.Client
}

type deployResultModel struct {
	ResourceUUID   types.String `tfsdk:"resource_uuid"`
	DeploymentUUID types.String `tfsdk:"deployment_uuid"`
	Message        types.String `tfsdk:"message"`
}

var deployResultAttrTypes = map[string]attr.Type{
	"resource_uuid":   types.StringType,
	"deployment_uuid": types.StringType,
	"message":         types.StringType,
}

// deploymentResourceModel.Results is types.List, not []deployResultModel:
// req.Plan.Get must decode every schema attribute, including Results while
// its plan value is still unknown on Create (no prior state to fall back
// to) — and a plain Go slice cannot represent "unknown" at all ("Value
// Conversion Error", confirmed live against a real deploy). types.List can.
type deploymentResourceModel struct {
	ResourceUUID      types.String `tfsdk:"resource_uuid"`
	Tag               types.String `tfsdk:"tag"`
	Force             types.Bool   `tfsdk:"force"`
	WaitForCompletion types.Bool   `tfsdk:"wait_for_completion"`
	TimeoutSeconds    types.Int64  `tfsdk:"timeout_seconds"`
	Triggers          types.Map    `tfsdk:"triggers"`
	Results           types.List   `tfsdk:"results"`
}

func (r *deploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *deploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers a deployment of one application/database/service " +
			"(`resource_uuid`) or every resource carrying a tag (`tag`), mirroring `POST /deploy`. " +
			"Fire-and-forget by default; set `wait_for_completion` to block the apply until Coolify " +
			"reports the deployment finished. Runs again whenever `triggers` changes (replacement " +
			"re-runs it) — exactly the `coolify_resource_action` pattern, deploy-specific.",
		Attributes: map[string]schema.Attribute{
			"resource_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the application/database/service to deploy. " +
					"Exactly one of `resource_uuid`/`tag`.",
				Optional:      true,
				PlanModifiers: replaceString(),
			},
			"tag": schema.StringAttribute{
				MarkdownDescription: "Deploy every resource carrying this tag " +
					"(see `coolify_resource_tag`). Exactly one of `resource_uuid`/`tag`.",
				Optional:      true,
				PlanModifiers: replaceString(),
			},
			"force": schema.BoolAttribute{
				MarkdownDescription: "Rebuild without Docker layer cache. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "Block until every triggered deployment reaches `finished` " +
					"(or `failed`, which fails the apply). Defaults to `false` — fire-and-forget, " +
					"matching what `instant_deploy` on `coolify_application` already does.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "Deadline for `wait_for_completion`, in seconds. Defaults to `600`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(600),
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary values; any change re-runs the deployment " +
					"(e.g. `{ image_tag = var.image_tag }`).",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapRequiresReplace()},
			},
			"results": schema.ListNestedAttribute{
				MarkdownDescription: "One entry per resource actually queued — more than one " +
					"when deploying by `tag`. Mirrors Coolify's own `/deploy` response.",
				Computed:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"resource_uuid":   schema.StringAttribute{Computed: true, MarkdownDescription: "UUID of the deployed resource."},
						"deployment_uuid": schema.StringAttribute{Computed: true, MarkdownDescription: "UUID of the queued deployment, empty if it could not be queued."},
						"message":         schema.StringAttribute{Computed: true, MarkdownDescription: "Coolify's message for this entry."},
					},
				},
			},
		},
	}
}

func (r *deploymentResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("resource_uuid"),
			path.MatchRoot("tag"),
		),
	}
}

func (r *deploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *deploymentResource) trigger(ctx context.Context, m deploymentResourceModel, resp *resource.CreateResponse) deploymentResourceModel {
	// Coolify's /deploy?tag= does an exact match against the stored tag name,
	// which coolify_resource_tag (and Coolify's own tag-creation endpoints)
	// always normalize to lowercase — an exact, case-preserved match against
	// a mixed-case tag config silently deploys nothing ("404: No resources
	// found with this tag", confirmed live). Normalize the same way here so
	// `tag = "Production"` finds what `tag_name = "Production"` attached.
	tag := strings.ToLower(strings.TrimSpace(m.Tag.ValueString()))
	results, err := r.client.Deploy(ctx, m.ResourceUUID.ValueString(), tag, m.Force.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Unable to trigger Coolify deployment", err.Error())
		m.Results = types.ListValueMust(types.ObjectType{AttrTypes: deployResultAttrTypes}, nil)
		return m
	}

	entries := make([]deployResultModel, 0, len(results))
	for _, res := range results {
		entries = append(entries, deployResultModel{
			ResourceUUID:   types.StringValue(res.ResourceUUID),
			DeploymentUUID: types.StringValue(res.DeploymentUUID),
			Message:        types.StringValue(res.Message),
		})
	}
	resultsList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: deployResultAttrTypes}, entries)
	resp.Diagnostics.Append(diags...)
	m.Results = resultsList

	if !m.WaitForCompletion.ValueBool() {
		return m
	}

	deadline := time.Duration(m.TimeoutSeconds.ValueInt64()) * time.Second
	for _, res := range results {
		if res.DeploymentUUID == "" {
			continue
		}
		if _, err := r.client.WaitForDeploymentCompletion(ctx, res.DeploymentUUID, deadline); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Deployment of %s did not finish successfully", res.ResourceUUID),
				err.Error(),
			)
		}
	}
	return m
}

func (r *deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan = r.trigger(ctx, plan, resp)
	// Always persist state, even on a wait_for_completion failure: the
	// deployment was genuinely queued (Results is populated), so losing
	// track of it on a failed apply would orphan it.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read is a no-op: like coolify_resource_action, this is fire-and-forget —
// there is nothing meaningful to reconcile drift against.
func (r *deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update only ever runs for wait_for_completion/timeout_seconds — every
// other attribute forces replacement, so there's nothing to re-trigger here.
// Results keeps its prior value via UseStateForUnknown.
func (r *deploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete does not touch the deployed resource — deploying is not
// reversible, matching coolify_resource_action's own Delete.
func (r *deploymentResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

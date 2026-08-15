package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

var (
	_ resource.Resource                = (*resourceTagResource)(nil)
	_ resource.ResourceWithConfigure   = (*resourceTagResource)(nil)
	_ resource.ResourceWithImportState = (*resourceTagResource)(nil)
)

// NewResourceTagResource is registered in provider.go.
func NewResourceTagResource() resource.Resource {
	return &resourceTagResource{}
}

type resourceTagResource struct {
	client *client.Client
}

type resourceTagResourceModel struct {
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceUUID types.String `tfsdk:"resource_uuid"`
	TagName      types.String `tfsdk:"tag_name"`
	TagUUID      types.String `tfsdk:"tag_uuid"`
}

func (r *resourceTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_tag"
}

func (r *resourceTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a tag to an application, database or service — the " +
			"attachment `coolify_tag` itself does not create. Coolify creates the team-wide tag " +
			"on first attachment if it doesn't already exist. Tags drive batch deployment via " +
			"`coolify_deployment`'s `tag` attribute (`/deploy?tag=...`).",
		Attributes: map[string]schema.Attribute{
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "`application`, `database` or `service`. Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("application", "database", "service")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"resource_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the resource to tag. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_name": schema.StringAttribute{
				MarkdownDescription: "Tag name (at least 2 characters after Coolify normalizes it — " +
					"trimmed and lowercased). Changing it forces replacement.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tag_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the underlying team-wide tag.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *resourceTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func taggableType(s string) client.TaggableResourceType {
	return client.TaggableResourceType(s)
}

// findTag returns the entry matching name case-insensitively — Coolify
// normalizes tag names to lowercase server-side (trim + lowercase), so an
// exact-case match against a mixed-case config would never hit.
func findTag(tags []client.Tag, name string) *client.Tag {
	name = strings.ToLower(strings.TrimSpace(name))
	for i := range tags {
		if tags[i].Name == name {
			return &tags[i]
		}
	}
	return nil
}

func (r *resourceTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := r.client.AttachResourceTag(ctx, taggableType(plan.ResourceType.ValueString()), plan.ResourceUUID.ValueString(), plan.TagName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to attach Coolify tag", err.Error())
		return
	}
	tag := findTag(tags, plan.TagName.ValueString())
	if tag == nil {
		resp.Diagnostics.AddError(
			"Tag attached but not found in response",
			fmt.Sprintf("Coolify accepted tag %q but it is not in the resource's tag list afterwards.", plan.TagName.ValueString()),
		)
		return
	}

	plan.TagUUID = types.StringValue(tag.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := r.client.ListResourceTags(ctx, taggableType(state.ResourceType.ValueString()), state.ResourceUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read tags of %s %s", state.ResourceType.ValueString(), state.ResourceUUID.ValueString()),
			err.Error(),
		)
		return
	}
	if findTag(tags, state.TagName.ValueString()) == nil {
		// Detached outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update never runs: every attribute forces replacement.
func (r *resourceTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DetachResourceTag(ctx, taggableType(state.ResourceType.ValueString()), state.ResourceUUID.ValueString(), state.TagUUID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to detach Coolify tag %s", state.TagUUID.ValueString()),
			err.Error(),
		)
	}
}

// ImportState expects "<resource_type>/<resource_uuid>/<tag_name>".
func (r *resourceTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected \"<resource_type>/<resource_uuid>/<tag_name>\", got %q.", req.ID),
		)
		return
	}
	resourceType, resourceUUID, tagName := parts[0], parts[1], parts[2]

	tags, err := r.client.ListResourceTags(ctx, taggableType(resourceType), resourceUUID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import Coolify resource tag", err.Error())
		return
	}
	tag := findTag(tags, tagName)
	if tag == nil {
		resp.Diagnostics.AddError(
			"Tag not found",
			fmt.Sprintf("%s %s has no tag named %q.", resourceType, resourceUUID, tagName),
		)
		return
	}

	state := resourceTagResourceModel{
		ResourceType: types.StringValue(resourceType),
		ResourceUUID: types.StringValue(resourceUUID),
		TagName:      types.StringValue(tagName),
		TagUUID:      types.StringValue(tag.UUID),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

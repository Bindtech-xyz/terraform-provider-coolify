package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource              = (*envVarsBulkResource)(nil)
	_ resource.ResourceWithConfigure = (*envVarsBulkResource)(nil)
)

// NewEnvVarsBulkResource is registered in provider.go.
func NewEnvVarsBulkResource() resource.Resource {
	return &envVarsBulkResource{}
}

type envVarsBulkResource struct {
	client *client.Client
}

// envVarsBulkResourceModel manages a whole set of variables in one bulk call —
// the ergonomic choice when wiring dozens of variables per application.
type envVarsBulkResourceModel struct {
	ParentType types.String `tfsdk:"parent_type"`
	ParentUUID types.String `tfsdk:"parent_uuid"`
	Variables  types.Map    `tfsdk:"variables"`
}

func (r *envVarsBulkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variables"
}

func (r *envVarsBulkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A batch of environment variables applied through the bulk endpoint — " +
			"ideal for wiring many variables at once. Keys removed from the map are deleted from " +
			"Coolify on the next apply. For per-variable flags (`is_preview`, `is_literal`…), " +
			"use `coolify_environment_variable` instead; both can coexist on the same resource " +
			"as long as keys do not overlap.",
		Attributes: map[string]schema.Attribute{
			"parent_type": schema.StringAttribute{
				MarkdownDescription: "`application`, `service` or `database`. Changing it forces replacement.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("application", "service", "database")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"parent_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent resource. Changing it forces replacement.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "Map of variable name → value.",
				Required:            true,
				ElementType:         types.StringType,
				Sensitive:           true,
			},
		},
	}
}

func (r *envVarsBulkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *envVarsBulkResource) parent(m envVarsBulkResourceModel) (client.EnvVarParent, string) {
	return client.EnvVarParent(m.ParentType.ValueString()), m.ParentUUID.ValueString()
}

func (r *envVarsBulkResource) variables(ctx context.Context, m envVarsBulkResourceModel) (map[string]string, error) {
	vars := map[string]string{}
	if diags := m.Variables.ElementsAs(ctx, &vars, false); diags.HasError() {
		return nil, fmt.Errorf("invalid variables map")
	}
	return vars, nil
}

// apply pushes the whole map through the bulk endpoint and deletes managed keys
// that disappeared from the configuration.
func (r *envVarsBulkResource) apply(ctx context.Context, plan envVarsBulkResourceModel, priorKeys map[string]struct{}) error {
	parent, parentUUID := r.parent(plan)
	vars, err := r.variables(ctx, plan)
	if err != nil {
		return err
	}

	if len(vars) > 0 {
		batch := make([]client.EnvVarRequest, 0, len(vars))
		for key, value := range vars {
			k, v := key, value
			batch = append(batch, client.EnvVarRequest{Key: &k, Value: &v})
		}
		if _, err := r.client.UpdateEnvVarsBulk(ctx, parent, parentUUID, batch); err != nil {
			return err
		}
	}

	// Delete keys we managed before but which left the configuration.
	var stale []string
	for key := range priorKeys {
		if _, still := vars[key]; !still {
			stale = append(stale, key)
		}
	}
	if len(stale) > 0 {
		existing, err := r.client.ListEnvVars(ctx, parent, parentUUID)
		if err != nil {
			return err
		}
		byKey := map[string]string{}
		for _, v := range existing {
			byKey[v.Key] = v.UUID
		}
		for _, key := range stale {
			if uuid, ok := byKey[key]; ok {
				if err := r.client.DeleteEnvVar(ctx, parent, parentUUID, uuid); err != nil && !client.IsNotFound(err) {
					return err
				}
			}
		}
	}
	return nil
}

func (r *envVarsBulkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan envVarsBulkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, plan, nil); err != nil {
		resp.Diagnostics.AddError("Unable to apply Coolify environment variables", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *envVarsBulkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state envVarsBulkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	existing, err := r.client.ListEnvVars(ctx, parent, parentUUID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Coolify environment variables", err.Error())
		return
	}

	managed, err := r.variables(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Invalid state", err.Error())
		return
	}

	// Refresh only managed keys; values hidden without read:sensitive keep the
	// configured value (empty string from the API means hidden).
	byKey := map[string]string{}
	for _, v := range existing {
		byKey[v.Key] = v.Value
	}
	refreshed := map[string]string{}
	for key, prior := range managed {
		if value, ok := byKey[key]; ok {
			if value == "" {
				value = prior
			}
			refreshed[key] = value
		}
		// Keys deleted outside Terraform drop out and show as re-create diffs.
	}

	variables, diags := types.MapValueFrom(ctx, types.StringType, refreshed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Variables = variables
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *envVarsBulkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior envVarsBulkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorVars, err := r.variables(ctx, prior)
	if err != nil {
		resp.Diagnostics.AddError("Invalid prior state", err.Error())
		return
	}
	priorKeys := make(map[string]struct{}, len(priorVars))
	for key := range priorVars {
		priorKeys[key] = struct{}{}
	}

	if err := r.apply(ctx, plan, priorKeys); err != nil {
		resp.Diagnostics.AddError("Unable to update Coolify environment variables", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *envVarsBulkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state envVarsBulkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent, parentUUID := r.parent(state)
	managed, err := r.variables(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Invalid state", err.Error())
		return
	}

	existing, err := r.client.ListEnvVars(ctx, parent, parentUUID)
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Unable to list Coolify environment variables", err.Error())
		return
	}
	for _, v := range existing {
		if _, ok := managed[v.Key]; ok {
			if err := r.client.DeleteEnvVar(ctx, parent, parentUUID, v.UUID); err != nil && !client.IsNotFound(err) {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Unable to delete environment variable %s", v.Key),
					err.Error(),
				)
				return
			}
		}
	}
}

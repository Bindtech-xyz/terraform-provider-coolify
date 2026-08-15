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
	_ resource.Resource                = (*s3StorageResource)(nil)
	_ resource.ResourceWithConfigure   = (*s3StorageResource)(nil)
	_ resource.ResourceWithImportState = (*s3StorageResource)(nil)
)

// NewS3StorageResource is registered in provider.go.
func NewS3StorageResource() resource.Resource {
	return &s3StorageResource{}
}

type s3StorageResource struct {
	client *client.Client
}

type s3StorageResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`
}

func (r *s3StorageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

func (r *s3StorageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An S3-compatible storage used as a backup destination for databases.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description. Coolify only accepts letters (including " +
					"Unicode), numbers, whitespace, and `- _ . , ! ? ( ) ' \" + = * @ / &` — other " +
					"punctuation (e.g. a colon or semicolon) is rejected with a 422.",
				Optional: true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "S3 endpoint URL, e.g. `https://s3.eu-west-1.amazonaws.com` " +
					"or a MinIO/Garage endpoint.",
				Required: true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "Bucket name.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Bucket region (any non-empty value for S3-compatible stores).",
				Required:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "Access key id.",
				Required:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret access key.",
				Required:            true,
				Sensitive:           true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify validated connectivity to the bucket.",
				Computed:            true,
			},
		},
	}
}

func (r *s3StorageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(req, resp)
}

func (r *s3StorageResource) toRequest(m s3StorageResourceModel) client.S3StorageRequest {
	return client.S3StorageRequest{
		Name:        stringOrNil(m.Name),
		Description: stringOrNil(m.Description),
		Endpoint:    stringOrNil(m.Endpoint),
		Bucket:      stringOrNil(m.Bucket),
		Region:      stringOrNil(m.Region),
		Key:         stringOrNil(m.AccessKey),
		Secret:      stringOrNil(m.SecretKey),
	}
}

func (r *s3StorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan s3StorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	storage, err := r.client.CreateS3Storage(ctx, r.toRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coolify S3 storage", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, s3StorageToModel(storage, plan))...)
}

func (r *s3StorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	storage, err := r.client.GetS3Storage(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Coolify S3 storage %s", state.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, s3StorageToModel(storage, state))...)
}

func (r *s3StorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan s3StorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	storage, err := r.client.UpdateS3Storage(ctx, plan.UUID.ValueString(), r.toRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update Coolify S3 storage %s", plan.UUID.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, s3StorageToModel(storage, plan))...)
}

func (r *s3StorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state s3StorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteS3Storage(ctx, state.UUID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Coolify S3 storage %s", state.UUID.ValueString()),
			err.Error(),
		)
	}
}

func (r *s3StorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// s3StorageToModel maps the API object to state. Credentials are hidden by the
// API unless the token has read:sensitive, so configured values win.
func s3StorageToModel(s *client.S3Storage, prior s3StorageResourceModel) s3StorageResourceModel {
	m := s3StorageResourceModel{
		UUID:      types.StringValue(s.UUID),
		Name:      types.StringValue(s.Name),
		Endpoint:  types.StringValue(s.Endpoint),
		Bucket:    types.StringValue(s.Bucket),
		Region:    types.StringValue(s.Region),
		AccessKey: prior.AccessKey,
		SecretKey: prior.SecretKey,
		IsUsable:  types.BoolValue(s.IsUsable),
	}
	if s.Description != "" || !prior.Description.IsNull() {
		m.Description = types.StringValue(s.Description)
	} else {
		m.Description = types.StringNull()
	}
	if m.AccessKey.IsNull() && s.Key != "" {
		m.AccessKey = types.StringValue(s.Key)
	}
	if m.SecretKey.IsNull() && s.Secret != "" {
		m.SecretKey = types.StringValue(s.Secret)
	}
	return m
}

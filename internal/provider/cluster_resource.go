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
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

// ClusterResource defines the k0s_cluster resource.
type ClusterResource struct {
	// binaryPath is inherited from provider configuration.
	binaryPath string
}

// ClusterResourceModel describes the resource data model.
type ClusterResourceModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Version         types.String `tfsdk:"version"`
	Kubeconfig      types.String `tfsdk:"kubeconfig"`
	SingleNode      types.Bool   `tfsdk:"single_node"`
	ControllerCount types.Int64  `tfsdk:"controller_count"`
	WorkerCount     types.Int64  `tfsdk:"worker_count"`
}

func (r *ClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a k0s testing cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the k0s cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "k0s version to deploy (e.g. v1.31.0+k0s.0).",
				Optional:            true,
				Computed:            true,
			},
			"kubeconfig": schema.StringAttribute{
				MarkdownDescription: "Kubeconfig contents for accessing the cluster.",
				Computed:            true,
				Sensitive:           true,
			},
			"single_node": schema.BoolAttribute{
				MarkdownDescription: "Whether to create a single-node controller+worker cluster.",
				Optional:            true,
				Computed:            true,
			},
			"controller_count": schema.Int64Attribute{
				MarkdownDescription: "Number of controller nodes.",
				Optional:            true,
				Computed:            true,
			},
			"worker_count": schema.Int64Attribute{
				MarkdownDescription: "Number of worker nodes.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *ClusterResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	binaryPath, ok := req.ProviderData.(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected string, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.binaryPath = binaryPath
}

func (r *ClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: implement cluster creation using k0s/k0sctl.
	data.Id = types.StringValue(data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: implement cluster read to refresh state.

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: implement cluster updates (e.g. scaling).

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: implement cluster deletion.
}

func (r *ClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

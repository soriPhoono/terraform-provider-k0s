package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultK0sVersion = "v1.35.1-k0s.1"
const clusterReadyTimeout = 90 * time.Second
const readinessPollInterval = 2 * time.Second

var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

type ClusterResource struct {
	binaryPath string
}

type ClusterResourceModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Version         types.String `tfsdk:"version"`
	Image           types.String `tfsdk:"image"`
	Kubeconfig      types.String `tfsdk:"kubeconfig"`
	SingleNode      types.Bool   `tfsdk:"single_node"`
	ControllerCount types.Int64  `tfsdk:"controller_count"`
	WorkerCount     types.Int64  `tfsdk:"worker_count"`
}

func imageForVersion(version string) string {
	tag := strings.ReplaceAll(version, "+", "-")
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return "docker.io/k0sproject/k0s:" + tag
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
		MarkdownDescription: "Manages a k0s testing cluster using Docker.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Docker container name / unique identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name; used as the Docker container name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "k0s version (e.g. v1.31.0+k0s.0).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultK0sVersion),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Full OCI image reference. Computed from version if not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kubeconfig": schema.StringAttribute{
				MarkdownDescription: "Kubeconfig contents for accessing the cluster.",
				Computed:            true,
				Sensitive:           true,
			},
			"single_node": schema.BoolAttribute{
				MarkdownDescription: "Run a single-node cluster (controller + worker in one container).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"controller_count": schema.Int64Attribute{
				MarkdownDescription: "Number of controller nodes (multi-node support coming soon).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_count": schema.Int64Attribute{
				MarkdownDescription: "Number of worker nodes (multi-node support coming soon).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
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
			fmt.Sprintf("Expected string, got: %T.", req.ProviderData),
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

	name := data.Name.ValueString()
	docker := newDockerClient(r.binaryPath)

	image := data.Image.ValueString()
	if image == "" {
		version := data.Version.ValueString()
		image = imageForVersion(version)
	}

	containerArgs := []string{"k0s", "controller"}
	if data.SingleNode.ValueBool() {
		containerArgs = append(containerArgs, "--enable-worker")
	}

	ports := []string{"6443:6443"}
	if !data.SingleNode.ValueBool() {
		ports = append(ports, "9443:9443", "8132:8132")
	}

	_, err := docker.createContainer(ctx,
		name, name, image,
		true,
		ports,
		[]string{"/var/lib/k0s"},
		[]string{"/run"},
		containerArgs,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create container", err.Error())
		return
	}

	if err := docker.startContainer(ctx, name); err != nil {
		_ = docker.removeContainer(ctx, name, true)
		resp.Diagnostics.AddError("Failed to start container", err.Error())
		return
	}

	kubeconfig, err := waitForReadiness(ctx, docker, name)
	if err != nil {
		_ = docker.removeContainer(ctx, name, true)
		resp.Diagnostics.AddError("Cluster did not become ready", err.Error())
		return
	}

	data.Id = types.StringValue(name)
	data.Image = types.StringValue(image)
	data.Kubeconfig = types.StringValue(kubeconfig)

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

	docker := newDockerClient(r.binaryPath)
	running, err := docker.isRunning(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to inspect container", err.Error())
		return
	}

	if !running {
		resp.State.RemoveResource(ctx)
		return
	}

	kubeconfig, err := docker.exec(ctx, data.Id.ValueString(), "k0s", "kubeconfig", "admin")
	if err == nil {
		data.Kubeconfig = types.StringValue(kubeconfig)
	}

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

	docker := newDockerClient(r.binaryPath)
	if err := docker.removeContainer(ctx, data.Id.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Failed to delete container", err.Error())
	}
}

func (r *ClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func waitForReadiness(ctx context.Context, docker *dockerClient, name string) (string, error) {
	deadline := time.Now().Add(clusterReadyTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		kubeconfig, err := docker.exec(ctx, name, "k0s", "kubeconfig", "admin")
		if err == nil && strings.Contains(kubeconfig, "server:") {
			return kubeconfig, nil
		}

		time.Sleep(readinessPollInterval)
	}
	return "", fmt.Errorf(
		"timed out after %v waiting for cluster to become ready",
		clusterReadyTimeout,
	)
}

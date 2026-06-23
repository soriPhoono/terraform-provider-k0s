package provider

import (
	"context"
	"fmt"
	"os"
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
const clusterReadyTimeout = 120 * time.Second
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

func controllerName(cluster string, idx int) string {
	return fmt.Sprintf("%s-controller-%d", cluster, idx)
}

func workerName(cluster string, idx int) string {
	return fmt.Sprintf("%s-worker-%d", cluster, idx)
}

func networkName(cluster string) string {
	return "k0s-" + cluster
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
				MarkdownDescription: "Cluster identifier (the cluster name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name; used to name containers and the Docker network.",
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
				MarkdownDescription: "Run a single-node cluster (controller + worker in one container). " +
					"When false, separate controller and worker containers are created.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"controller_count": schema.Int64Attribute{
				MarkdownDescription: "Number of controller nodes (only used when single_node is false).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_count": schema.Int64Attribute{
				MarkdownDescription: "Number of worker nodes (only used when single_node is false).",
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
		image = imageForVersion(data.Version.ValueString())
	}

	if data.SingleNode.ValueBool() {
		createSingleNode(ctx, docker, name, image, &data, resp)
	} else {
		createMultiNode(ctx, docker, name, image, &data, resp)
	}
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
	clusterName := data.Id.ValueString()

	if data.SingleNode.ValueBool() {
		running, err := docker.isRunning(ctx, clusterName)
		if err != nil {
			resp.Diagnostics.AddError("Failed to inspect container", err.Error())
			return
		}
		if !running {
			resp.State.RemoveResource(ctx)
			return
		}
		kubeconfig, err := docker.exec(ctx, clusterName, "k0s", "kubeconfig", "admin")
		if err == nil {
			data.Kubeconfig = types.StringValue(kubeconfig)
		}
	} else {
		ctrlName := controllerName(clusterName, 1)
		running, err := docker.isRunning(ctx, ctrlName)
		if err != nil {
			resp.Diagnostics.AddError("Failed to inspect controller", err.Error())
			return
		}
		if !running {
			resp.State.RemoveResource(ctx)
			return
		}
		kubeconfig, err := docker.exec(ctx, ctrlName, "k0s", "kubeconfig", "admin")
		if err == nil {
			data.Kubeconfig = types.StringValue(kubeconfig)
		}
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

	clusterName := data.Id.ValueString()
	docker := newDockerClient(r.binaryPath)

	if data.SingleNode.ValueBool() {
		_ = docker.removeContainer(ctx, clusterName, true)
	} else {
		for i := 1; i <= int(data.WorkerCount.ValueInt64()); i++ {
			_ = docker.removeContainer(ctx, workerName(clusterName, i), true)
		}
		for i := 1; i <= int(data.ControllerCount.ValueInt64()); i++ {
			_ = docker.removeContainer(ctx, controllerName(clusterName, i), true)
		}
		_ = docker.removeNetwork(ctx, networkName(clusterName))
	}
}

func (r *ClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- single-node -----------------------------------------------------------

func createSingleNode(
	ctx context.Context,
	docker *dockerClient,
	name, image string,
	data *ClusterResourceModel,
	resp *resource.CreateResponse,
) {
	containerArgs := []string{"k0s", "controller", "--enable-worker"}
	ports := []string{"6443:6443"}

	_, err := docker.createContainer(ctx,
		name, name, image,
		true, ports,
		[]string{"/var/lib/k0s"}, []string{"/run"},
		"",
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
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// --- multi-node ------------------------------------------------------------

func createMultiNode(
	ctx context.Context,
	docker *dockerClient,
	clusterName, image string,
	data *ClusterResourceModel,
	resp *resource.CreateResponse,
) {
	netName := networkName(clusterName)

	if _, err := docker.createNetwork(ctx, netName); err != nil {
		resp.Diagnostics.AddError("Failed to create network", err.Error())
		return
	}
	removeNet := true
	defer func() {
		if removeNet {
			_ = docker.removeNetwork(ctx, netName)
		}
	}()

	cc := int(data.ControllerCount.ValueInt64())
	wc := int(data.WorkerCount.ValueInt64())

	// --- create controllers ---
	for i := 1; i <= cc; i++ {
		cName := controllerName(clusterName, i)
		ports := []string{fmt.Sprintf("%d:6443", 6443+i-1)}
		if i == 1 {
			ports = append(ports, fmt.Sprintf("%d:9443", 9443+i-1))
		}

		ctrlArgs := []string{"k0s", "controller"}
		_, err := docker.createContainer(ctx,
			cName, cName, image,
			true, ports,
			[]string{"/var/lib/k0s"}, []string{"/run"},
			netName,
			ctrlArgs,
		)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %s", cName), err.Error())
			return
		}

		if err := docker.startContainer(ctx, cName); err != nil {
			_ = docker.removeContainer(ctx, cName, true)
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to start %s", cName), err.Error())
			return
		}
	}

	firstController := controllerName(clusterName, 1)

	kubeconfig, err := waitForReadiness(ctx, docker, firstController)
	if err != nil {
		resp.Diagnostics.AddError("Cluster did not become ready", err.Error())
		return
	}

	// --- generate worker token ---
	token, err := docker.exec(ctx, firstController, "k0s", "token", "create", "--role=worker")
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate worker join token", err.Error())
		return
	}
	token = strings.TrimSpace(token)

	tokenFile, err := os.CreateTemp("", "k0s-join-token-*")
	if err != nil {
		resp.Diagnostics.AddError("Failed to create temp token file", err.Error())
		return
	}
	tokenPath := tokenFile.Name()
	if _, err := tokenFile.WriteString(token); err != nil {
		_ = tokenFile.Close()
		_ = os.Remove(tokenPath)
		resp.Diagnostics.AddError("Failed to write token file", err.Error())
		return
	}
	_ = tokenFile.Close()
	defer func() { _ = os.Remove(tokenPath) }()

	// --- create workers ---
	for i := 1; i <= wc; i++ {
		wName := workerName(clusterName, i)
		workerArgs := []string{
			"k0s", "worker",
			"--token-file", tokenPath,
		}

		_, err := docker.createContainer(ctx,
			wName, wName, image,
			true, nil,
			[]string{"/var/lib/k0s", tokenPath + ":" + tokenPath},
			[]string{"/run"},
			netName,
			workerArgs,
		)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %s", wName), err.Error())
			return
		}

		if err := docker.startContainer(ctx, wName); err != nil {
			_ = docker.removeContainer(ctx, wName, true)
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to start %s", wName), err.Error())
			return
		}
	}

	removeNet = false
	data.Id = types.StringValue(clusterName)
	data.Image = types.StringValue(image)
	data.Kubeconfig = types.StringValue(kubeconfig)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// --- readiness -------------------------------------------------------------

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

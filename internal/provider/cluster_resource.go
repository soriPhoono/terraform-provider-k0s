package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultK0sVersion = "v1.35.1-k0s.1"
const readinessPollInterval = 2 * time.Second

var dockerNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

type dockerNameValidator struct{}

func (v dockerNameValidator) Description(_ context.Context) string {
	return "must be a valid Docker container name"
}

func (v dockerNameValidator) MarkdownDescription(_ context.Context) string {
	return "must be at most 128 characters and match `[a-zA-Z0-9][a-zA-Z0-9_.-]*`"
}

func (v dockerNameValidator) ValidateString(
	_ context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	name := req.ConfigValue.ValueString()
	if len(name) > 128 || !dockerNameRe.MatchString(name) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid container name",
			fmt.Sprintf(
				"Container name %q must be at most 128 characters and match [a-zA-Z0-9][a-zA-Z0-9_.-]*",
				name,
			),
		)
	}
}

type atLeastInt64Validator struct {
	min int64
}

func (v atLeastInt64Validator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be at least %d", v.min)
}

func (v atLeastInt64Validator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value must be at least `%d`", v.min)
}

func (v atLeastInt64Validator) ValidateInt64(
	_ context.Context,
	req validator.Int64Request,
	resp *validator.Int64Response,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() < v.min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Sprintf("Value must be at least %d, got %d", v.min, req.ConfigValue.ValueInt64()),
		)
	}
}

var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

type ClusterResource struct {
	binaryPath string
}

type ClusterResourceModel struct {
	Id                   types.String   `tfsdk:"id"`
	Name                 types.String   `tfsdk:"name"`
	Version              types.String   `tfsdk:"version"`
	Image                types.String   `tfsdk:"image"`
	Kubeconfig           types.String   `tfsdk:"kubeconfig"`
	KubeconfigPath       types.String   `tfsdk:"kubeconfig_path"`
	WaitForReady         types.Bool     `tfsdk:"wait_for_ready"`
	Ports                types.List     `tfsdk:"ports"`
	Volumes              types.List     `tfsdk:"volumes"`
	Tmpfs                types.List     `tfsdk:"tmpfs"`
	Env                  types.Map      `tfsdk:"env"`
	ExtraArgs            types.List     `tfsdk:"extra_args"`
	Cpu                  types.String   `tfsdk:"cpu"`
	Memory               types.String   `tfsdk:"memory"`
	ReadinessTimeout     types.String   `tfsdk:"readiness_timeout"`
	Network              types.String   `tfsdk:"network"`
	Timeouts             timeouts.Value `tfsdk:"timeouts"`
	Endpoint             types.String   `tfsdk:"endpoint"`
	ClientCertificate    types.String   `tfsdk:"client_certificate"`
	ClientKey            types.String   `tfsdk:"client_key"`
	ClusterCACertificate types.String   `tfsdk:"cluster_ca_certificate"`
	SingleNode           types.Bool     `tfsdk:"single_node"`
	ControllerCount      types.Int64    `tfsdk:"controller_count"`
	WorkerCount          types.Int64    `tfsdk:"worker_count"`
}

func imageForVersion(version string) string {
	if version == "" {
		return ""
	}
	tag := strings.ReplaceAll(version, "+", "-")
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return "docker.io/k0sproject/k0s:" + tag
}

func expandStringList(v types.List) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	elems := v.Elements()
	r := make([]string, len(elems))
	for i, e := range elems {
		r[i] = e.(types.String).ValueString()
	}
	return r
}

func expandStringMap(v types.Map) map[string]string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	elems := v.Elements()
	r := make(map[string]string, len(elems))
	for k, e := range elems {
		r[k] = e.(types.String).ValueString()
	}
	return r
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
				Validators: []validator.String{
					dockerNameValidator{},
				},
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
			"kubeconfig_path": schema.StringAttribute{
				MarkdownDescription: "Path to write the kubeconfig file on the local filesystem.",
				Optional:            true,
			},
			"wait_for_ready": schema.BoolAttribute{
				MarkdownDescription: "Wait for the cluster control plane to be ready before returning.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"ports": schema.ListAttribute{
				MarkdownDescription: "Container port mappings (e.g. [\"6443:6443\"]). " +
					"Defaults to auto-assigned host ports when not set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"volumes": schema.ListAttribute{
				MarkdownDescription: "Container volume mounts (e.g. [\"/host/path:/container/path\"]). " +
					"Defaults to [\"/var/lib/k0s\"] when not set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tmpfs": schema.ListAttribute{
				MarkdownDescription: "Container tmpfs mounts (e.g. [\"/run\"]). " +
					"Defaults to [\"/run\"] when not set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"env": schema.MapAttribute{
				MarkdownDescription: "Environment variables to set in the container.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"extra_args": schema.ListAttribute{
				MarkdownDescription: "Extra CLI arguments to pass to the k0s command " +
					"(e.g. [\"--kubelet-extra-flags\", \"--fail-swap-on=false\"]).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"cpu": schema.StringAttribute{
				MarkdownDescription: "CPU limit for the container (e.g. \"0.5\", \"2\"). " +
					"Maps to Docker --cpus.",
				Optional: true,
			},
			"memory": schema.StringAttribute{
				MarkdownDescription: "Memory limit for the container (e.g. \"512m\", \"2g\"). " +
					"Maps to Docker --memory.",
				Optional: true,
			},
			"readiness_timeout": schema.StringAttribute{
				MarkdownDescription: "Maximum time to wait for the cluster control plane to become ready " +
					"(e.g. \"30s\", \"5m\"). Defaults to \"120s\".",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("120s"),
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "Docker network to use for the cluster containers. " +
					"When set, the network must already exist and will not be removed on destroy. " +
					"When unset in multi-node mode, a network is auto-created as k0s-{name}.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Delete: true,
			}),
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Kubernetes API server endpoint.",
				Computed:            true,
			},
			"client_certificate": schema.StringAttribute{
				MarkdownDescription: "Client certificate for authenticating to the cluster.",
				Computed:            true,
				Sensitive:           true,
			},
			"client_key": schema.StringAttribute{
				MarkdownDescription: "Client key for authenticating to the cluster.",
				Computed:            true,
				Sensitive:           true,
			},
			"cluster_ca_certificate": schema.StringAttribute{
				MarkdownDescription: "CA certificate for verifying the API server.",
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
				Validators: []validator.Int64{
					atLeastInt64Validator{min: 1},
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"worker_count": schema.Int64Attribute{
				MarkdownDescription: "Number of worker nodes (only used when single_node is false).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators: []validator.Int64{
					atLeastInt64Validator{min: 0},
				},
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

	waitForReady := data.WaitForReady.ValueBool()

	readinessTimeout, diags := data.Timeouts.Create(ctx, 120*time.Second)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if customTimeout, err := time.ParseDuration(data.ReadinessTimeout.ValueString()); err == nil {
		readinessTimeout = customTimeout
	}

	if data.SingleNode.ValueBool() {
		createSingleNode(ctx, docker, name, image, waitForReady, readinessTimeout, &data, resp)
	} else {
		createMultiNode(ctx, docker, name, image, waitForReady, readinessTimeout, &data, resp)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	setKubeconfigOutputs(&data, &resp.Diagnostics)

	if data.KubeconfigPath.ValueString() != "" {
		if err := writeKubeconfigFile(
			data.KubeconfigPath.ValueString(),
			data.Kubeconfig.ValueString(),
		); err != nil {
			resp.Diagnostics.AddWarning("Could not write kubeconfig file", err.Error())
		}
	}

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

	readTimeout, diags := data.Timeouts.Read(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

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
		setKubeconfigOutputs(&data, &resp.Diagnostics)
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
		setKubeconfigOutputs(&data, &resp.Diagnostics)
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

	deleteTimeout, diags := data.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	clusterName := data.Id.ValueString()
	docker := newDockerClient(r.binaryPath)

	if data.SingleNode.ValueBool() {
		if err := docker.removeContainer(ctx, clusterName, true); err != nil {
			resp.Diagnostics.AddWarning("Failed to remove container", err.Error())
		}
	} else {
		for i := 1; i <= int(data.WorkerCount.ValueInt64()); i++ {
			if err := docker.removeContainer(ctx, workerName(clusterName, i), true); err != nil {
				resp.Diagnostics.AddWarning("Failed to remove worker container", err.Error())
			}
		}
		for i := 1; i <= int(data.ControllerCount.ValueInt64()); i++ {
			if err := docker.removeContainer(
				ctx,
				controllerName(clusterName, i),
				true,
			); err != nil {
				resp.Diagnostics.AddWarning("Failed to remove controller container", err.Error())
			}
		}
		if data.Network.ValueString() == "" {
			if err := docker.removeNetwork(ctx, networkName(clusterName)); err != nil {
				resp.Diagnostics.AddWarning("Failed to remove network", err.Error())
			}
		}
	}
}

func (r *ClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	id := req.ID
	docker := newDockerClient(r.binaryPath)

	running, err := docker.isRunning(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to inspect container during import", err.Error())
		return
	}

	if running {
		resp.State.SetAttribute(ctx, path.Root("id"), id)
		resp.State.SetAttribute(ctx, path.Root("name"), id)
		resp.State.SetAttribute(ctx, path.Root("single_node"), true)
		resp.State.SetAttribute(ctx, path.Root("controller_count"), int64(1))
		resp.State.SetAttribute(ctx, path.Root("worker_count"), int64(0))
		resp.State.SetAttribute(ctx, path.Root("wait_for_ready"), true)
		resp.State.SetAttribute(ctx, path.Root("readiness_timeout"), "120s")
		image, err := docker.inspectField(ctx, id, "{{.Config.Image}}")
		if err == nil {
			resp.State.SetAttribute(ctx, path.Root("image"), image)
			resp.State.SetAttribute(ctx, path.Root("version"), extractVersionFromImage(image))
		}
		return
	}

	ctrlName := controllerName(id, 1)
	running, err = docker.isRunning(ctx, ctrlName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to inspect controller during import", err.Error())
		return
	}
	if !running {
		resp.Diagnostics.AddError("Cluster not found",
			"No running container found matching the import ID: "+id)
		return
	}

	cc := 1
	for {
		if r, _ := docker.isRunning(ctx, controllerName(id, cc+1)); !r {
			break
		}
		cc++
	}

	wc := 0
	for {
		if r, _ := docker.isRunning(ctx, workerName(id, wc+1)); !r {
			break
		}
		wc++
	}

	resp.State.SetAttribute(ctx, path.Root("id"), id)
	resp.State.SetAttribute(ctx, path.Root("name"), id)
	resp.State.SetAttribute(ctx, path.Root("single_node"), false)
	resp.State.SetAttribute(ctx, path.Root("controller_count"), int64(cc))
	resp.State.SetAttribute(ctx, path.Root("worker_count"), int64(wc))
	resp.State.SetAttribute(ctx, path.Root("wait_for_ready"), true)
	resp.State.SetAttribute(ctx, path.Root("readiness_timeout"), "120s")
	image, err := docker.inspectField(ctx, ctrlName, "{{.Config.Image}}")
	if err == nil {
		resp.State.SetAttribute(ctx, path.Root("image"), image)
		resp.State.SetAttribute(ctx, path.Root("version"), extractVersionFromImage(image))
	}
}

// --- single-node -----------------------------------------------------------

func createSingleNode(
	ctx context.Context,
	docker *dockerClient,
	name, image string,
	waitForReady bool,
	readinessTimeout time.Duration,
	data *ClusterResourceModel,
	resp *resource.CreateResponse,
) {
	containerArgs := append(
		[]string{"k0s", "controller", "--enable-worker"},
		expandStringList(data.ExtraArgs)...,
	)

	ports := expandStringList(data.Ports)
	if ports == nil {
		ports = []string{"6443:6443"}
	}
	volumes := expandStringList(data.Volumes)
	if volumes == nil {
		volumes = []string{"/var/lib/k0s"}
	}
	tmpfs := expandStringList(data.Tmpfs)
	if tmpfs == nil {
		tmpfs = []string{"/run"}
	}
	env := expandStringMap(data.Env)
	cpus := data.Cpu.ValueString()
	memory := data.Memory.ValueString()
	network := data.Network.ValueString()

	_, err := docker.createContainer(ctx,
		name, name, image,
		true, ports, volumes, tmpfs, env,
		cpus, memory,
		network,
		containerArgs,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create container", err.Error())
		return
	}

	if err := docker.startContainer(ctx, name); err != nil {
		if rerr := docker.removeContainer(ctx, name, true); rerr != nil {
			resp.Diagnostics.AddWarning(
				"Failed to remove container after start failure",
				rerr.Error(),
			)
		}
		resp.Diagnostics.AddError("Failed to start container", err.Error())
		return
	}

	var kubeconfig string
	if waitForReady {
		kubeconfig, err = waitForReadiness(ctx, docker, name, readinessTimeout)
		if err != nil {
			if rerr := docker.removeContainer(ctx, name, true); rerr != nil {
				resp.Diagnostics.AddWarning(
					"Failed to remove container after readiness timeout",
					rerr.Error(),
				)
			}
			resp.Diagnostics.AddError("Cluster did not become ready", err.Error())
			return
		}
	} else {
		kubeconfig, _ = docker.exec(ctx, name, "k0s", "kubeconfig", "admin")
	}

	data.Id = types.StringValue(name)
	data.Image = types.StringValue(image)
	data.Kubeconfig = types.StringValue(kubeconfig)
}

// --- multi-node ------------------------------------------------------------

func createMultiNode(
	ctx context.Context,
	docker *dockerClient,
	clusterName, image string,
	waitForReady bool,
	readinessTimeout time.Duration,
	data *ClusterResourceModel,
	resp *resource.CreateResponse,
) {
	cpus := data.Cpu.ValueString()
	memory := data.Memory.ValueString()
	network := data.Network.ValueString()

	netName := network
	if netName == "" {
		netName = networkName(clusterName)
		if _, err := docker.createNetwork(ctx, netName); err != nil {
			resp.Diagnostics.AddError("Failed to create network", err.Error())
			return
		}
	}
	removeNet := network == ""
	defer func() {
		if removeNet {
			if err := docker.removeNetwork(ctx, netName); err != nil {
				resp.Diagnostics.AddWarning("Failed to remove network", err.Error())
			}
		}
	}()

	cc := int(data.ControllerCount.ValueInt64())
	wc := int(data.WorkerCount.ValueInt64())

	userPorts := expandStringList(data.Ports)
	volumes := expandStringList(data.Volumes)
	if volumes == nil {
		volumes = []string{"/var/lib/k0s"}
	}
	tmpfs := expandStringList(data.Tmpfs)
	if tmpfs == nil {
		tmpfs = []string{"/run"}
	}
	env := expandStringMap(data.Env)
	extraArgs := expandStringList(data.ExtraArgs)

	// --- create controllers ---
	for i := 1; i <= cc; i++ {
		cName := controllerName(clusterName, i)
		var ports []string
		if userPorts != nil {
			ports = userPorts
		} else {
			ports = []string{fmt.Sprintf("%d:6443", 6443+i-1)}
			if i == 1 {
				ports = append(ports, fmt.Sprintf("%d:9443", 9443+i-1))
			}
		}

		ctrlArgs := append([]string{"k0s", "controller"}, extraArgs...)
		_, err := docker.createContainer(ctx,
			cName, cName, image,
			true, ports, volumes, tmpfs, env,
			cpus, memory,
			netName,
			ctrlArgs,
		)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %s", cName), err.Error())
			return
		}

		if err := docker.startContainer(ctx, cName); err != nil {
			if rerr := docker.removeContainer(ctx, cName, true); rerr != nil {
				resp.Diagnostics.AddWarning(
					fmt.Sprintf("Failed to remove %s after start failure", cName),
					rerr.Error(),
				)
			}
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to start %s", cName), err.Error())
			return
		}
	}

	firstController := controllerName(clusterName, 1)

	var kubeconfig string
	var err error

	if waitForReady {
		kubeconfig, err = waitForReadiness(ctx, docker, firstController, readinessTimeout)
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
			workerArgs := append(
				[]string{"k0s", "worker", "--token-file", tokenPath},
				extraArgs...,
			)

			workerVolumes := append(append([]string{}, volumes...), tokenPath+":"+tokenPath)

			_, err := docker.createContainer(ctx,
				wName, wName, image,
				true, userPorts, workerVolumes, tmpfs, env,
				cpus, memory,
				netName,
				workerArgs,
			)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %s", wName), err.Error())
				return
			}

			if err := docker.startContainer(ctx, wName); err != nil {
				if rerr := docker.removeContainer(ctx, wName, true); rerr != nil {
					resp.Diagnostics.AddWarning(
						fmt.Sprintf("Failed to remove %s after start failure", wName),
						rerr.Error(),
					)
				}
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to start %s", wName), err.Error())
				return
			}
		}
	}

	removeNet = false
	data.Id = types.StringValue(clusterName)
	data.Image = types.StringValue(image)
	data.Kubeconfig = types.StringValue(kubeconfig)
}

// --- kubeconfig helpers ---------------------------------------------------

func setKubeconfigOutputs(data *ClusterResourceModel, diags *diag.Diagnostics) {
	ep := ""
	ca := ""
	cert := ""
	key := ""

	kcfg, err := parseKubeconfig(data.Kubeconfig.ValueString())
	if err == nil {
		ep = kcfg.Endpoint
		ca = kcfg.ClusterCACertificate
		cert = kcfg.ClientCertificate
		key = kcfg.ClientKey
	} else {
		diags.AddWarning("Could not parse kubeconfig", err.Error())
	}

	data.Endpoint = types.StringValue(ep)
	data.ClusterCACertificate = types.StringValue(ca)
	data.ClientCertificate = types.StringValue(cert)
	data.ClientKey = types.StringValue(key)
}

// --- readiness -------------------------------------------------------------

func waitForReadiness(
	ctx context.Context,
	docker *dockerClient,
	name string,
	timeout time.Duration,
) (string, error) {
	deadline := time.Now().Add(timeout)
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
		timeout,
	)
}

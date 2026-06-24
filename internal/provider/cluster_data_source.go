package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ClusterDataSource{}

func NewClusterDataSource() datasource.DataSource {
	return &ClusterDataSource{}
}

type ClusterDataSource struct {
	binaryPath string
}

type ClusterDataSourceModel struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Version              types.String `tfsdk:"version"`
	Image                types.String `tfsdk:"image"`
	Kubeconfig           types.String `tfsdk:"kubeconfig"`
	Status               types.String `tfsdk:"status"`
	SingleNode           types.Bool   `tfsdk:"single_node"`
	Endpoint             types.String `tfsdk:"endpoint"`
	ClientCertificate    types.String `tfsdk:"client_certificate"`
	ClientKey            types.String `tfsdk:"client_key"`
	ClusterCACertificate types.String `tfsdk:"cluster_ca_certificate"`
}

func (d *ClusterDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *ClusterDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read an existing k0s cluster by container name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Docker container name / unique identifier.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Docker container name of the cluster.",
				Required:            true,
			},
			"version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "k0s version running in the cluster.",
			},
			"image": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OCI image reference used by the container.",
			},
			"kubeconfig": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Kubeconfig contents for accessing the cluster.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Container status (running, exited, etc.).",
			},
			"single_node": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the cluster was created as single-node.",
			},
			"endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Kubernetes API server endpoint.",
			},
			"client_certificate": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Client certificate for authenticating to the cluster.",
			},
			"client_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Client key for authenticating to the cluster.",
			},
			"cluster_ca_certificate": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "CA certificate for verifying the API server.",
			},
		},
	}
}

func (d *ClusterDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	binaryPath, ok := req.ProviderData.(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected string, got: %T.", req.ProviderData),
		)
		return
	}
	d.binaryPath = binaryPath
}

func (d *ClusterDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data ClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	docker := newDockerClient(d.binaryPath)

	running, err := docker.isRunning(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to inspect container", err.Error())
		return
	}

	containerName := name
	singleNode := true

	if !running {
		ctrlName := controllerName(name, 1)
		running, err = docker.isRunning(ctx, ctrlName)
		if err != nil {
			resp.Diagnostics.AddError("Failed to inspect controller container", err.Error())
			return
		}
		if !running {
			resp.Diagnostics.AddError("Cluster not found or not running",
				fmt.Sprintf("No running container found for %q.", name))
			return
		}
		containerName = ctrlName
		singleNode = false
	}

	image, err := docker.inspectField(ctx, containerName, "{{.Config.Image}}")
	if err != nil {
		resp.Diagnostics.AddError("Could not read image", err.Error())
		return
	}
	status, err := docker.inspectField(ctx, containerName, "{{.State.Status}}")
	if err != nil {
		resp.Diagnostics.AddError("Could not read status", err.Error())
		return
	}

	kubeconfig, err := docker.exec(ctx, containerName, "k0s", "kubeconfig", "admin")
	if err != nil {
		resp.Diagnostics.AddError("Could not read kubeconfig", err.Error())
		return
	}

	data.Id = types.StringValue(name)
	data.Image = types.StringValue(image)
	data.Status = types.StringValue(status)
	data.Kubeconfig = types.StringValue(kubeconfig)
	data.Version = types.StringValue(extractVersionFromImage(image))
	data.SingleNode = types.BoolValue(singleNode)

	if kcfg, err := parseKubeconfig(kubeconfig); err == nil {
		data.Endpoint = types.StringValue(kcfg.Endpoint)
		data.ClientCertificate = types.StringValue(kcfg.ClientCertificate)
		data.ClientKey = types.StringValue(kcfg.ClientKey)
		data.ClusterCACertificate = types.StringValue(kcfg.ClusterCACertificate)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// extractVersionFromImage extracts the k0s version from the OCI image tag.
// e.g. "docker.io/k0sproject/k0s:v1.35.1-k0s.1" -> "v1.35.1-k0s.1"
func extractVersionFromImage(image string) string {
	if image == "" {
		return ""
	}
	for i := len(image) - 1; i >= 0; i-- {
		if image[i] == ':' {
			tag := image[i+1:]
			if tag == "" {
				return ""
			}
			return tag
		}
	}
	return image
}

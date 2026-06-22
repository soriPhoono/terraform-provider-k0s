package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure K0sProvider satisfies the provider interfaces.
var _ provider.Provider = &K0sProvider{}

// K0sProvider defines the provider implementation.
type K0sProvider struct {
	// version is set to the provider version on release, "dev" when built
	// locally, and "test" when running acceptance tests.
	version string
}

// K0sProviderModel describes the provider data model.
type K0sProviderModel struct {
	// BinaryPath is an optional path to the k0s or k0sctl executable.
	// If unset, the provider searches PATH.
	BinaryPath types.String `tfsdk:"binary_path"`
}

func (p *K0sProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "k0s"
	resp.Version = p.version
}

func (p *K0sProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"binary_path": schema.StringAttribute{
				MarkdownDescription: "Optional path to the k0s or k0sctl binary. If omitted, the provider searches PATH.",
				Optional:            true,
			},
		},
	}
}

func (p *K0sProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var data K0sProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provider-level configuration is passed to resources and data sources via
	// ResourceData and DataSourceData.
	resp.ResourceData = data.BinaryPath.ValueString()
	resp.DataSourceData = data.BinaryPath.ValueString()
}

func (p *K0sProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
	}
}

func (p *K0sProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// New returns a factory function used by the plugin framework to instantiate
// the provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &K0sProvider{
			version: version,
		}
	}
}

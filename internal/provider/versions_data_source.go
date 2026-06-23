package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const versionsGHURL = "https://api.github.com/repos/k0sproject/k0s/releases?per_page=10"
const versionsHTTPTimeout = 15 * time.Second

var _ datasource.DataSource = &VersionsDataSource{}

func NewVersionsDataSource() datasource.DataSource {
	return &VersionsDataSource{}
}

type VersionsDataSource struct{}

type VersionsDataSourceModel struct {
	Versions types.List   `tfsdk:"versions"`
	Latest   types.String `tfsdk:"latest"`
}

// ghRelease is a minimal representation of a GitHub release.
type ghRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

func (d *VersionsDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_versions"
}

func (d *VersionsDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query available k0s versions from GitHub releases.",
		Attributes: map[string]schema.Attribute{
			"versions": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of available k0s version strings (stable releases only).",
			},
			"latest": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Latest stable k0s version.",
			},
		},
	}
}

func (d *VersionsDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
}

func (d *VersionsDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data VersionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	versions, err := fetchK0sVersions(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch k0s versions",
			fmt.Sprintf("Could not retrieve k0s releases from GitHub: %s", err),
		)
		return
	}

	if len(versions) == 0 {
		resp.Diagnostics.AddError("No versions found", "No k0s releases were returned from GitHub.")
		return
	}

	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	versionValues := make([]types.String, len(versions))
	for i, v := range versions {
		versionValues[i] = types.StringValue(v)
	}

	listVal, diag := types.ListValueFrom(ctx, types.StringType, versionValues)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Versions = listVal
	data.Latest = types.StringValue(versions[0])

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func fetchK0sVersions(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionsGHURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: versionsHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var versions []string
	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		version := strings.TrimPrefix(r.TagName, "v")
		version = "v" + version
		versions = append(versions, version)
	}

	return versions, nil
}

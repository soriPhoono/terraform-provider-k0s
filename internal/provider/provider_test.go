package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderMetadata(t *testing.T) {
	p := New("test")()
	req := provider.MetadataRequest{}
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "k0s" {
		t.Errorf("expected type name 'k0s', got %q", resp.TypeName)
	}
	if resp.Version != "test" {
		t.Errorf("expected version 'test', got %q", resp.Version)
	}
}

func TestProviderVersion(t *testing.T) {
	p := New("1.0.0")()
	req := provider.MetadataRequest{}
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), req, &resp)

	if resp.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	p := New("test")()
	req := provider.SchemaRequest{}
	var resp provider.SchemaResponse
	p.Schema(context.Background(), req, &resp)

	attrs := resp.Schema.Attributes
	if len(attrs) == 0 {
		t.Fatal("expected at least one schema attribute")
	}

	if _, ok := attrs["binary_path"]; !ok {
		t.Error("expected binary_path attribute")
	}

	if att, ok := attrs["binary_path"].(schema.StringAttribute); ok {
		if !att.Optional {
			t.Error("expected binary_path to be optional")
		}
	} else {
		t.Error("expected binary_path to be a StringAttribute")
	}
}

func TestProviderConfigure(t *testing.T) {
	p := New("test")()

	// Build a config with only binary_path set.
	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"binary_path": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"binary_path": tftypes.NewValue(tftypes.String, "/custom/docker"),
	})

	var schemaResp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

	req := provider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw:    configVal,
			Schema: schemaResp.Schema,
		},
	}
	var resp provider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	if resp.ResourceData != "/custom/docker" {
		t.Errorf("expected resource data '/custom/docker', got %q", resp.ResourceData)
	}
	if resp.DataSourceData != "/custom/docker" {
		t.Errorf("expected data source data '/custom/docker', got %q", resp.DataSourceData)
	}
}

func TestProviderConfigureDefault(t *testing.T) {
	p := New("test")()

	// Build an empty config to test defaults.
	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"binary_path": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"binary_path": tftypes.NewValue(tftypes.String, ""),
	})

	var schemaResp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

	req := provider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw:    configVal,
			Schema: schemaResp.Schema,
		},
	}
	var resp provider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	// With an empty binary_path, ResourceData/DataSourceData should be empty string.
	if resp.ResourceData != "" {
		t.Errorf("expected empty resource data, got %q", resp.ResourceData)
	}
}

func TestProviderResources(t *testing.T) {
	p := New("test")()
	resources := p.Resources(context.Background())

	if len(resources) == 0 {
		t.Fatal("expected at least one resource")
	}

	found := false
	for _, r := range resources {
		res := r()
		req := resource.MetadataRequest{ProviderTypeName: "k0s"}
		var resp resource.MetadataResponse
		res.Metadata(context.Background(), req, &resp)
		if resp.TypeName == "k0s_cluster" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected k0s_cluster resource")
	}
}

func TestProviderDataSources(t *testing.T) {
	p := New("test")()
	datasources := p.DataSources(context.Background())

	if len(datasources) != 0 {
		t.Errorf("expected no data sources, got %d", len(datasources))
	}
}

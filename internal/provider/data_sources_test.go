package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestExtractVersionFromImage(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"docker.io/k0sproject/k0s:v1.35.1-k0s.1", "v1.35.1-k0s.1"},
		{"docker.io/k0sproject/k0s:v1.32.2-k0s.0", "v1.32.2-k0s.0"},
		{"k0sproject/k0s:v1.36.0-head", "v1.36.0-head"},
		{"", ""},
		{"no-tag", "no-tag"},
	}

	for _, tt := range tests {
		got := extractVersionFromImage(tt.image)
		if got != tt.want {
			t.Errorf("extractVersionFromImage(%q) = %q, want %q", tt.image, got, tt.want)
		}
	}
}

func TestClusterDataSourceMetadata(t *testing.T) {
	ds := NewClusterDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "k0s"}
	var resp datasource.MetadataResponse
	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "k0s_cluster" {
		t.Errorf("expected type name 'k0s_cluster', got %q", resp.TypeName)
	}
}

func TestClusterDataSourceSchema(t *testing.T) {
	ds := NewClusterDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), req, &resp)

	if len(resp.Schema.Attributes) == 0 {
		t.Fatal("expected at least one attribute")
	}

	expectedAttrs := []string{
		"id",
		"name",
		"version",
		"image",
		"kubeconfig",
		"status",
		"single_node",
	}
	for _, name := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}

	if att, ok := resp.Schema.Attributes["kubeconfig"].(schema.StringAttribute); ok {
		if !att.Sensitive {
			t.Error("expected kubeconfig to be sensitive")
		}
	} else {
		t.Error("expected kubeconfig to be a StringAttribute")
	}
}

func TestVersionsDataSourceMetadata(t *testing.T) {
	ds := NewVersionsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "k0s"}
	var resp datasource.MetadataResponse
	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "k0s_versions" {
		t.Errorf("expected type name 'k0s_versions', got %q", resp.TypeName)
	}
}

func TestVersionsDataSourceSchema(t *testing.T) {
	ds := NewVersionsDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), req, &resp)

	if len(resp.Schema.Attributes) == 0 {
		t.Fatal("expected at least one attribute")
	}

	expectedAttrs := []string{"versions", "latest"}
	for _, name := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q in schema", name)
		}
	}

	if att, ok := resp.Schema.Attributes["versions"].(schema.ListAttribute); ok {
		if att.ElementType.String() != "types.StringType" {
			t.Logf("versions element type: %s", att.ElementType.String())
		}
	} else {
		t.Error("expected versions to be a ListAttribute")
	}
}

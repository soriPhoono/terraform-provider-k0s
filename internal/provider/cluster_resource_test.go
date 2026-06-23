package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestImageForVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{
			version: "v1.35.1-k0s.1",
			want:    "docker.io/k0sproject/k0s:v1.35.1-k0s.1",
		},
		{
			version: "v1.32.2-k0s.0",
			want:    "docker.io/k0sproject/k0s:v1.32.2-k0s.0",
		},
		{
			version: "v1.31.0+k0s.0",
			want:    "docker.io/k0sproject/k0s:v1.31.0-k0s.0",
		},
		{
			version: "1.32.2+k0s.0",
			want:    "docker.io/k0sproject/k0s:v1.32.2-k0s.0",
		},
		{
			version: "v1.36.0-head",
			want:    "docker.io/k0sproject/k0s:v1.36.0-head",
		},
		{
			version: "",
			want:    "docker.io/k0sproject/k0s:v",
		},
	}

	for _, tt := range tests {
		got := imageForVersion(tt.version)
		if got != tt.want {
			t.Errorf("imageForVersion(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

func TestResourceMetadata(t *testing.T) {
	r := NewClusterResource()
	req := resource.MetadataRequest{ProviderTypeName: "k0s"}
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "k0s_cluster" {
		t.Errorf("expected type name 'k0s_cluster', got %q", resp.TypeName)
	}
}

func TestResourceSchema(t *testing.T) {
	r := NewClusterResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), req, &resp)

	attrs := resp.Schema.Attributes
	if len(attrs) == 0 {
		t.Fatal("expected at least one schema attribute")
	}

	expectedAttrs := []struct {
		name      string
		required  bool
		optional  bool
		computed  bool
		sensitive bool
	}{
		{name: "id", computed: true},
		{name: "name", required: true},
		{name: "version", optional: true, computed: true},
		{name: "image", optional: true, computed: true},
		{name: "kubeconfig", computed: true, sensitive: true},
		{name: "single_node", optional: true, computed: true},
		{name: "controller_count", optional: true, computed: true},
		{name: "worker_count", optional: true, computed: true},
	}

	for _, ea := range expectedAttrs {
		attr, ok := resp.Schema.Attributes[ea.name]
		if !ok {
			t.Errorf("expected attribute %q in schema", ea.name)
			continue
		}

		if attr.IsRequired() != ea.required {
			t.Errorf(
				"attribute %q IsRequired = %v, want %v",
				ea.name,
				attr.IsRequired(),
				ea.required,
			)
		}
		if attr.IsOptional() != ea.optional {
			t.Errorf(
				"attribute %q IsOptional = %v, want %v",
				ea.name,
				attr.IsOptional(),
				ea.optional,
			)
		}
		if attr.IsComputed() != ea.computed {
			t.Errorf(
				"attribute %q IsComputed = %v, want %v",
				ea.name,
				attr.IsComputed(),
				ea.computed,
			)
		}
		if attr.IsSensitive() != ea.sensitive {
			t.Errorf(
				"attribute %q IsSensitive = %v, want %v",
				ea.name,
				attr.IsSensitive(),
				ea.sensitive,
			)
		}
	}
}

func TestResourceSchemaAttributes(t *testing.T) {
	r := NewClusterResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), req, &resp)

	attrs := resp.Schema.Attributes
	if len(attrs) != 8 {
		t.Errorf("expected 8 attributes, got %d", len(attrs))
	}

	for name, attr := range attrs {
		switch attr := attr.(type) {
		case schema.StringAttribute:
			if name == "kubeconfig" && !attr.Sensitive {
				t.Errorf("expected kubeconfig to be sensitive")
			}
			if name == "name" && !attr.Required {
				t.Errorf("expected name to be required")
			}
		}
	}
}

func TestResourceTypeName(t *testing.T) {
	r := NewClusterResource()

	// Verify it implements the expected interfaces
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("expected resource to implement resource.ResourceWithImportState")
	}
}

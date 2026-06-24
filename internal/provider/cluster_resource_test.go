package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			want:    "",
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
		{name: "kubeconfig_path", optional: true},
		{name: "wait_for_ready", optional: true, computed: true},
		{name: "ports", optional: true},
		{name: "volumes", optional: true},
		{name: "tmpfs", optional: true},
		{name: "env", optional: true},
		{name: "extra_args", optional: true},
		{name: "cpu", optional: true},
		{name: "memory", optional: true},
		{name: "readiness_timeout", optional: true, computed: true},
		{name: "network", optional: true},
		{name: "timeouts", optional: true},
		{name: "endpoint", computed: true},
		{name: "client_certificate", computed: true, sensitive: true},
		{name: "client_key", computed: true, sensitive: true},
		{name: "cluster_ca_certificate", computed: true, sensitive: true},
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
	if len(attrs) != 24 {
		t.Errorf("expected 24 attributes, got %d", len(attrs))
	}

	for name, attr := range attrs {
		switch attr := attr.(type) {
		case schema.StringAttribute:
			if (name == "kubeconfig" || name == "client_certificate" || name == "client_key" || name == "cluster_ca_certificate") &&
				!attr.Sensitive {
				t.Errorf("expected %s to be sensitive", name)
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

// --- helper function tests ------------------------------------------------

func TestExpandStringList(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		got := expandStringList(types.ListNull(types.StringType))
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		got := expandStringList(types.ListUnknown(types.StringType))
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("empty", func(t *testing.T) {
		got := expandStringList(types.ListValueMust(types.StringType, []attr.Value{}))
		if got == nil || len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})
	t.Run("populated", func(t *testing.T) {
		got := expandStringList(types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("a"),
			types.StringValue("b"),
		}))
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("expected [a b], got %v", got)
		}
	})
}

func TestExpandStringMap(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		got := expandStringMap(types.MapNull(types.StringType))
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		got := expandStringMap(types.MapUnknown(types.StringType))
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("empty", func(t *testing.T) {
		got := expandStringMap(types.MapValueMust(types.StringType, map[string]attr.Value{}))
		if got == nil || len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
	t.Run("populated", func(t *testing.T) {
		got := expandStringMap(types.MapValueMust(types.StringType, map[string]attr.Value{
			"k1": types.StringValue("v1"),
			"k2": types.StringValue("v2"),
		}))
		if len(got) != 2 || got["k1"] != "v1" || got["k2"] != "v2" {
			t.Errorf("expected {k1:v1 k2:v2}, got %v", got)
		}
	})
}

func TestControllerName(t *testing.T) {
	if g := controllerName("mycluster", 1); g != "mycluster-controller-1" {
		t.Errorf("controllerName = %q, want %q", g, "mycluster-controller-1")
	}
	if g := controllerName("mycluster", 2); g != "mycluster-controller-2" {
		t.Errorf("controllerName = %q, want %q", g, "mycluster-controller-2")
	}
	if g := controllerName("c", 0); g != "c-controller-0" {
		t.Errorf("controllerName = %q, want %q", g, "c-controller-0")
	}
}

func TestWorkerName(t *testing.T) {
	if g := workerName("mycluster", 1); g != "mycluster-worker-1" {
		t.Errorf("workerName = %q, want %q", g, "mycluster-worker-1")
	}
	if g := workerName("mycluster", 2); g != "mycluster-worker-2" {
		t.Errorf("workerName = %q, want %q", g, "mycluster-worker-2")
	}
	if g := workerName("c", 0); g != "c-worker-0" {
		t.Errorf("workerName = %q, want %q", g, "c-worker-0")
	}
}

func TestNetworkName(t *testing.T) {
	if g := networkName("mycluster"); g != "k0s-mycluster" {
		t.Errorf("networkName = %q, want %q", g, "k0s-mycluster")
	}
	if g := networkName(""); g != "k0s-" {
		t.Errorf("networkName = %q, want %q", g, "k0s-")
	}
	if g := networkName("a.b-c"); g != "k0s-a.b-c" {
		t.Errorf("networkName = %q, want %q", g, "k0s-a.b-c")
	}
}

// --- validator tests ------------------------------------------------------

func TestDockerNameValidator(t *testing.T) {
	v := dockerNameValidator{}

	t.Run("valid names", func(t *testing.T) {
		names := []string{
			"mycluster",
			"my-cluster",
			"my.cluster",
			"my_cluster",
			"a",
			"a123",
			"123abc",
		}
		for _, name := range names {
			var resp validator.StringResponse
			req := validator.StringRequest{
				ConfigValue: types.StringValue(name),
			}
			v.ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for valid name %q", name)
			}
		}
	})

	t.Run("invalid names", func(t *testing.T) {
		names := []string{"", "-cluster", ".cluster", "cluster!", "cluster name"}
		for _, name := range names {
			var resp validator.StringResponse
			req := validator.StringRequest{
				ConfigValue: types.StringValue(name),
			}
			v.ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for invalid name %q", name)
			}
		}
	})

	t.Run("too long", func(t *testing.T) {
		name := ""
		for i := 0; i < 129; i++ {
			name += "a"
		}
		var resp validator.StringResponse
		req := validator.StringRequest{
			ConfigValue: types.StringValue(name),
		}
		v.ValidateString(context.Background(), req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for name >128 chars")
		}
	})

	t.Run("null", func(t *testing.T) {
		var resp validator.StringResponse
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
		}
		v.ValidateString(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Error("expected no error for null")
		}
	})

	t.Run("unknown", func(t *testing.T) {
		var resp validator.StringResponse
		req := validator.StringRequest{
			ConfigValue: types.StringUnknown(),
		}
		v.ValidateString(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Error("expected no error for unknown")
		}
	})
}

func TestAtLeastInt64Validator(t *testing.T) {
	t.Run("min 1 - at boundary", func(t *testing.T) {
		v := atLeastInt64Validator{min: 1}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Value(1)}
		v.ValidateInt64(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Error("expected no error for value 1 with min 1")
		}
	})
	t.Run("min 1 - below boundary", func(t *testing.T) {
		v := atLeastInt64Validator{min: 1}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Value(0)}
		v.ValidateInt64(context.Background(), req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for value 0 with min 1")
		}
	})
	t.Run("min 1 - negative", func(t *testing.T) {
		v := atLeastInt64Validator{min: 1}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Value(-5)}
		v.ValidateInt64(context.Background(), req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for value -5 with min 1")
		}
	})
	t.Run("min 0 - boundary", func(t *testing.T) {
		v := atLeastInt64Validator{min: 0}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Value(0)}
		v.ValidateInt64(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Error("expected no error for value 0 with min 0")
		}
	})
	t.Run("min 0 - negative", func(t *testing.T) {
		v := atLeastInt64Validator{min: 0}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Value(-1)}
		v.ValidateInt64(context.Background(), req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for value -1 with min 0")
		}
	})
	t.Run("null", func(t *testing.T) {
		v := atLeastInt64Validator{min: 1}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Null()}
		v.ValidateInt64(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Error("expected no error for null")
		}
	})
	t.Run("unknown", func(t *testing.T) {
		v := atLeastInt64Validator{min: 1}
		var resp validator.Int64Response
		req := validator.Int64Request{ConfigValue: types.Int64Unknown()}
		v.ValidateInt64(context.Background(), req, &resp)
		if resp.Diagnostics.HasError() {
			t.Error("expected no error for unknown")
		}
	})
}

// --- setKubeconfigOutputs tests -------------------------------------------

func TestSetKubeconfigOutputs_Valid(t *testing.T) {
	var data ClusterResourceModel
	data.Kubeconfig = types.StringValue(mustReadTestdata(t, "kubeconfig-valid.txt"))
	var d diag.Diagnostics
	setKubeconfigOutputs(&data, &d)
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}
	if d.WarningsCount() > 0 {
		t.Fatalf("unexpected warnings: %v", d.Warnings())
	}
	if data.Endpoint.ValueString() != "https://127.0.0.1:6443" {
		t.Errorf("Endpoint = %q", data.Endpoint.ValueString())
	}
	if data.ClusterCACertificate.ValueString() != "Y2FfZGF0YQ==" {
		t.Errorf("ClusterCACertificate = %q", data.ClusterCACertificate.ValueString())
	}
	if data.ClientCertificate.ValueString() != "Y2VydF9kYXRh" {
		t.Errorf("ClientCertificate = %q", data.ClientCertificate.ValueString())
	}
	if data.ClientKey.ValueString() != "a2V5X2RhdGE=" {
		t.Errorf("ClientKey = %q", data.ClientKey.ValueString())
	}
}

func TestSetKubeconfigOutputs_InvalidKubeconfig(t *testing.T) {
	var data ClusterResourceModel
	data.Kubeconfig = types.StringValue("this is not valid yaml {{{")
	var d diag.Diagnostics
	setKubeconfigOutputs(&data, &d)
	if d.WarningsCount() != 1 {
		t.Errorf("expected 1 warning, got %d", d.WarningsCount())
	}
	if data.Endpoint.ValueString() != "" {
		t.Errorf(
			"expected empty Endpoint on invalid kubeconfig, got %q",
			data.Endpoint.ValueString(),
		)
	}
}

func TestSetKubeconfigOutputs_EmptyKubeconfig(t *testing.T) {
	var data ClusterResourceModel
	data.Kubeconfig = types.StringValue("")
	var d diag.Diagnostics
	setKubeconfigOutputs(&data, &d)
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}
	if data.Endpoint.ValueString() != "" {
		t.Errorf(
			"expected empty Endpoint for empty kubeconfig, got %q",
			data.Endpoint.ValueString(),
		)
	}
}

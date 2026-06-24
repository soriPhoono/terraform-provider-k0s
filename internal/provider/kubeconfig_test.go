package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading testdata/%s: %v", name, err)
	}
	return string(data)
}

func TestParseKubeconfig_Valid(t *testing.T) {
	got, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-valid.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "https://127.0.0.1:6443" {
		t.Errorf("Endpoint = %q", got.Endpoint)
	}
	if got.ClusterCACertificate != "Y2FfZGF0YQ==" {
		t.Errorf("ClusterCACertificate = %q", got.ClusterCACertificate)
	}
	if got.ClientCertificate != "Y2VydF9kYXRh" {
		t.Errorf("ClientCertificate = %q", got.ClientCertificate)
	}
	if got.ClientKey != "a2V5X2RhdGE=" {
		t.Errorf("ClientKey = %q", got.ClientKey)
	}
}

func TestParseKubeconfig_WithExtras(t *testing.T) {
	got, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-with-extras.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "https://192.168.1.100:6443" {
		t.Errorf("Endpoint = %q", got.Endpoint)
	}
	if got.ClientKey != "key-base64" {
		t.Errorf("ClientKey = %q", got.ClientKey)
	}
}

func TestParseKubeconfig_EmptyString(t *testing.T) {
	got, err := parseKubeconfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" || got.ClusterCACertificate != "" || got.ClientCertificate != "" ||
		got.ClientKey != "" {
		t.Error("expected all empty fields")
	}
}

func TestParseKubeconfig_EmptyObject(t *testing.T) {
	got, err := parseKubeconfig("{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" || got.ClusterCACertificate != "" || got.ClientCertificate != "" ||
		got.ClientKey != "" {
		t.Error("expected all empty fields")
	}
}

func TestParseKubeconfig_NullYAML(t *testing.T) {
	for _, input := range []string{"null", "~", ""} {
		got, err := parseKubeconfig(input)
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", input, err)
		}
		if got.Endpoint != "" || got.ClusterCACertificate != "" || got.ClientCertificate != "" ||
			got.ClientKey != "" {
			t.Errorf("expected all empty fields for input %q", input)
		}
	}
}

func TestParseKubeconfig_EmptyClustersUsers(t *testing.T) {
	got, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-empty-clusters-users.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" || got.ClusterCACertificate != "" || got.ClientCertificate != "" ||
		got.ClientKey != "" {
		t.Error("expected all empty fields")
	}
}

func TestParseKubeconfig_MissingClusters(t *testing.T) {
	got, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-missing-clusters.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" {
		t.Errorf("expected empty Endpoint, got %q", got.Endpoint)
	}
	if got.ClientCertificate != "cert" {
		t.Errorf("ClientCertificate = %q", got.ClientCertificate)
	}
}

func TestParseKubeconfig_MissingUsers(t *testing.T) {
	got, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-missing-users.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "https://x:6443" {
		t.Errorf("Endpoint = %q", got.Endpoint)
	}
	if got.ClientCertificate != "" {
		t.Errorf("expected empty ClientCertificate, got %q", got.ClientCertificate)
	}
}

func TestParseKubeconfig_FirstClusterEmpty(t *testing.T) {
	got, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-first-empty.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" {
		t.Errorf("expected empty Endpoint, got %q", got.Endpoint)
	}
	if got.ClusterCACertificate != "" {
		t.Error("expected empty CA from empty first cluster")
	}
	if got.ClientCertificate != "" {
		t.Error("expected empty client cert from empty first user")
	}
}

func TestParseKubeconfig_MalformedYAML(t *testing.T) {
	inputs := []string{
		"this is not yaml ::: {{{",
		mustReadTestdata(t, "kubeconfig-unclosed-quote.txt"),
	}
	for _, input := range inputs {
		_, err := parseKubeconfig(input)
		if err == nil {
			t.Errorf("expected error for malformed input")
		}
	}
}

func TestParseKubeconfig_TypeMismatch(t *testing.T) {
	_, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-type-clusters.txt"))
	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

func TestParseKubeconfig_ScalarYAML(t *testing.T) {
	_, err := parseKubeconfig(mustReadTestdata(t, "kubeconfig-scalar.txt"))
	if err == nil {
		t.Error("expected error for scalar YAML")
	}
}

func TestParseKubeconfig_EmptyFields(t *testing.T) {
	input := "clusters:\n- cluster:\n    server: \"\"\n    certificate-authority-data: \"\"\nusers:\n- user:\n    client-certificate-data: \"\"\n    client-key-data: \"\"\n"
	got, err := parseKubeconfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" || got.ClusterCACertificate != "" || got.ClientCertificate != "" ||
		got.ClientKey != "" {
		t.Error("expected all empty fields")
	}
}

func TestParseKubeconfig_MissingSubKeys(t *testing.T) {
	input := "clusters:\n- name: x\nusers:\n- name: admin\n"
	got, err := parseKubeconfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "" || got.ClusterCACertificate != "" || got.ClientCertificate != "" ||
		got.ClientKey != "" {
		t.Error("expected all empty fields")
	}
}

func TestParseKubeconfig_Unicode(t *testing.T) {
	input := "clusters:\n- cluster:\n    server: https://møøse:6443\nusers:\n- user:\n    client-certificate-data: \xc3\xa9\n"
	got, err := parseKubeconfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Endpoint != "https://møøse:6443" {
		t.Errorf("Endpoint = %q", got.Endpoint)
	}
}

func TestParseKubeconfig_LongStrings(t *testing.T) {
	longCA := strings.Repeat("ABCDEFGH", 1000)
	input := "clusters:\n- cluster:\n    server: https://x:6443\n    certificate-authority-data: " + longCA + "\n"
	got, err := parseKubeconfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ClusterCACertificate != longCA {
		t.Error("long CA string not preserved")
	}
}

func TestWriteKubeconfigFile_Normal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig.txt")
	kubeconfig := "clusters:\n- cluster:\n    server: https://x:6443\n"

	if err := writeKubeconfigFile(path, kubeconfig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != kubeconfig {
		t.Errorf("file content = %q, want %q", string(data), kubeconfig)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat written file: %v", err)
	}
	if info.Mode() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode())
	}
}

func TestWriteKubeconfigFile_DeepDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "kubeconfig.txt")

	if err := writeKubeconfigFile(path, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file was not created at nested path")
	}
}

func TestWriteKubeconfigFile_BareFilename(t *testing.T) {
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := writeKubeconfigFile("bare.txt", "content"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat("bare.txt"); os.IsNotExist(err) {
		t.Error("file was not created in CWD")
	}
}

func TestWriteKubeconfigFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	if err := writeKubeconfigFile(path, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestWriteKubeconfigFile_ReadOnlyDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	dir := t.TempDir()
	if err := os.Chmod(dir, 0444); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "kubeconfig.txt")

	err := writeKubeconfigFile(path, "test")
	if err == nil {
		t.Error("expected error when writing to read-only directory")
	}
}

func TestFsDir_Normal(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/tmp/a/b/c/kubeconfig.txt", "/tmp/a/b/c"},
		{"./config/file", "./config"},
		{"../config/file", "../config"},
		{"/kubeconfig.txt", ""},
		{"/", ""},
		{"/foo//bar/file", "/foo//bar"},
		{"bare-filename", "."},
		{"", "."},
	}
	for _, tt := range tests {
		got := fsDir(tt.path)
		if got != tt.want {
			t.Errorf("fsDir(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

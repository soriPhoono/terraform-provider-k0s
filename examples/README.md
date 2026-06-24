# Examples

This directory contains example Terraform configurations for the k0s provider.

- `provider/` - Provider configuration example.
- `resources/k0s_cluster/`
  - `resource.tf` - Single-node cluster with kubeconfig output.
  - `multi-node.tf` - Multi-node cluster with 1 controller and 2 workers.
  - `kubeconfig-path.tf` - Cluster with kubeconfig written to a local file.
  - `import.sh` - Import existing cluster by container name.
- `data-sources/k0s_cluster/` - Read existing cluster attributes via data source.
- `data-sources/k0s_versions/` - Query available k0s releases from GitHub.

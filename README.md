# Terraform Provider: k0s

[![Registry](https://img.shields.io/badge/registry-soriphoono%2Fk0s-blue)](https://registry.terraform.io/providers/soriphoono/k0s)
[![License](https://img.shields.io/badge/license-MPL--2.0-blue)](LICENSE)

A [Terraform](https://www.terraform.io) provider for provisioning local [k0s](https://k0sproject.io) Kubernetes testing clusters via Docker.

Create single-node or multi-node k0s clusters as Docker containers for development, testing, and CI.

## Quick Start

```hcl
terraform {
  required_providers {
    k0s = {
      source = "soriphoono/k0s"
    }
  }
}

provider "k0s" {}

resource "k0s_cluster" "example" {
  name        = "my-cluster"
  version     = "v1.32.2-k0s.0"
  single_node = true
}

output "kubeconfig" {
  value     = k0s_cluster.example.kubeconfig
  sensitive = true
}
```

```bash
terraform init
terraform apply
# Cluster is ready — kubectl is already configured in the kubeconfig
kubectl --kubeconfig <(echo "$(terraform output -raw kubeconfig)") get nodes
```

## Requirements

- [Docker](https://docs.docker.com/engine/install/) Engine 24+ (with permission to access `/var/run/docker.sock`)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0+

## Documentation

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/soriphoono/k0s/latest/docs).

### Resources

| Name | Description |
|---|---|
| [`k0s_cluster`](docs/resources/cluster.md) | Create and manage k0s clusters in Docker |

### Data Sources

| Name | Description |
|---|---|
| [`k0s_cluster`](docs/data-sources/cluster.md) | Read an existing cluster by container name |
| [`k0s_versions`](docs/data-sources/versions.md) | List available k0s versions from GitHub |

## Cluster Modes

### Single-node (default)

One privileged container runs both the k0s controller and worker workloads. Minimal resource usage, ideal for quick testing.

```hcl
resource "k0s_cluster" "single" {
  name        = "dev-cluster"
  single_node = true
}
```

### Multi-node

Creates a dedicated Docker bridge network with separate controller and worker containers. Workers join via auto-generated tokens.

```hcl
resource "k0s_cluster" "multi" {
  name             = "multi-cluster"
  single_node      = false
  controller_count = 1
  worker_count     = 2
}
```

## Development

See [AGENTS.md](AGENTS.md) for detailed development instructions. Quick reference:

```bash
# Enter the dev shell
nix develop .          # or: direnv allow

# Build
CGO_ENABLED=0 go build -o terraform-provider-k0s

# Unit tests
CGO_ENABLED=0 go test ./internal/provider/... -count=1

# Acceptance tests (requires Docker)
TF_ACC=1 CGO_ENABLED=0 go test ./internal/provider/... -count=1 -timeout 20m
```

The repository uses a Nix flake for development. Run `nix develop .` to get a shell with Go, Terraform, k0sctl, kubectl, and all development tools.

## License

Mozilla Public License 2.0

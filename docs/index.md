______________________________________________________________________

## page_title: "k0s Provider" subcategory: "" description: |- Provision and manage local k0s Kubernetes testing clusters via Docker.

# k0s Provider

The k0s provider lets you create, manage, and destroy [k0s](https://k0sproject.io)
Kubernetes testing clusters using Docker. Clusters run as privileged containers
on the local Docker daemon and can be single-node (controller + worker) or
multi-node (separate controllers and workers on a bridge network).

Use the navigation to the left to read about the available resources and data
sources.

## Requirements

- [Docker](https://docs.docker.com/engine/install/) (Engine 24+)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0+

## Example Usage

```terraform
terraform {
  required_providers {
    k0s = {
      source = "soriphoono/k0s"
    }
  }
}

provider "k0s" {
  # Optional: override the path to the k0s/k0sctl binary.
  # binary_path = "/usr/local/bin/k0sctl"
}
```

## Getting the kubeconfig

The `k0s_cluster` resource and data source both expose a `kubeconfig` attribute
that contains the full kubeconfig for accessing the cluster:

```hcl
resource "k0s_cluster" "example" {
  name       = "my-cluster"
  single_node = true
}

output "kubeconfig" {
  value     = k0s_cluster.example.kubeconfig
  sensitive = true
}
```

## Cluster Lifecycle

- **Single-node**: One container runs both controller and worker workloads.
  Creation ~3-5 seconds after image pull.
- **Multi-node**: A dedicated Docker bridge network is created. Controllers
  run `k0s controller`, workers join via `k0s token create`. Networks and all
  containers are removed on destroy.
- **Import**: Existing clusters can be imported by container name.

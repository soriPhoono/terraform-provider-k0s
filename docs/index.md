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
  Single-node: `terraform import k0s_cluster.example <container-name>`.
  Multi-node: `terraform import k0s_cluster.example <cluster-name>` (the provider
  detects controllers and workers automatically).
- **Customization**: Ports, volumes, tmpfs, environment variables, extra CLI
  arguments, CPU, and memory limits are all configurable per container.
- **Timeouts**: A `timeouts` block supports `create`, `read`, and `delete`
  deadlines. The `readiness_timeout` attribute controls how long to wait for
  the cluster control plane to become ready.

## Troubleshooting

### Port conflicts

If you see `port is already allocated`, another cluster or service is using
the default port 6443. Set a custom `ports` attribute to avoid conflicts:

```hcl
resource "k0s_cluster" "example" {
  name    = "my-cluster"
  ports   = ["6444:6443"]
}
```

### Cluster does not become ready

Check Docker container logs for details. The `wait_for_ready` attribute
can be set to `false` to return immediately without polling:

```hcl
resource "k0s_cluster" "example" {
  name           = "my-cluster"
  wait_for_ready = false
}
```

Container logs are included in readiness timeout errors automatically.

### Docker daemon access

The provider communicates with the Docker daemon via the CLI. Ensure:

- `docker` is on your PATH
- Your user has permission to access `/var/run/docker.sock`
- Run `docker info` to verify

### Cleanup after failed destroy

If `terraform destroy` fails mid-operation, remove leftover resources manually:

```shell
docker container ls -a --filter "name=k0s-" -q | xargs docker rm -f
docker network ls --filter "name=k0s-" -q | xargs docker network rm
```

### Limitations

- **Update is not supported**. All mutable attributes use `RequiresReplace`,
  so any change triggers destroy + recreate.
- **Single-node port 6443** is the default host mapping. Create multiple
  clusters with different `ports` or one at a time.
- **`k0s_versions` data source** fetches releases from the GitHub API.
  The number of results is controlled by `per_page` (default 10).
  API rate limits apply to unauthenticated requests.

# AGENTS.md

## Project Overview

Terraform provider for provisioning local [k0s](https://k0sproject.io) Kubernetes testing clusters via Docker.

- **Provider address**: `registry.terraform.io/soriphoono/k0s`
- **Protocol**: Plugin Framework (protocol 6.0)
- **Language**: Go 1.25+
- **Backend**: Docker CLI (`os/exec` wrapper in `internal/provider/docker.go`)
- **Build system**: Nix flake (flake-parts) with direnv

### Architecture

```
main.go                          # Entry point, goreleaser version injection
internal/provider/
├── provider.go                  # K0sProvider — schema, configure, resource/data source registration
├── cluster_resource.go          # k0s_cluster resource — single-node & multi-node CRUD
├── cluster_data_source.go       # k0s_cluster data source — read existing cluster
├── versions_data_source.go      # k0s_versions data source — fetch releases from GitHub API
├── docker.go                    # dockerClient wrapper (exec, inspect, create, network, etc.)
├── kubeconfig.go                # kubeconfig YAML parser, file writer
├── *_test.go                    # Unit + acceptance tests
```

### Resources & Data Sources

| Type | Name | Description |
|---|---|---|
| Resource | `k0s_cluster` | Create/manage k0s clusters in Docker. Single-node (one container) or multi-node (N controllers + M workers on a bridge network). |
| Data source | `k0s_cluster` | Read an existing cluster by container name. Returns status, kubeconfig, version. |
| Data source | `k0s_versions` | Fetch stable k0s releases from `api.github.com/repos/k0sproject/k0s/releases`. |

## Setup Commands

```bash
# Enter the dev shell (requires direnv or nix)
nix develop .    # or: direnv allow

# Build the provider binary
CGO_ENABLED=0 go build -o terraform-provider-k0s

# Local dev override (so Terraform finds the local build)
# .terraformrc already configured in the repo:
provider_installation {
  dev_overrides { "registry.terraform.io/soriphoono/k0s" = "/path/to/repo" }
  direct {}
}
# Use: TF_CLI_CONFIG_FILE=$PWD/.terraformrc terraform init
```

## Development Workflow

### Code structure conventions

- **Provider config**: `internal/provider/provider.go` — `K0sProvider` with `Metadata`, `Schema`, `Configure`, `Resources`, `DataSources`.
- **Resources**: One file per resource in `internal/provider/`. Follow the `*Resource` struct + `*ResourceModel` pattern. Must implement `resource.Resource` and optionally `resource.ResourceWithImportState`.
- **Data sources**: One file per data source in `internal/provider/`. Follow the `*DataSource` struct + `*DataSourceModel` pattern.
- **Docker interactions**: Add methods to `dockerClient` in `docker.go` (e.g., `isRunning`, `createContainer`, `exec`, `inspectField`, `createNetwork`).
- **Kubeconfig parsing**: `kubeconfig.go` — parse kubeconfig YAML to extract endpoint, TLS certs.

### Adding a new attribute to an existing resource

1. Add the field to the `*ResourceModel` struct with the correct `tfsdk:"name"` tag.
1. Add an entry in the `Schema()` method with the correct attribute type and `MarkdownDescription`.
1. If it's an input, handle it in `Create()` / `Read()` / `Update()`.
1. If it's a computed output, populate it after the kubeconfig is available.
1. Run `make generate` to regenerate docs.
1. Add/update tests in `*_test.go`.

### Formatting and linting

```bash
# Format all files
nix fmt                    # treefmt: alejandra, gofmt, terraform fmt, yamlfmt, etc.

# Lint
golangci-lint run          # Go linters
```

Pre-commit hooks run automatically on `git commit` (via `git-hooks.nix`). The hooks include `gofmt`, `govet`, `golangci-lint`, `terraform-format`, `gitleaks`, `typos`, `actionlint`, `editorconfig-checker`, and file quality checks.

## Testing Instructions

### Unit tests (fast, no Docker required)

```bash
CGO_ENABLED=0 go test ./internal/provider/... -count=1
```

Covers: `imageForVersion`, `extractVersionFromImage`, provider schema/metadata/configure, resource/data source metadata/schema, interface implementations.

### Acceptance tests (requires Docker, real Terraform apply/destroy)

```bash
TF_ACC=1 CGO_ENABLED=0 go test ./internal/provider/... -count=1 -timeout 20m
```

| Test | What it does |
|---|---|
| `TestAccClusterResource_SingleNode` | Creates single-node cluster, verifies attributes, destroys |
| `TestAccClusterResource_MultiNode` | Creates 1 controller + 1 worker, verifies nodes, destroys |
| `TestAccClusterDataSource_SingleNode` | Creates cluster, reads via data source, checks status=running |
| `TestAccClusterResource_Import` | Creates cluster, imports by container name, verifies state |

### Test file naming

- `*_test.go` — unit tests (no `TF_ACC` needed)
- `acc_test.go` — acceptance tests (require `TF_ACC=1`)
- Test functions starting with `TestAcc` are acceptance tests
- Test functions starting with `Test` are unit tests

## Build and Deployment

### Local builds

```bash
make build          # CGO_ENABLED=0 go build -o terraform-provider-k0s
make test           # CGO_ENABLED=0 go test ./...
make testacc        # TF_ACC=1 acceptance tests
make generate       # go generate .  (regenerate docs/)
make lint           # golangci-lint run
```

### Releases (goreleaser)

```bash
# Dry-run to verify config
make release-dry-run

# Actual release
git tag v0.1.0 && git push origin v0.1.0
# GitHub Actions release.yml handles the rest
```

The `.goreleaser.yaml` cross-compiles for: `linux/darwin/windows/freebsd` × `amd64/386/arm/arm64`. Artifacts are zip archives with SHA256 checksums signed by GPG.

`terraform-registry-manifest.json` declares protocol 6.0.

### GitHub Actions

- `ci.yml` — runs on push/PR to `main`: `go build`, `go vet`, unit tests.
- `release.yml` — runs on `v*` tags: imports GPG key, runs goreleaser.

## Publishing to Terraform Registry

1. Tag a release: `git tag v0.1.0 && git push origin v0.1.0`
1. Release workflow creates a signed GitHub release with all platform binaries.
1. Go to https://registry.terraform.io/publish, select `soriphoono/k0s`.
1. Add GPG public key (the one corresponding to the private key in `GPG_PRIVATE_KEY` secret).
1. Publish.

## Documentation

Docs are auto-generated from schema descriptions + example files via `tfplugindocs`:

```bash
make generate
# Output in docs/
```

- `templates/index.md.tmpl` — custom provider landing page
- `examples/resources/k0s_cluster/` — example `.tf` and `import.sh`
- `examples/data-sources/` — example `.tf` for each data source

After regeneration, the rendered markdown in `docs/` is committed to the repo (required for Terraform Registry).

## Code Style

- **Go**: `gofmt` + `goimports` (enforced by treefmt). Use tabs for indentation (`.editorconfig`).
- **Terraform HCL**: `terraform fmt` style (enforced by treefmt).
- **Nix**: `alejandra` (enforced by treefmt).
- **YAML**: `yamlfmt` (enforced by treefmt).
- **Markdown**: `mdformat` (enforced by treefmt).
- Imports are organized by `goimports` automatically.
- Use `CGO_ENABLED=0` for all builds to avoid cgo dependencies.
- Keep `MarkdownDescription` on every schema attribute — these feed into generated docs.

## Pull Request Guidelines

- Title format: `<type>: <description>` matching conventional commits (`feat:`, `fix:`, `build:`, `test:`, `docs:`, `refactor:`).
- All pre-commit hooks must pass.
- Unit tests must pass. Acceptance tests pass in CI (with Docker).
- Run `nix flake check` to verify treefmt and pre-commit pass.
- Regenerate docs if schemas change: `make generate`.

## Security Considerations

- Do not commit GPG private keys or passphrases — set them as GitHub Actions secrets (`GPG_PRIVATE_KEY`, `PASSPHRASE`).
- The provider runs Docker containers with `--privileged` (required by k0s kubelet). Users should be aware of this.
- Kubeconfig and TLS credentials are marked `Sensitive: true` in the schema and should not be logged.

## Troubleshooting

- **Build fails with `gcc not found`**: Set `CGO_ENABLED=0` (already set in Makefile and CI).
- **Pre-commit hooks fail on commit**: Check `nix flake check` output. Common issues: formatting (run `nix fmt`), `golangci-lint` (run `golangci-lint run`).
- **Acceptance tests fail**: Ensure Docker is running and you have permission to access `/var/run/docker.sock`. The `docker info` command must succeed.
- **`go generate .` fails**: Make sure you're in the Nix dev shell (`nix develop .`) so `terraform` is on PATH.

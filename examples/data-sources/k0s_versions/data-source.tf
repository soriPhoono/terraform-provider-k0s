data "k0s_versions" "all" {}

output "latest" {
  value = data.k0s_versions.all.latest
}

output "versions" {
  value = data.k0s_versions.all.versions
}

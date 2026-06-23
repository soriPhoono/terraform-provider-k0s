resource "k0s_cluster" "example" {
  name        = "my-cluster"
  version     = "v1.32.2-k0s.0"
  single_node = true
}

data "k0s_cluster" "example" {
  name = k0s_cluster.example.name
}

output "status" {
  value = data.k0s_cluster.example.status
}

output "kubeconfig" {
  value     = data.k0s_cluster.example.kubeconfig
  sensitive = true
}

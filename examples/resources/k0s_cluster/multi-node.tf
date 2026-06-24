resource "k0s_cluster" "example" {
  name             = "example"
  version          = "v1.32.2-k0s.0"
  single_node      = false
  controller_count = 1
  worker_count     = 2
}

output "kubeconfig" {
  value     = k0s_cluster.example.kubeconfig
  sensitive = true
}

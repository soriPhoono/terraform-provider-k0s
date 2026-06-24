resource "k0s_cluster" "example" {
  name            = "example"
  version         = "v1.32.2-k0s.0"
  single_node     = true
  kubeconfig_path = "./kubeconfig/example.yaml"
}

# The kubeconfig is written to disk, ready for use with kubectl:
# $ export KUBECONFIG=./kubeconfig/example.yaml
# $ kubectl get nodes

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

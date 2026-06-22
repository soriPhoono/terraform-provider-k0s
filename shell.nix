{
  pkgs,
  config,
  ...
}:
with pkgs;
  mkShell {
    packages = [
      gh

      # Nix
      nixd
      nil
      alejandra

      # Generic
      gnumake

      # Go
      go
      gopls
      delve

      # Terraform
      terraform

      # Kubernetes
      k0sctl
      kubectl
    ];

    shellHook = ''
      source ${config.agenix-shell.installationScript}/bin/install-agenix-shell

      ${config.pre-commit.shellHook}
    '';
  }

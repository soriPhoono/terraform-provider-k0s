_: {
  projectRootFile = "flake.nix";

  programs = {
    # Nix
    alejandra.enable = true;
    deadnix.enable = true;
    statix.enable = true;

    # Go
    gofmt.enable = true;
    goimports.enable = true;
    golines.enable = true;

    # Terraform / HCL
    terraform.enable = true;

    # Data / config
    jsonfmt.enable = true;
    taplo.enable = true;
    yamlfmt.enable = true;

    # Documentation
    mdformat.enable = true;
  };

  settings = {
    global.excludes = [
      "*.lock"
      "go.mod"
      "go.sum"
      ".pre-commit-config.yaml"
      "terraform-provider-k0s"
      ".direnv/"
    ];
  };
}

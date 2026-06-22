{pkgs, ...}: {
  # Go hooks need network access to fetch modules; only run them in the
  # dev shell, not as a flake check.
  check.enable = false;

  settings = {
    excludes = [
      "flake\\.lock"
      "go\\.sum"
      "go\\.mod"
      "\\.pre-commit-config\\.yaml"
      "terraform-provider-k0s"
      "\\.direnv/"
    ];

    hooks = {
      # Nix
      nil.enable = true;
      treefmt.enable = true;

      # Go
      gofmt.enable = true;

      govet = {
        enable = true;
        entry = "${pkgs.bash}/bin/bash -c 'CGO_ENABLED=0 go vet ./...'";
        pass_filenames = false;
        extraPackages = [pkgs.go];
      };

      golangci-lint = {
        enable = true;
        entry = "${pkgs.bash}/bin/bash -c 'CGO_ENABLED=0 ${pkgs.golangci-lint}/bin/golangci-lint run'";
        pass_filenames = false;
        extraPackages = [pkgs.go];
      };

      # Terraform
      terraform-format.enable = true;

      # GitHub Actions
      actionlint.enable = true;

      # General file quality
      check-added-large-files.enable = true;
      check-merge-conflicts.enable = true;
      end-of-file-fixer.enable = true;
      trim-trailing-whitespace.enable = true;
      editorconfig-checker.enable = true;

      # Syntax checks
      check-json.enable = true;
      check-toml.enable = true;
      check-yaml.enable = true;

      # Spelling
      typos = {
        enable = true;
        pass_filenames = false;
      };

      # Secrets
      gitleaks = {
        enable = true;
        name = "gitleaks";
        entry = "${pkgs.gitleaks}/bin/gitleaks protect --verbose --redact --staged";
        pass_filenames = false;
      };
    };
  };
}

{
  description = "Terraform provider for k0s testing clusters";

  inputs = {
    systems.url = "github:nix-systems/default";
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    agenix-shell = {
      url = "github:aciceri/agenix-shell";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    git-hooks-nix = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    github-actions-nix = {
      url = "github:synapdeck/github-actions-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = inputs @ {
    flake-parts,
    nixpkgs,
    ...
  }: let
    # --- System Support & Package Cache --- #
    systems = import inputs.systems;

    lib = nixpkgs.lib.extend (import ./nix/lib.nix);
  in
    flake-parts.lib.mkFlake {inherit inputs;} {
      imports = with inputs; [
        agenix-shell.flakeModules.default
        treefmt-nix.flakeModule
        git-hooks-nix.flakeModule
        github-actions-nix.flakeModule
      ];

      inherit systems;

      agenix-shell = {
        identityPaths = [
          "$HOME/.ssh/id_ed25519"
        ];
        secrets = {};
      };

      perSystem = {
        pkgs,
        config,
        system,
        ...
      }: {
        _module.args.pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };

        # --- Configuration Builders --- #
        treefmt = import ./treefmt.nix {inherit lib pkgs;};
        pre-commit = import ./pre-commit.nix {inherit lib pkgs;};
        githubActions = import ./actions.nix {inherit lib pkgs;};

        # --- Devshell Configuration --- #
        devShells.default = import ./shell.nix {
          inherit lib pkgs;
          config = {
            inherit (config) pre-commit agenix-shell;
          };
        };
      };
    };
}

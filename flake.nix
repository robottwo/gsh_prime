{
  description = "A modern, POSIX-compatible, generative shell";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};
      # Read VERSION file and normalize (trim whitespace/newlines)
      rawVersion = builtins.replaceStrings ["\n" "\r" " "] ["" "" ""] (builtins.readFile ./VERSION);
      # Ensure version starts with "v" (avoid double-prefixing if VERSION already has "v")
      version =
        if builtins.substring 0 1 rawVersion == "v"
        then rawVersion
        else "v${rawVersion}";
    in {
      packages.default = pkgs.buildGoModule {
        pname = "bish";
        version = version;
        src = ./.;
        vendorHash = "sha256-Lcl6fyZf3ku8B8q4J4ljUyqhLhJ+q61DLj/Bs/RrQZo=";

        ldflags = [
          "-s"
          "-w"
          "-X main.BUILD_VERSION=${version}"
        ];

        subPackages = ["cmd/bish"];

        checkFlags = let
          # Skip tests that require network access or violate
          # the filesystem sandboxing
          skippedTests = [
            "TestReadLatestVersion"
            "TestHandleSelfUpdate_UpdateNeeded"
            "TestHandleSelfUpdate_NoUpdateNeeded"
            "TestFileCompletions"
          ];
        in ["-skip=^${builtins.concatStringsSep "$|^" skippedTests}$"];

        meta = with pkgs.lib; {
          description = "A modern, POSIX-compatible, generative shell";
          homepage = "https://github.com/robottwo/bishop";
          license = licenses.gpl3Plus;
          maintainers = [];
          mainProgram = "bish";
        };
      };

      # Backwards compatibility alias
      defaultPackage = self.packages.${system}.default;

      # Development shell
      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          gopls
          gotools
          go-tools
        ];
      };
    });
}

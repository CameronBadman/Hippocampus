{
  description = "Hippocampus";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
          ];

          shellHook = ''
            export CGO_ENABLED=0
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "hippocampus";
          version = "0.1.0";
          src = ./.;

          vendorHash = null;

          CGO_ENABLED = 0;

          meta = with pkgs.lib; {
            description = "Hippocampus - Titan embeddings service";
            license = licenses.mit;
          };
        };
      }
    );
}

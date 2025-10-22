{
  description = "Hippocampus vs FAISS Scaling Benchmark";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        pythonEnv = pkgs.python3.withPackages (ps: with ps; [
          numpy
          faiss
          boto3
        ]);
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Core tools
            bash
            bc
            
            # Go for building hippocampus
            go
            
            # Python with required packages
            pythonEnv
            
            # Additional utilities
            coreutils
            findutils
            gnused
            gnugrep
          ];

          shellHook = ''
            echo "=================================================="
            echo "Hippocampus vs FAISS Benchmark Environment"
            echo "=================================================="
            echo "Available tools:"
            echo "  • Go $(go version | cut -d' ' -f3)"
            echo "  • Python $(python --version | cut -d' ' -f2)"
            echo "  • All required Python packages (numpy, faiss, boto3)"
            echo "  • Bash and utilities for benchmarking"
            echo ""
            echo "Python packages available:"
            python -c "import numpy, faiss, boto3; print('  ✓ numpy, faiss, boto3 imported successfully')" || echo "  ✗ Missing packages"
          '';
        };
      }
    );
}

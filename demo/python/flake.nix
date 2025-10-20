{
  description = "Hippocampus Agent Demos";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      
      pythonEnv = pkgs.python3.withPackages (ps: with ps; [
        boto3
        requests
      ]);
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = [ pythonEnv ];

        shellHook = ''
          echo "Hippocampus Demo Environment"
          echo "Python: $(python --version)"
          echo ""
          echo "Available demos:"
          echo "  python python/basic_agent.py"
          echo "  python python/agentcore_agent.py"
          echo "  python python/safety_demo.py"
        '';
      };
    };
}

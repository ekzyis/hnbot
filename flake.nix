{
  description = "hnbot - crosspost top HN stories to SN";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";

  outputs = { self, nixpkgs }:
    let
      pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in
    {
      packages.x86_64-linux.default = pkgs.buildGoModule {
        name = "hnbot";
        src = ./.;

        vendorHash = "sha256-oCkPAny61rEZF1IkHAWZE+fWg7csM/3XUuFP4yyDP6g=";

        buildInputs = [ pkgs.sqlite ];

        preBuild = ''
          export CGO_ENABLED=1
          export CGO_CFLAGS="-I${pkgs.sqlite.dev}/include"
          export CGO_LDFLAGS="-L${pkgs.sqlite}/lib"
        '';
      };

      apps.x86_64-linux.default = {
        type = "app";
        program = "${self.packages.x86_64-linux.default}/bin/hnbot";
      };

      devShells.x86_64-linux.default = pkgs.mkShell {
        buildInputs = with pkgs; [ go sqlite gcc ];
        CGO_ENABLED = "1";
        shellHook = ''
          export CGO_CFLAGS="-I${pkgs.sqlite.dev}/include"
          export CGO_LDFLAGS="-L${pkgs.sqlite}/lib"
        '';
      };
    };
}

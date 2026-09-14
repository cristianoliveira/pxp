{
  description = "pxp screenshot comparison CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, utils, ... }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = "0.2.0";
        pxp = pkgs.buildGoModule {
          pname = "pxp";
          inherit version;
          src = ./.;
          vendorHash = "sha256-4pKNmHBJn50Q1hdv/7g+ep7nxEkdCCqj2eyUkZIMB5Q=";
          ldflags = [ "-X main.version=${version}" ];
          subPackages = [ "cmd/pxp" ];
          proxyVendor = true;
        };
      in {
        packages = { inherit pxp; default = pxp; };
        apps = {
          pxp = utils.lib.mkApp { drv = pxp; };
          default = utils.lib.mkApp { drv = pxp; };
        };
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go golangci-lint gotools gotestsum lefthook ];
        };
      });
}

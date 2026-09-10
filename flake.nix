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
        pxp = pkgs.buildGoModule {
          pname = "pxp";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-yN6RmmJD1ir+2LDnjMCySiiO31iE4jg/SuPp6FylrBw=";
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

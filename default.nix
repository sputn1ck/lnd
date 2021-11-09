{ pkgs ? import <nixpkgs> {}
, tags ? [ "autopilotrpc" "signrpc" "walletrpc" "chainrpc" "invoicesrpc" "watchtowerrpc" "routerrpc" "monitoring" ]
}:

pkgs.buildGoModule rec {
  pname = "lnd";
  version = "peerswap-fix";

  src = ./.;

  vendorSha256 = "sha256-RcibmGhRN4DKB6I6qaIRVT21o2c+Rdq+YJWMv8A2/pk=";

  subPackages = ["cmd/lncli" "cmd/lnd"];
  preBuild = let
      buildVars = {
        RawTags = pkgs.lib.concatStringsSep "," tags;
        GoVersion = "$(go version | egrep -o 'go[0-9]+[.][^ ]*')";
      };
      buildVarsFlags = pkgs.lib.concatStringsSep " " (pkgs.lib.mapAttrsToList (k: v: "-X github.com/lightningnetwork/lnd/build.${k}=${v}") buildVars);
    in
    pkgs.lib.optionalString (tags != []) ''
      buildFlagsArray+=("-tags=${pkgs.lib.concatStringsSep " " tags}")
      buildFlagsArray+=("-ldflags=${buildVarsFlags}")
    '';

}
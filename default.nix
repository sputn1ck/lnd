{ pkgs ? import <nixpkgs> {}
, tags ? [ "autopilotrpc" "signrpc" "walletrpc" "chainrpc" "invoicesrpc" "watchtowerrpc" "routerrpc" "monitoring" ]
}:

pkgs.buildGoModule rec {
  pname = "lnd";
  version = "peerswap-fix";

  src = ./.;

  vendorSha256 = "089drfb5kp9gyjg8xsa1wcv0pba964mrjmd7jb79ydfq12zay7fd";

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
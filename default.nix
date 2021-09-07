{ pkgs ? import <nixpkgs> {}
, tags ? [ "autopilotrpc" "signrpc" "walletrpc" "chainrpc" "invoicesrpc" "watchtowerrpc" "routerrpc" "monitoring" ]
}:

pkgs.buildGoModule rec {
  pname = "lnd";
  version = "peerswap-fix";

  src = ./.;

  vendorSha256 = "0pc40jfmv3svnbycj8028n8hgwq2x9hidgv048snl4w4g7ydh79f";

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
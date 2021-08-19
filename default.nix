{ pkgs ? import <nixpkgs> {}
, tags ? [ "autopilotrpc" "signrpc" "walletrpc" "chainrpc" "invoicesrpc" "watchtowerrpc" "routerrpc" "monitoring" ]
}:

pkgs.buildGoModule rec {
  pname = "lnd";
  version = "peerswap-fix";

  src = ./.;

  vendorSha256 = "0pc40jfmv3svnbycj8028n8hgwq2x9hidgv048snl4w4g7ydh79f";

  subPackages = ["cmd/lncli" "cmd/lnd"];

}
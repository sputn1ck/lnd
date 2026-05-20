# lnd Browser WASM Status

This branch makes `lnd` start inside a browser WASM runtime and exposes its RPC
clients through an in-process `bufconn` gRPC listener. The browser build is not
a reimplementation of lnd. It runs the normal lnd main loop with environment
adapters for pieces that browsers do not expose directly, such as raw TCP,
filesystem paths, and SQLite storage.

## Regenerating the demo

Generated browser assets are intentionally not committed. Rebuild them locally:

```shell
make wasm-assets
make wasm-build
make wasm-test
```

The targets do the following:

* `wasm-assets` extracts the `go-wasmsqlite` JavaScript/WASM runtime files,
  copies Go's `wasm_exec.js`, and copies `web/lndwasm/index.html` into
  `web/lndwasm/static/`.
* `wasm-build` compiles `./cmd/lndwasm` with `GOOS=js`, `GOARCH=wasm`, and the
  `kvdb_sqlite` tag, then writes `lndwasm.wasm` and `lndwasm.wasm.gz` into the
  generated static directory.
* `wasm-test` builds the demo and runs the Playwright browser tests.

The generated `web/lndwasm/static/` directory is ignored because the lnd WASM
binary is large and the runtime assets can be reproduced from source.

## What works

The current browser demo has been verified with Playwright against local helper
nodes. The test path starts lnd in Chromium, creates or unlocks the wallet,
generates addresses, funds the browser wallet, sends an on-chain transaction,
connects to Lightning peers through the WebSocket peer proxy, opens zero-conf
channels, creates invoices, receives payments, and sends payments.

The SDK path is:

* `wasmsdk.Start` starts lnd and returns a running instance.
* lnd listens for RPC on an injected `bufconn` listener.
* callers can use typed RPC clients from the SDK without TLS or macaroon files
  crossing a browser filesystem boundary.
* browser peer connections use an injected `tor.Net` implementation backed by a
  WebSocket tunnel.

The demo defaults to signet with the public mempool.space Esplora API and a
configurable WSS peer proxy. Local browser tests use regtest helpers so they are
repeatable and do not depend on public infrastructure.

## Why this is fairly mature

Most of the running system is the production lnd stack. Startup, wallet unlock,
RPC services, wallet state, invoice state, channel state, payment routing, and
peer protocol handling still run through normal lnd packages. The browser work
is concentrated at integration boundaries:

* storage is adapted to browser OPFS through `go-wasmsqlite`;
* chain access is adapted through an Esplora-compatible backend;
* RPC transport is adapted through `bufconn`;
* Lightning peer transport is adapted through a browser WebSocket bridge;
* direct filesystem-only pieces are replaced by JS-safe wrappers where needed.

That means the risk is mostly in the browser adapters and deployment topology,
not in a separate wallet or Lightning implementation.

## Current limitations

Raw TCP is not available in browsers. A browser lnd cannot directly dial a
normal `host:port` Lightning peer. It needs a proxy that accepts WebSocket or
WebTransport from the browser and forwards raw brontide bytes to the Lightning
peer. A non-upgraded Lightning peer is still reachable, but only through that
proxy. The peer does not need to know about WebSockets.

Public Esplora endpoints can rate-limit aggressively. The Esplora client retries
context-cancelably on transient HTTP failures such as 429, 408, 502, 503, 504,
and other 5xx responses. For reliable applications, a dedicated Esplora/electrs
instance is still preferred over public mempool.space.

The Esplora chain backend is polling-based. It is enough for the current wallet,
funding, channel, and payment demo path, but it is not equivalent to a fully
indexed bitcoind or neutrino backend with native notification streams. Reorg and
notification behavior should receive more focused testing before treating this
as production-ready.

Channel backup persistence currently uses an injected in-memory swapper for the
browser SDK path. That avoids direct filesystem writes, but it is not a durable
static channel backup strategy. A production browser wallet should persist and
export encrypted SCBs through OPFS or an explicit application backup flow.

Browser storage depends on OPFS and cross-origin isolation. The demo defaults to
SQLite's `opfs-sahpool` VFS and requires persistent storage for non-memory
modes. It also exposes an explicit `memory` mode for debugging; when selected,
the page shows a warning because wallet, channel, and payment state are
ephemeral. The demo server sets COOP/COEP headers and the GitHub Pages flow uses
a service-worker based setup so SQLite can use persistent OPFS storage. Sites
embedding the SDK need to serve the same isolation headers and should handle
browsers where OPFS is unavailable or cleared by the user.

The WASM binary is large and startup is heavier than a normal web wallet. This
is expected because the browser is running lnd itself. The compressed artifact is
more practical for deployment, but still large enough that application UX should
show explicit loading and log output.

The browser build has no direct access to OS services such as normal signal
handling, filesystem locks, TLS certificate files, or local TCP listeners. The
SDK injects alternatives for the demo path, but other lnd subsystems may still
need more wrappers as additional features are enabled.

The signet WSS probe reached an active lnd instance and attempted to dial the
configured proxy URL, but the public tunnel returned HTTP 502 during the
WebSocket handshake. That shows the browser code issued the expected upgrade
request, while the remaining failure was at the deployed proxy/tunnel boundary
before a brontide session could start.

## Next hardening work

The main remaining work is operational hardening rather than proving that lnd can
start. The most important next steps are:

* make static channel backup storage durable in OPFS or an application-provided
  export path;
* run a private Esplora/electrs service for deterministic signet tests;
* harden the public peer proxy with structured logging before and after the
  WebSocket upgrade;
* add long-running reorg and rescan tests for the Esplora chain backend;
* reduce artifact size and improve startup progress reporting.

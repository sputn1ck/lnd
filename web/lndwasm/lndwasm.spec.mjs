import { test, expect } from "@playwright/test";
import { spawn } from "node:child_process";
import readline from "node:readline";

const repoRoot = new URL("../..", import.meta.url).pathname;

test.afterEach(async ({ page }) => {
  await page.evaluate(async () => {
    if (typeof window.lndWasmStop === "function") {
      await window.lndWasmStop();
    }
  }).catch(() => {});
});

test("starts lnd in browser wasm and reaches RPC ready", async ({ page }) => {
  const messages = [];
  page.on("console", (message) => {
    const text = message.text();
    messages.push(text);
    console.log(text);
  });

  await page.goto("/");
    await page.waitForFunction(() => typeof window.lndWasmStart === "function", null, {
      timeout: 60_000,
    });

    await page.getByTestId("network").selectOption("regtest");
    await page.getByTestId("esplora-url").fill("");
    await page.getByTestId("peer-proxy-url").fill("");
    await expect(page.getByTestId("storage-status")).toContainText("auto");
    await page.getByTestId("storage-vfs").selectOption("memory");
    await expect(page.getByTestId("storage-warning")).toBeVisible();
    await page.getByTestId("storage-vfs").selectOption("auto");
    await expect(page.getByTestId("storage-warning")).toBeHidden();
    const result = await page.evaluate(async () => window.startLndWasmDemo());

  expect(result.rpcReady).toBe(true);
  expect(result.storage.requestedVFS).toBe("auto");
  expect(result.storage.activeVFS).toBe("opfs-wl");
  expect(result.storage.persistent).toBe(true);
  expect(result.walletState).toBeTruthy();
  expect(result.wallet.walletState).toMatch(/RPC_ACTIVE|SERVER_ACTIVE/);
  expect(result.wallet.identityPubkey).toMatch(/^[0-9a-f]{66}$/);
  expect(result.wallet.alias).toBeTruthy();
  expect(await page.evaluate(() => window.__lndWasmFailure)).toBe("");
  expect(await page.evaluate(() => window.__lndWasmPhase)).toBe("get-info");
  expect(messages.join("\n")).not.toContain("wasm runtime failed");
  expect(messages.join("\n")).not.toContain("database already open");
});

test("connects to a brontide peer through the browser peer proxy", async ({ page }) => {
  const peer = await startBrontidePeer();
  test.info().attach("peer", {
    body: JSON.stringify(peer.info, null, 2),
    contentType: "application/json",
  });

  try {
    await page.goto("/");
    await page.waitForFunction(() => typeof window.lndWasmStart === "function", null, {
      timeout: 60_000,
    });

    const result = await page.evaluate(async () => {
      window.lndWasmBitcoinNetwork = "regtest";
      window.lndWasmPeerProxyURL = "ws://127.0.0.1:8765/peer-proxy";
      return window.startLndWasmDemo(window.lndWasmDefaultArgs());
    });
    expect(result.wallet.identityPubkey).toMatch(/^[0-9a-f]{66}$/);

    const connect = await page.evaluate(
      async ({ pubkey, host }) => window.lndWasmConnectPeer(pubkey, host),
      peer.info,
    );

    expect(connect.status).toContain("connection to");
    expect(connect.numPeers).toBeGreaterThanOrEqual(1);
  } finally {
    await peer.stop();
  }
});

test("connects to a native lnd peer through the browser peer proxy", async ({ page }) => {
  const peer = await startNativeLndPeer();
  test.info().attach("peer", {
    body: JSON.stringify(peer.info, null, 2),
    contentType: "application/json",
  });

  try {
    await page.goto("/");
    await page.waitForFunction(() => typeof window.lndWasmStart === "function", null, {
      timeout: 60_000,
    });

    const result = await page.evaluate(async () => {
      window.lndWasmBitcoinNetwork = "regtest";
      window.lndWasmPeerProxyURL = "ws://127.0.0.1:8765/peer-proxy";
      return window.startLndWasmDemo(window.lndWasmDefaultArgs());
    });
    expect(result.wallet.identityPubkey).toMatch(/^[0-9a-f]{66}$/);

    const connect = await page.evaluate(
      async ({ pubkey, host }) => window.lndWasmConnectPeer(pubkey, host),
      peer.info,
    );

    expect(connect.status).toContain("connection to");
    expect(connect.numPeers).toBeGreaterThanOrEqual(1);
  } finally {
    await peer.stop();
  }
});

test("opens a zero-conf channel from a chain-backed lnd peer", async ({ page }) => {
  const peer = await startChainLndPeer();
  test.info().attach("peer", {
    body: JSON.stringify(peer.info, null, 2),
    contentType: "application/json",
  });

  try {
    await page.goto("/");
    await page.waitForFunction(() => typeof window.lndWasmStart === "function", null, {
      timeout: 60_000,
    });

    const result = await page.evaluate(async ({ esploraURL }) => {
      window.lndWasmBitcoinNetwork = "regtest";
      window.lndWasmPeerProxyURL = "ws://127.0.0.1:8765/peer-proxy";
      window.lndWasmEsploraURL = esploraURL;
      const started = await window.startLndWasmDemo(window.lndWasmDefaultArgs());
      await window.lndWasmAcceptZeroConfChannels();
      return started;
    }, { esploraURL: peer.info.esplora_url });
    expect(result.wallet.identityPubkey).toMatch(/^[0-9a-f]{66}$/);

    const connect = await page.evaluate(
      async ({ pubkey, host }) => window.lndWasmConnectPeer(pubkey, host),
      peer.info,
    );
    expect(connect.numPeers).toBeGreaterThanOrEqual(1);

    const opened = await peer.command("open_zero_conf", {
      pubkey: result.wallet.identityPubkey,
    });
    expect(opened.ok, opened.error).toBe(true);
    expect(opened.channel_point).toContain(":");

    await expect.poll(async () => {
      const channels = await page.evaluate(async () => window.lndWasmListChannels());
      return channels.channels.length;
    }, { timeout: 60_000 }).toBeGreaterThanOrEqual(1);

    const channels = await page.evaluate(async () => window.lndWasmListChannels());
    expect(channels.channels.some((channel) => channel.zeroConf)).toBe(true);

    const browserInvoice = await page.evaluate(
      async () => window.lndWasmAddInvoice(10_000),
    );
    expect(browserInvoice.paymentRequest).toMatch(/^lnbcrt/);
    expect(browserInvoice.paymentHash).toMatch(/^[0-9a-f]{64}$/);

    const nativePayment = await peer.command("pay_invoice", {
      invoice: browserInvoice.paymentRequest,
    });
    expect(nativePayment.ok, nativePayment.error).toBe(true);
    expect(nativePayment.payment_hash).toBe(browserInvoice.paymentHash);
    expect(nativePayment.payment_preimage).toMatch(/^[0-9a-f]{64}$/);

    const settled = await page.evaluate(
      async (hash) => window.lndWasmWaitInvoiceSettled(hash),
      browserInvoice.paymentHash,
    );
    expect(settled.state).toBe("SETTLED");
    expect(settled.amtPaidSat).toBe(10_000);

    const channelsAfterReceive = await page.evaluate(
      async () => window.lndWasmListChannels(),
    );
    const sendChannel = channelsAfterReceive.channels.find(
      (channel) => channel.zeroConf && channel.active && channel.localBalance > 1_000,
    );
    expect(sendChannel).toBeTruthy();

    const nativeInvoice = await peer.command("create_invoice", {
      amount: 1_000,
      pubkey: result.wallet.identityPubkey,
    });
    expect(nativeInvoice.ok, nativeInvoice.error).toBe(true);
    expect(nativeInvoice.payment_request).toMatch(/^lnbcrt/);
    expect(nativeInvoice.payment_hash).toMatch(/^[0-9a-f]{64}$/);

    const browserPayment = await page.evaluate(
      async ({ invoice, chanId }) => window.lndWasmPayInvoice(invoice, chanId),
      {
        invoice: nativeInvoice.payment_request,
        chanId: sendChannel.chanId,
      },
    );
    expect(browserPayment.paymentHash).toBe(nativeInvoice.payment_hash);
    expect(browserPayment.paymentPreimage).toMatch(/^[0-9a-f]{64}$/);
  } finally {
    await peer.stop();
  }
});

test("drives the wallet interface through wallet, peer, channel, and payment operations", async ({ page }) => {
  const peer = await startChainLndPeer();
  test.info().attach("peer", {
    body: JSON.stringify(peer.info, null, 2),
    contentType: "application/json",
  });

  try {
    await page.goto("/");
    await page.waitForFunction(() => typeof window.lndWasmStart === "function", null, {
      timeout: 60_000,
    });

    await page.getByTestId("esplora-url").fill(peer.info.esplora_url);
    await page.getByTestId("peer-proxy-url").fill("ws://127.0.0.1:8765/peer-proxy");
    await page.getByTestId("network").selectOption("regtest");
    await page.getByTestId("start-wallet").click();
    await expect(page.locator("#dashboard-view")).toBeVisible({ timeout: 180_000 });

    await page.getByTestId("new-address").click();
    await expect(page.locator("#address-result")).toContainText("bcrt", { timeout: 30_000 });
    const fundAddress = await page.getByTestId("send-address").inputValue();
    expect(fundAddress).toMatch(/^bcrt/);

    const funded = await peer.command("fund_address", {
      address: fundAddress,
      amount: 300_000,
    });
    expect(funded.ok, funded.error).toBe(true);
    expect(funded.txid).toMatch(/^[0-9a-f]{64}$/);

    await expect.poll(async () => {
      return Number(await page.locator("#confirmed-balance").textContent());
    }, { timeout: 90_000 }).toBeGreaterThanOrEqual(250_000);

    await page.getByTestId("send-amount").fill("1000");
    await page.getByTestId("send-coins").click();
    await expect(page.locator("#send-result")).toContainText("txid", { timeout: 60_000 });

    await page.getByTestId("peer-pubkey").fill(peer.info.pubkey);
    await page.getByTestId("peer-host").fill(peer.info.host);
    await page.getByTestId("connect-peer").click();
    await expect(page.getByTestId("peers-table")).toContainText(peer.info.pubkey, {
      timeout: 30_000,
    });

    await page.evaluate(async () => window.lndWasmAcceptZeroConfChannels());
    const nativeOpened = await peer.command("open_zero_conf", {
      pubkey: await page.evaluate(() => window.__lndWasmWallet.identityPubkey),
    });
    expect(nativeOpened.ok, nativeOpened.error).toBe(true);
    await page.getByTestId("refresh-channels").click();
    await expect(page.getByTestId("channels-table")).toContainText(peer.info.pubkey, {
      timeout: 30_000,
    });
    const paymentChannel = await page.evaluate(async () => {
      const channels = await window.lndWasmListChannels();
      return channels.channels.find((channel) => channel.localBalance > 1000);
    });
    expect(paymentChannel).toBeTruthy();

    const accepting = await peer.command("accept_zero_conf");
    expect(accepting.ok, accepting.error).toBe(true);

    await page.getByTestId("channel-amount").fill("100000");
    await page.getByTestId("channel-push").fill("50000");
    await page.locator("#channel-zero-conf").check();
    await page.getByTestId("open-channel").click();
    await expect(page.locator("#channel-result")).toContainText("channelPoint", {
      timeout: 60_000,
    });
    await expect(page.getByTestId("channels-table")).toContainText(peer.info.pubkey, {
      timeout: 30_000,
    });

    await page.getByTestId("invoice-amount").fill("5000");
    await page.getByTestId("create-invoice").click();
    await expect(page.locator("#invoice-result")).toContainText("paymentRequest", {
      timeout: 30_000,
    });
    const browserInvoice = await page.getByTestId("payment-request").inputValue();
    expect(browserInvoice).toMatch(/^lnbcrt/);

    const nativeInvoice = await peer.command("create_invoice", {
      amount: 1000,
      pubkey: await page.evaluate(() => window.__lndWasmWallet.identityPubkey),
    });
    expect(nativeInvoice.ok, nativeInvoice.error).toBe(true);

    await page.getByTestId("payment-request").fill(nativeInvoice.payment_request);
    await page.getByTestId("payment-channel").selectOption(paymentChannel.chanId);
    await page.getByTestId("send-payment").click();
    await expect(page.locator("#payment-result")).toContainText("paymentPreimage", {
      timeout: 60_000,
    });
  } finally {
    await peer.stop();
  }
});

async function startBrontidePeer() {
  return startGoJSONHelper("brontide peer", ["run", "./cmd/lndwasmpeer"]);
}

async function startNativeLndPeer() {
  return startGoJSONHelper("native lnd peer", ["run", "./cmd/lndwasmnativepeer"]);
}

async function startChainLndPeer() {
  return startGoJSONHelper("chain-backed lnd peer", ["run", "./cmd/lndwasmchainpeer"]);
}

async function startGoJSONHelper(name, args) {
  const child = spawn("go", args, {
    cwd: repoRoot,
    env: {
      ...process.env,
      GOCACHE: "/private/tmp/codex-go-build",
    },
    stdio: ["pipe", "pipe", "pipe"],
  });

  let stderr = "";
  let stdout = "";
  child.stderr.on("data", (chunk) => {
    stderr += chunk.toString();
  });

  const lines = readline.createInterface({ input: child.stdout });
  const pending = new Map();
  let commandID = 0;
  let ready = false;
  const info = await new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error(`timed out starting ${name}: stdout=${stdout} stderr=${stderr}`));
    }, 180_000);

    child.once("exit", (code) => {
      clearTimeout(timer);
      reject(new Error(`${name} exited with ${code}: stdout=${stdout} stderr=${stderr}`));
    });

    lines.on("line", (line) => {
      stdout += `${line}\n`;
      if (!line.startsWith("{")) {
        return;
      }

      try {
        const parsed = JSON.parse(line);
        if (!ready) {
          ready = true;
          clearTimeout(timer);
          resolve(parsed);
          return;
        }

        if (parsed.id && pending.has(parsed.id)) {
          pending.get(parsed.id).resolve(parsed);
          pending.delete(parsed.id);
        }
      } catch (error) {
        clearTimeout(timer);
        reject(new Error(`invalid ${name} JSON ${line}: ${error.message}`));
      }
    });
  });

  return {
    info,
    command(method, params = {}) {
      const id = `${name}-${++commandID}`;
      const payload = JSON.stringify({ id, method, ...params });

      return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
          pending.delete(id);
          reject(new Error(`timed out waiting for ${method}: stdout=${stdout} stderr=${stderr}`));
        }, 120_000);

        pending.set(id, {
          resolve(result) {
            clearTimeout(timer);
            resolve(result);
          },
        });
        child.stdin.write(`${payload}\n`, (error) => {
          if (error) {
            clearTimeout(timer);
            pending.delete(id);
            reject(error);
          }
        });
      });
    },
    stop() {
      return new Promise((resolve) => {
        if (child.exitCode !== null) {
          resolve();
          return;
        }

        const timer = setTimeout(() => {
          if (child.exitCode === null) {
            child.kill("SIGTERM");
          }
        }, 2_000);
        timer.unref();

        child.once("exit", () => {
          clearTimeout(timer);
          resolve();
        });
        child.stdin.end();
      });
    },
  };
}

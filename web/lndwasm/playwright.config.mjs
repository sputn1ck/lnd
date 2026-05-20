export default {
  testDir: ".",
  testMatch: "lndwasm.spec.mjs",
  timeout: 360_000,
  use: {
    baseURL: "http://127.0.0.1:8765",
    browserName: "chromium",
  },
  webServer: {
    command: "node server.mjs static",
    url: "http://127.0.0.1:8765",
    reuseExistingServer: false,
    timeout: 10_000,
  },
};

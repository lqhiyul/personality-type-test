import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL: "http://127.0.0.1:18080",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "go run ./cmd/server",
    url: "http://127.0.0.1:18080/healthz",
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
    env: {
      HOST: "127.0.0.1",
      PORT: "18080",
      ADMIN_PASSWORD: "e2e-admin-password",
      DATA_FILE: ".e2e-data/results.json",
      DATABASE_PATH: ".e2e-data/app.db",
      COOKIE_SECURE: "false",
    },
  },
});

import { chromium } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import { spawn } from "node:child_process";

const baseURL = "http://127.0.0.1:18180";
const outDir = "docs/screenshots";

function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHealth() {
  for (let i = 0; i < 60; i += 1) {
    try {
      const response = await fetch(`${baseURL}/healthz`);
      if (response.ok) return;
    } catch (_) {}
    await wait(500);
  }
  throw new Error("server did not become healthy");
}

async function withServer(fn) {
  const child = spawn("go", ["run", "./cmd/server"], {
    stdio: "inherit",
    env: {
      ...process.env,
      HOST: "127.0.0.1",
      PORT: "18180",
      ADMIN_PASSWORD: "screenshot-admin-password",
      DATA_FILE: ".screenshot-data/results.json",
      DATABASE_PATH: ".screenshot-data/app.db",
      COOKIE_SECURE: "false",
    },
  });
  try {
    await waitForHealth();
    await fn();
  } finally {
    child.kill();
  }
}

async function answerQuiz(page) {
  await page.locator("#personName").fill("Portfolio Reviewer");
  const count = await page.locator("[data-question]").count();
  for (let index = 0; index < count; index += 1) {
    await page.locator(`[data-question="${index}"] .option`).first().click();
  }
  await page.locator("#submitBtn").click();
  await page.locator("#resultBox").waitFor({ state: "visible" });
}

await mkdir(outDir, { recursive: true });

await withServer(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1440, height: 1200 } });

  await page.goto(baseURL);
  await page.screenshot({ path: `${outDir}/home.png`, fullPage: false });

  await answerQuiz(page);
  await page.screenshot({ path: `${outDir}/quiz-result.png`, fullPage: false });

  await page.locator("#tabTypes").click();
  await page.screenshot({ path: `${outDir}/types.png`, fullPage: false });

  await page.locator("#tabCompatibility").click();
  await page.locator("#compatTypeA").selectOption("INTJ");
  await page.locator("#compatTypeB").selectOption("ENFP");
  await page.locator("[data-run-compatibility]").click();
  await page.locator(".compatibility-result").waitFor({ state: "visible" });
  await page.screenshot({ path: `${outDir}/compatibility.png`, fullPage: false });

  await page.locator("#accountBtn").click();
  await page.locator("#authRegisterModeBtn").click();
  await page.locator("#authUsername").fill(`shot-${Date.now()}`);
  await page.locator("#authEmail").fill(`shot-${Date.now()}@example.com`);
  await page.locator("#authPassword").fill("StrongPassword123");
  await page.locator("#authSubmitBtn").click();
  await page.locator("#accountSignedIn").waitFor({ state: "visible" });
  await page.screenshot({ path: `${outDir}/profile.png`, fullPage: false });

  await page.goto(`${baseURL}/?admin=1`);
  await page.locator("#adminCard").waitFor({ state: "visible" });
  await page.locator("#adminPassword").fill("screenshot-admin-password");
  await page.locator("#loginBtn").click();
  await page.locator("#adminPanel.visible").waitFor({ state: "visible" });
  await page.screenshot({ path: `${outDir}/admin.png`, fullPage: false });

  await browser.close();
});

console.log(`screenshots written to ${outDir}`);

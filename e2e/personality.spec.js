import { expect, test } from "@playwright/test";

test("home page loads", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#quizSection")).toBeVisible();
  await expect(page.locator("#submitBtn")).toBeVisible();
});

test("quiz can be completed and shows a result", async ({ page }) => {
  await page.goto("/");
  await page.locator("#personName").fill("E2E Tester");
  const count = await page.locator("[data-question]").count();
  for (let index = 0; index < count; index += 1) {
    await page.locator(`[data-question="${index}"] .option`).first().click();
  }
  await page.locator("#submitBtn").click();
  await expect(page.locator("#resultBox")).toBeVisible();
  await expect(page.locator("#resultBox")).toContainText(/E|I/);
});

test("types and compatibility sections work", async ({ page }) => {
  await page.goto("/");
  await page.locator("#tabTypes").click();
  await expect(page.locator("#typesSection")).toBeVisible();
  await expect(page.locator("#typeSearch")).toBeVisible();

  await page.locator("#tabCompatibility").click();
  await page.locator("#compatTypeA").selectOption("INTJ");
  await page.locator("#compatTypeB").selectOption("ENFP");
  await page.locator("[data-run-compatibility]").click();
  await expect(page.locator(".compatibility-result")).toBeVisible();
});

test("user can register, view profile, and logout", async ({ page }) => {
  const suffix = Date.now();
  await page.goto("/");
  await page.locator("#accountBtn").click();
  await page.locator("#authRegisterModeBtn").click();
  await page.locator("#authUsername").fill(`e2e-${suffix}`);
  await page.locator("#authEmail").fill(`e2e-${suffix}@example.com`);
  await page.locator("#authPassword").fill("StrongPassword123");
  await page.locator("#authSubmitBtn").click();
  await expect(page.locator("#accountSignedIn")).toBeVisible();

  await page.locator("#viewPublicProfileBtn").click();
  await expect(page.locator("#profileSection")).toBeVisible();
  await expect(page.locator("#profileSection")).toContainText(`e2e-${suffix}`);

  await page.locator("#accountBtn").click();
  await page.locator("#authLogoutBtn").click();
  await expect(page.locator("#accountSignedOut")).toBeVisible();
});

test("friends, messages, and admin protected surfaces do not crash", async ({ page }) => {
  const suffix = Date.now();
  await page.goto("/?admin=1");
  await expect(page.locator("#adminCard")).toBeVisible();
  await expect(page.locator("#adminPanel")).not.toHaveClass(/visible/);

  await page.locator("#accountBtn").click();
  await page.locator("#authRegisterModeBtn").click();
  await page.locator("#authUsername").fill(`social-${suffix}`);
  await page.locator("#authEmail").fill(`social-${suffix}@example.com`);
  await page.locator("#authPassword").fill("StrongPassword123");
  await page.locator("#authSubmitBtn").click();

  await expect(page.locator("#friendsList")).toBeVisible();
  await expect(page.locator("#messagesConversationList")).toBeVisible();
});

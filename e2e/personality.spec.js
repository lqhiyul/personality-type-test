import { expect, test } from "@playwright/test";

async function answerAllFirstOptions(page) {
  await page.locator("[data-question]").first().waitFor();
  await page.locator("[data-question]").evaluateAll((questions) => {
    for (const question of questions) {
      question.querySelector(".option")?.click();
    }
  });
}

test("home page loads", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#quizSection")).toBeVisible();
  await expect(page.locator("#submitBtn")).toBeVisible();
});

test("quiz can be completed and shows a result", async ({ page }) => {
  await page.goto("/");
  await page.locator("#personName").fill("E2E Tester");
  await answerAllFirstOptions(page);
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

test("localized quiz, compatibility, and result text renders cleanly", async ({ page }) => {
  const mojibake = /РЎ|Рџ|Рў|Рµ|СЊ|Сѓ|С‚|вЂ|Рґ|Рє|РЅ|Рѕ|�/;
  const languages = [
    {
      code: "uk",
      quizTitle: "Відповідайте на ситуації, а не на ярлики",
      firstQuestion: "Коли після напруженого дня",
      compatibilityTitle: "Сумісність типів",
      compatibilitySection: "Що може працювати добре",
      resultAction: "Поділитися результатом",
    },
    {
      code: "ru",
      quizTitle: "Отвечайте на ситуации, а не на ярлыки",
      firstQuestion: "Когда после напряженного дня",
      compatibilityTitle: "Совместимость типов",
      compatibilitySection: "Что может работать хорошо",
      resultAction: "Поделиться результатом",
    },
    {
      code: "en",
      quizTitle: "Answer situations, not labels",
      firstQuestion: "After a demanding day",
      compatibilityTitle: "Type Compatibility",
      compatibilitySection: "What can work well",
      resultAction: "Share result",
    },
  ];

  await page.goto("/");
  await page.locator("#personName").fill("Localization Check");
  await answerAllFirstOptions(page);
  await page.locator("#submitBtn").click();
  await expect(page.locator("#resultBox")).toBeVisible();

  for (const language of languages) {
    await page.locator(`[data-lang="${language.code}"]`).click();
    await page.locator("#tabQuiz").click();
    await expect(page.locator("#quizTitle")).toHaveText(language.quizTitle);
    await expect(page.locator('[data-question="0"]')).toContainText(language.firstQuestion);
    await expect(page.locator("#resultBox")).toContainText(language.resultAction);
    await expect(page.locator("#quizSection")).not.toContainText(mojibake);

    await page.locator("#tabCompatibility").click();
    await expect(page.locator("#compatibilityTitle")).toHaveText(language.compatibilityTitle);
    await page.locator("#compatTypeA").selectOption("INTJ");
    await page.locator("#compatTypeB").selectOption("ENFP");
    await page.locator("[data-run-compatibility]").click();
    await expect(page.locator(".compatibility-result")).toContainText(language.compatibilitySection);
    await expect(page.locator("#compatibilitySection")).not.toContainText(mojibake);
  }
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

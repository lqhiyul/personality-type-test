function getContent(lang = state.lang) {
  return CONTENT[lang] || CONTENT.uk || CONTENT.en || {};
}

function t(path, fallback = "", params = {}) {
  const value = path.split(".").reduce((acc, key) => (acc && acc[key] !== undefined ? acc[key] : undefined), getContent());
  return template(value ?? fallback, params);
}

function questions() {
  return (getContent().questions || []).map((question, index) => ({
    ...question,
    meta: {
      ...(QUESTION_METADATA[index] || {}),
      ...(question.meta || {}),
    },
  }));
}

function totalQuestions() {
  return questions().length;
}

function detectLanguage() {
  const saved = safeGetStorage(LANG_KEY);
  if (LANGS.includes(saved)) return saved;

  const browser = (navigator.language || navigator.userLanguage || "").toLowerCase();
  if (browser.startsWith("ru")) return "ru";
  if (browser.startsWith("en")) return "en";
  return "uk";
}

function setLanguage(lang, options = {}) {
  const { persist = true, render = true } = options;
  state.lang = LANGS.includes(lang) ? lang : "uk";
  state.typeFilter = "all";
  if (persist) safeSetStorage(LANG_KEY, state.lang);

  const total = totalQuestions();
  if (state.answers.length !== total) {
    state.answers = Array(total).fill(null);
  }

  applyStaticText();
  renderLanguageButtons();
  if (!render) return;
  renderQuiz();
  updateProgress({ persist: false });
  if (!E("typesSection")?.hidden) {
    state.activeType ? renderType(state.activeType) : renderTypesGrid();
  }
  if (!E("compatibilitySection")?.hidden) {
    renderCompatibility();
  }
  if (state.lastResult) {
    showResult(state.lastResult.type, state.lastResult.name, state.lastResult.profile, { scroll: false });
  }
}

function setText(id, value) {
  const el = E(id);
  if (el) el.textContent = value;
}

function setPlaceholder(id, value) {
  const el = E(id);
  if (el instanceof HTMLInputElement) el.placeholder = value;
}

function applyStaticText() {
  const content = getContent();
  document.documentElement.lang = content.code || state.lang;
  document.title = content.title || "Personality Type Test";

  const description = document.querySelector("meta[name='description']");
  if (description) description.setAttribute("content", content.description || "");
  const ogTitle = document.querySelector("meta[property='og:title']");
  if (ogTitle) ogTitle.setAttribute("content", content.title || "");
  const ogDescription = document.querySelector("meta[property='og:description']");
  if (ogDescription) ogDescription.setAttribute("content", content.description || "");

  setText("skipLink", t("ui.skip"));
  setText("tabQuiz", t("ui.tabs.quiz"));
  setText("tabTypes", t("ui.tabs.types"));
  setText("tabCompatibility", t("ui.tabs.compatibility", t("ui.compatibility.toolsLabel", "Compatibility")));
  const languageSwitcher = E("languageSwitcher");
  if (languageSwitcher) languageSwitcher.setAttribute("aria-label", t("ui.languageLabel"));
  const adminAccess = E("adminAccessBtn");
  if (adminAccess) {
    const label = state.adminAccessOpen ? t("ui.admin.closeAccess", t("ui.admin.closePanel", "Hide admin tools")) : t("ui.admin.openPanel", "Open admin tools");
    adminAccess.setAttribute("aria-label", label);
    adminAccess.setAttribute("title", label);
    adminAccess.setAttribute("aria-expanded", String(state.adminAccessOpen));
  }
  setText("adminAccessTitle", t("ui.admin.accessTitle", "Admin tools"));
  setText("adminAccessHint", t("ui.admin.accessHint", "Local QA, demo run, and exports."));
  setText("adminOpenBtn", t("ui.admin.accessOpen", "Open panel"));

  setText("quizEyebrow", t("ui.heroQuiz.eyebrow"));
  setText("quizTitle", t("ui.heroQuiz.title"));
  setText("quizCopy", t("ui.heroQuiz.copy"));
  setText("typesEyebrow", t("ui.heroTypes.eyebrow"));
  setText("typesTitle", t("ui.heroTypes.title"));
  setText("typesCopy", t("ui.heroTypes.copy"));
  setText("compatibilityEyebrow", t("ui.heroCompatibility.eyebrow", t("ui.compatibility.toolsLabel", "Compatibility")));
  setText("compatibilityTitle", t("ui.heroCompatibility.title", t("ui.tabs.compatibility", "Type Compatibility")));
  setText("compatibilityCopy", t("ui.heroCompatibility.copy", ""));

  setText("metricQuestionsLabel", t("ui.progress.questions"));
  setText("metricDoneLabel", t("ui.progress.done"));
  setText("metricLeftLabel", t("ui.progress.left"));
  setText("personNameLabel", t("ui.form.nameLabel"));
  setPlaceholder("personName", t("ui.form.namePlaceholder"));
  setText("submitBtn", t("ui.form.submit"));
  setText("resetBtn", t("ui.form.reset"));

  const resultBox = E("resultBox");
  if (resultBox) resultBox.setAttribute("aria-label", t("ui.result.aria"));
  setText("resultTitle", t("ui.result.emptyTitle"));
  setText("resultDesc", t("ui.result.emptyCopy"));

  setText("adminTitle", t("ui.admin.title"));
  setText("adminCopy", t("ui.admin.copy"));
  setPlaceholder("adminPassword", t("ui.admin.password"));
  setText("loginBtn", t("ui.admin.login"));
  setText("logoutBtn", t("ui.admin.logout"));
  setText("exportBtn", t("ui.admin.exportCsv"));
  setText("exportJsonBtn", t("ui.admin.exportJson"));
  setText("clearBtn", t("ui.admin.clear"));
  setText("adminSearchLabel", t("ui.admin.search"));
  setPlaceholder("adminSearch", t("ui.admin.searchPlaceholder"));
  setText("adminStatsLabel", t("ui.admin.results"));
  setText("thName", t("ui.admin.table.name"));
  setText("thType", t("ui.admin.table.type"));
  setText("thDate", t("ui.admin.table.date"));
  setText("thAction", t("ui.admin.table.action"));
  setText("adminDemoTitle", t("ui.admin.demoTitle", "Demo/autopass"));
  setText("adminDemoCopy", t("ui.admin.demoCopy", "Preview a result page without saving it."));
  setText("demoTypeLabel", t("ui.admin.demoType", "Type to preview"));
  setText("demoNameLabel", t("ui.admin.demoName", "Demo name"));
  setPlaceholder("demoName", t("ui.admin.demoNamePlaceholder", "Demo INTJ"));
  setText("demoRunBtn", t("ui.admin.demoRun", "Demo run"));
  setText("demoCancelBtn", t("ui.admin.demoCancel", "Cancel"));
  renderDemoTypeOptions();
  if (typeof applyAuthStaticText === "function") applyAuthStaticText();

  const trust = E("trustNote");
  if (trust) {
    trust.innerHTML = `<strong>${esc(t("ui.trust.title"))}</strong><p>${esc(t("ui.trust.text"))}</p>`;
  }
  setText("footerText", t("ui.footer"));
}

function renderLanguageButtons() {
  document.querySelectorAll("[data-lang]").forEach((button) => {
    const active = button.dataset.lang === state.lang;
    button.classList.toggle("active", active);
    button.setAttribute("aria-pressed", String(active));
  });
}

function renderDemoTypeOptions() {
  const select = E("demoTypeSelect");
  if (!select) return;
  const current = select.value || "random";
  select.innerHTML = [
    `<option value="random">${esc(t("ui.admin.demoRandom", "Random type"))}</option>`,
    ...TYPE_GRID_ORDER.map((code) => `<option value="${esc(code)}">${esc(code)} - ${esc(getTypeName(code))}</option>`),
  ].join("");
  select.value = TYPE_GRID_ORDER.includes(current) ? current : "random";
}


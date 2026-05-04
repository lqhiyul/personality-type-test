const API = {
  submit: "/api/submit",
  login: "/api/login",
  logout: "/api/logout",
  results: "/api/results",
  export: "/api/results/export",
};

const DRAFT_KEY = "personality-test-draft:v3";
const LANG_KEY = "personality-test-language";
const TELEGRAM_URL = "https://t.me/+H1RfT8lJFYA0MDI6";
const LANGS = ["uk", "ru", "en"];
const TYPE_GRID_ORDER = ["INTJ", "INTP", "ENTJ", "ENTP", "INFJ", "INFP", "ENFJ", "ENFP", "ISTJ", "ISFJ", "ESTJ", "ESFJ", "ISTP", "ISFP", "ESTP", "ESFP"];
const COMPATIBILITY_CONTEXTS = ["friendship", "relationship", "work"];
const SHARE_CARD_ASSETS = Object.freeze(Object.fromEntries(TYPE_GRID_ORDER.map((type) => [type, `/assets/share-cards/${type.toLowerCase()}.png`])));
const QUESTION_METADATA = [
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["energy", "social-recharge"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["group-entry", "communication"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["thinking-process"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["recovery"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["teamwork", "pace"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["outer-style"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["idea-sharing"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["perception", "evidence"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["information-style"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["ideas", "application"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["attention"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["learning-style"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["planning"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["abstraction"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["decision-making"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["feedback"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["conflict"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["impact"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["mistakes"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["communication-tone"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["conflict-regulation"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["planning", "task-load"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["deadlines"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["starting-work"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["rest"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["everyday-rhythm"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["change"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["work-style"] },
];

const CONTENT = window.APP_CONTENT || {};
const state = {
  answers: [],
  adminResults: [],
  lastResult: null,
  lang: "uk",
  startedAt: Date.now(),
  typeFilter: "all",
  typeSearch: "",
  activeType: "",
  adminCardVisible: false,
  adminAccessOpen: false,
  demoRunId: 0,
  demoRunning: false,
  compatibility: {
    typeA: "",
    typeB: "",
    context: "friendship",
    result: null,
  },
};

const E = (id) => document.getElementById(id);
const focusableSelector = "a[href], button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex='-1'])";
let activeModal = null;
let inlineNoticeTimer = null;
let adminPopoverHideTimer = null;

function safeGetStorage(key) {
  try {
    return window.localStorage.getItem(key);
  } catch (_) {
    return null;
  }
}

function safeSetStorage(key, value) {
  try {
    window.localStorage.setItem(key, value);
    return true;
  } catch (_) {
    return false;
  }
}

function safeRemoveStorage(key) {
  try {
    window.localStorage.removeItem(key);
  } catch (_) {}
}

function esc(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function template(value, params = {}) {
  return String(value ?? "").replace(/\{(\w+)\}/g, (_, key) => String(params[key] ?? ""));
}

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

function formatDate(value) {
  return new Date(value).toLocaleDateString(getContent().locale || "uk-UA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatDuration(seconds = 0) {
  const safe = Math.max(0, Number(seconds) || 0);
  const minutes = Math.floor(safe / 60);
  const rest = safe % 60;
  if (state.lang === "en") return minutes <= 0 ? `${rest}s` : `${minutes}m ${String(rest).padStart(2, "0")}s`;
  return minutes <= 0 ? `${rest} \u0441` : `${minutes} \u0445\u0432 ${String(rest).padStart(2, "0")} \u0441`;
}

function ensureToastRoot() {
  let root = E("uiToast");
  if (root) return root;
  root = document.createElement("div");
  root.id = "uiToast";
  root.className = "soft-toast-host";
  root.setAttribute("role", "status");
  root.setAttribute("aria-live", "polite");
  document.body.appendChild(root);
  return root;
}

function toastCloseLabel() {
  const labels = {
    uk: "Закрити повідомлення",
    ru: "Закрыть сообщение",
    en: "Close notification",
  };
  return t("ui.notices.toastClose", labels[state.lang] || labels.en);
}

function showToast(message, options = {}) {
  const { title = t("ui.notices.message"), tone = "info", duration = 2800 } = options;
  const root = ensureToastRoot();
  const toast = document.createElement("div");
  toast.className = `soft-toast soft-toast--${tone}`;
  toast.innerHTML = `
      <span aria-hidden="true" class="soft-toast__icon">${tone === "error" ? "!" : "i"}</span>
      <div class="soft-toast__copy"><strong>${esc(title)}</strong><span>${esc(message)}</span></div>
      <button type="button" class="soft-toast__close" aria-label="${esc(toastCloseLabel())}">
        <span aria-hidden="true">&times;</span>
      </button>`;
  root.appendChild(toast);

  let hidden = false;
  const hideToast = () => {
    if (hidden) return;
    hidden = true;
    toast.classList.remove("visible");
    setTimeout(() => {
      toast.remove();
      if (!root.children.length) root.classList.remove("visible");
    }, 220);
  };

  toast.querySelector(".soft-toast__close")?.addEventListener("click", hideToast, { once: true });
  requestAnimationFrame(() => root.classList.add("visible"));
  requestAnimationFrame(() => toast.classList.add("visible"));
  setTimeout(hideToast, duration);
}

function showInlineNotice({ title, message, tone = "info", duration = 3600 }) {
  const notice = E("adminNotice");
  if (!notice) return;
  notice.className = `soft-notice soft-notice--${tone}`;
  notice.innerHTML = `
    <span aria-hidden="true" class="soft-notice__icon">${tone === "error" ? "!" : tone === "success" ? "\u2713" : "i"}</span>
    <div class="soft-notice__copy"><strong>${esc(title)}</strong><span>${esc(message)}</span></div>`;
  requestAnimationFrame(() => notice.classList.add("visible"));
  if (inlineNoticeTimer) clearTimeout(inlineNoticeTimer);
  if (duration > 0) inlineNoticeTimer = setTimeout(() => notice.classList.remove("visible"), duration);
}

function setInputInvalidState(input, invalid) {
  if (!(input instanceof HTMLElement)) return;
  input.classList.toggle("input-invalid", invalid);
  input.setAttribute("aria-invalid", invalid ? "true" : "false");
  if (!invalid) return;
  input.classList.remove("input-bump");
  void input.offsetWidth;
  input.classList.add("input-bump");
}

function getFocusable(container) {
  return [...container.querySelectorAll(focusableSelector)].filter((el) => el.offsetParent !== null || el === document.activeElement);
}

function closeActiveModal(value) {
  if (!activeModal) return;
  const { backdrop, resolve, previousFocus, keyHandler, cleanup } = activeModal;
  activeModal = null;
  document.removeEventListener("keydown", keyHandler, true);
  backdrop.classList.remove("visible");
  setTimeout(() => {
    backdrop.remove();
    if (typeof cleanup === "function") cleanup();
  }, 220);
  if (previousFocus instanceof HTMLElement) previousFocus.focus({ preventScroll: true });
  resolve(value);
}

function openModal({ title, copy, confirmLabel, cancelLabel, input = false, initialValue = "" }) {
  if (activeModal) closeActiveModal(null);
  return new Promise((resolve) => {
    const previousFocus = document.activeElement;
    const backdrop = document.createElement("div");
    backdrop.className = "modal-backdrop";
    backdrop.innerHTML = `
      <div class="modal-panel" role="dialog" aria-modal="true" aria-labelledby="modalTitle">
        <div class="modal-head"><h3 id="modalTitle">${esc(title)}</h3></div>
        <p class="modal-copy">${esc(copy || "")}</p>
        ${input ? `<input id="modalInput" class="modal-input" type="text" autocomplete="name" maxlength="64" value="${esc(initialValue)}" />` : ""}
        <div class="modal-actions">
          <button type="button" id="modalCancel" class="modal-btn">${esc(cancelLabel || t("ui.modal.cancel"))}</button>
          <button type="button" id="modalConfirm" class="modal-btn modal-btn--primary">${esc(confirmLabel || t("ui.modal.confirm"))}</button>
        </div>
      </div>`;

    const keyHandler = (event) => {
      if (!activeModal || activeModal.backdrop !== backdrop) return;
      if (event.key === "Escape") {
        event.preventDefault();
        closeActiveModal(input ? "" : false);
        return;
      }
      if (event.key !== "Tab") return;
      const focusable = getFocusable(backdrop);
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.body.appendChild(backdrop);
    activeModal = { backdrop, resolve, previousFocus, keyHandler };
    document.addEventListener("keydown", keyHandler, true);
    E("modalCancel")?.addEventListener("click", () => closeActiveModal(input ? "" : false));
    E("modalConfirm")?.addEventListener("click", () => {
      const value = input ? E("modalInput")?.value.trim() || "" : true;
      closeActiveModal(value);
    });
    backdrop.addEventListener("mousedown", (event) => {
      if (event.target === backdrop) closeActiveModal(input ? "" : false);
    });
    requestAnimationFrame(() => backdrop.classList.add("visible"));
    setTimeout(() => (E("modalInput") || E("modalConfirm"))?.focus(), 0);
  });
}

function askName() {
  return openModal({
    title: t("ui.modal.nameTitle"),
    copy: t("ui.modal.nameCopy"),
    input: true,
    initialValue: E("personName")?.value.trim() || "",
    confirmLabel: t("ui.modal.showResult"),
  });
}

function askConfirm(message) {
  return openModal({ title: message, copy: "", confirmLabel: t("ui.modal.confirm") });
}
function getTypeData(code) {
  const key = String(code || "").toUpperCase();
  const base = window.TYPES_DATA?.[key] || null;
  const override = getContent().types?.[key] || {};
  if (!base && !Object.keys(override).length) return null;
  return {
    ...(base || {}),
    ...override,
    code: key,
    socioCode: override.socioCode || base?.socioCode || "",
    socioName: override.socioName || base?.socioName || "",
    quadra: override.quadra || base?.quadra || "",
    sections: override.sections || base?.sections || [],
    source: override.fullProfile ? false : (override.source === undefined ? base?.source : override.source),
    summary: override.summary || base?.summary || null,
  };
}

function getTypeName(code) {
  return getTypeData(code)?.name || code;
}

function getTypeTagline(code) {
  return getTypeData(code)?.tagline || "";
}

function allTypes() {
  return TYPE_GRID_ORDER.map(getTypeData).filter(Boolean);
}

function resultInsightsText(section, key, fallback = "") {
  const langText = window.RESULT_INSIGHTS?.text?.[state.lang] || window.RESULT_INSIGHTS?.text?.en || {};
  return key ? langText?.[section]?.[key] ?? fallback : langText?.[section] ?? fallback;
}

function getSection(type, key) {
  return (type?.sections || []).find((section) => section.key === key) || null;
}

function sectionPreview(type, key) {
  const section = getSection(type, key);
  return section?.paragraphs?.[0] || section?.items?.[0] || getTypeTagline(type?.code) || "";
}

function typePermalink(typeCode) {
  const code = String(typeCode || "").toUpperCase();
  if (!TYPE_GRID_ORDER.includes(code)) return `${location.origin}${location.pathname}`;
  const params = new URLSearchParams();
  params.set("tab", "types");
  params.set("type", code);
  return `${location.origin}${location.pathname}?${params}`;
}

function shareUrl(typeCode = "") {
  if (typeCode) return typePermalink(typeCode);
  return `${location.origin}${location.pathname}`;
}

function shareDisplayUrl(typeCode = "") {
  return shareUrl(typeCode);
}

function shareCardAssetPath(type) {
  return SHARE_CARD_ASSETS[type] || "";
}

async function loadStaticShareCard(type) {
  const path = shareCardAssetPath(type);
  if (!path) return null;
  try {
    const response = await fetch(path, { cache: "force-cache" });
    if (!response.ok) return null;
    const blob = await response.blob();
    if (!blob?.type?.startsWith("image/")) return null;
    const previewUrl = URL.createObjectURL(blob);
    return {
      blob,
      previewUrl,
      source: "static",
      cleanup: () => URL.revokeObjectURL(previewUrl),
    };
  } catch (_) {
    return null;
  }
}

function sharePayload(typeCode) {
  const data = getTypeData(typeCode);
  if (!data) return null;
  const traits = (data.summary?.strengths || []).slice(0, 3);
  return {
    type: data.code,
    name: data.name,
    socioCode: data.socioCode,
    socioName: data.socioName,
    shareText: data.shareDeepText || data.shareText || data.summary?.shortSummary || data.tagline,
    copyText: data.shareText || data.shareDeepText || data.summary?.shortSummary || data.tagline,
    traits,
    title: `${data.code} - ${data.name}`,
    url: shareUrl(typeCode),
    displayUrl: shareDisplayUrl(typeCode),
  };
}

function shareTextFor(typeCode) {
  const payload = sharePayload(typeCode);
  if (!payload) return "";
  const socio = payload.socioCode ? ` (${t("ui.shareCard.socionics")}: ${payload.socioCode})` : "";
  return `${t("ui.share.prefix")} ${payload.type} - ${payload.name}${socio}. ${payload.copyText || payload.shareText} ${t("ui.share.cta")} ${payload.url}`;
}

function compatibilityEngine() {
  return window.COMPATIBILITY_ENGINE || null;
}

function compatibilityContexts() {
  const engineContexts = compatibilityEngine()?.contexts || [];
  return engineContexts.length ? engineContexts : COMPATIBILITY_CONTEXTS;
}

function typeSelectOptions(selected = "") {
  const placeholder = `<option value="">${esc(t("ui.compatibility.selectPlaceholder", "Choose type"))}</option>`;
  const options = TYPE_GRID_ORDER.map((code) => {
    const data = getTypeData(code);
    const label = data ? `${code} - ${data.name}` : code;
    return `<option value="${esc(code)}" ${selected === code ? "selected" : ""}>${esc(label)}</option>`;
  }).join("");
  return `${placeholder}${options}`;
}

function setCompatibilityState(patch = {}) {
  const next = { ...state.compatibility, ...patch };
  next.typeA = TYPE_GRID_ORDER.includes(next.typeA) ? next.typeA : "";
  next.typeB = TYPE_GRID_ORDER.includes(next.typeB) ? next.typeB : "";
  next.context = compatibilityContexts().includes(next.context) ? next.context : "friendship";
  state.compatibility = next;
}

function buildCompatibilityResult() {
  const engine = compatibilityEngine();
  const typeA = getTypeData(state.compatibility.typeA);
  const typeB = getTypeData(state.compatibility.typeB);
  if (!engine || !typeA || !typeB) {
    state.compatibility.result = null;
    return null;
  }
  state.compatibility.result = engine.analyze({
    typeA,
    typeB,
    context: state.compatibility.context,
    lang: state.lang,
    url: shareUrl(),
  });
  return state.compatibility.result;
}

function renderCompatibilityContextButtons() {
  const contexts = compatibilityContexts();
  return contexts.map((context) => {
    const active = state.compatibility.context === context;
    return `<button type="button" class="compatibility-context ${active ? "active" : ""}" data-compat-context="${esc(context)}" aria-pressed="${active}">${esc(t(`ui.compatibility.contexts.${context}`, context))}</button>`;
  }).join("");
}

function renderCompatibilityList(title, items) {
  return `<section class="compatibility-card">
    <h3>${esc(title)}</h3>
    <ul>${(items || []).map((item) => `<li>${esc(item)}</li>`).join("")}</ul>
  </section>`;
}

function renderCompatibilityScales(scales) {
  return `<section class="compatibility-scales" aria-label="${esc(t("ui.compatibility.scalesTitle", "Mini scales"))}">
    <h3>${esc(t("ui.compatibility.scalesTitle", "Mini scales"))}</h3>
    ${(scales || []).map((scale) => `<div class="compatibility-scale">
      <div class="compatibility-scale__top"><span>${esc(scale.label)}</span><strong>${esc(scale.value)}%</strong></div>
      <div class="compatibility-scale__bar" aria-hidden="true"><span style="width:${esc(scale.value)}%"></span></div>
    </div>`).join("")}
  </section>`;
}

function renderCompatibilityResult(result) {
  if (!result) {
    return `<section class="compatibility-empty">
      <h3>${esc(t("ui.compatibility.emptyTitle", "Start with two types"))}</h3>
      <p>${esc(t("ui.compatibility.emptyCopy", "Choose two types and a context to see interaction dynamics."))}</p>
    </section>`;
  }
  return `<section class="compatibility-result" aria-label="${esc(t("ui.compatibility.resultLabel", "Comparison result"))}">
    <div class="compatibility-result__head">
      <div>
        <div class="compatibility-result__pair">${esc(result.pair)}</div>
        <p>${esc(result.contextLabel)}</p>
      </div>
      <div class="compatibility-score"><strong>${esc(result.score)}%</strong><span>${esc(result.scoreLabel || t("ui.compatibility.scoreSuffix", "interaction potential"))}</span></div>
    </div>
    <p class="compatibility-summary">${esc(result.summary)}</p>
    <div class="compatibility-result__grid">
      ${renderCompatibilityList(t("ui.compatibility.worksTitle", "What can work well"), result.works)}
      ${renderCompatibilityList(t("ui.compatibility.tensionsTitle", "Where tension may appear"), result.tensions)}
      ${renderCompatibilityList(t("ui.compatibility.tipsTitle", "How to interact better"), result.tips)}
      ${renderCompatibilityScales(result.scales)}
    </div>
    <div class="compatibility-actions">
      <button type="button" class="result-type-btn result-type-btn--share" data-copy-compatibility>${esc(t("ui.compatibility.copy", "Copy result"))}</button>
    </div>
  </section>`;
}

function renderCompatibility() {
  const section = E("compatibilitySection");
  if (!section) return;
  setCompatibilityState();
  const result = state.compatibility.typeA && state.compatibility.typeB ? buildCompatibilityResult() : null;
  section.innerHTML = `
    <div class="compatibility-layout">
      <section class="card compatibility-tools" aria-label="${esc(t("ui.compatibility.toolsLabel", "Comparison"))}">
        <div class="compatibility-tools__intro">
          <div class="types-tools__label">${esc(t("ui.compatibility.toolsLabel", "Comparison"))}</div>
          <h2>${esc(t("ui.compatibility.toolsTitle", "Choose two types"))}</h2>
          <p>${esc(t("ui.compatibility.toolsCopy", "The score is a heuristic, not a verdict."))}</p>
        </div>
        <div class="compatibility-form">
          <div class="field">
            <label for="compatTypeA">${esc(t("ui.compatibility.firstType", "First type"))}</label>
            <select id="compatTypeA">${typeSelectOptions(state.compatibility.typeA)}</select>
          </div>
          <div class="field">
            <label for="compatTypeB">${esc(t("ui.compatibility.secondType", "Second type"))}</label>
            <select id="compatTypeB">${typeSelectOptions(state.compatibility.typeB)}</select>
          </div>
        </div>
        <div class="compatibility-context-wrap">
          <div class="compatibility-context-label">${esc(t("ui.compatibility.contextLabel", "Context"))}</div>
          <div class="compatibility-contexts" role="group" aria-label="${esc(t("ui.compatibility.contextLabel", "Context"))}">${renderCompatibilityContextButtons()}</div>
        </div>
        <button type="button" class="btn primary compatibility-submit" data-run-compatibility>${esc(t("ui.compatibility.compare", "Compare"))}</button>
      </section>
      ${renderCompatibilityResult(result)}
    </div>`;
}

function runCompatibilityFromControls(options = {}) {
  const typeA = E("compatTypeA")?.value || "";
  const typeB = E("compatTypeB")?.value || "";
  setCompatibilityState({ typeA, typeB });
  renderCompatibility();
  if (options.updateHistory !== false && !E("compatibilitySection")?.hidden) updateRoute("compatibility");
}

function openCompatibilityWithType(typeCode, options = {}) {
  setCompatibilityState({ typeA: TYPE_GRID_ORDER.includes(typeCode) ? typeCode : state.compatibility.typeA });
  setTab("compatibility", { updateHistory: options.updateHistory !== false });
  renderCompatibility();
  setTimeout(() => E("compatTypeB")?.focus({ preventScroll: true }), 0);
}

function copyCompatibilityResult() {
  const result = buildCompatibilityResult();
  if (!result?.copyText) return;
  if (!navigator.clipboard?.writeText) {
    showToast(result.copyText, { title: t("ui.compatibility.copy", "Copy result"), duration: 5200 });
    return;
  }
  navigator.clipboard.writeText(result.copyText)
    .then(() => showToast(t("ui.compatibility.copied", "Comparison copied."), { title: t("ui.notices.done"), duration: 2200 }))
    .catch(() => showToast(result.copyText, { title: t("ui.compatibility.copy", "Copy result"), duration: 5200 }));
}

function initScrollTopButton() {
  if (E("scrollTopBtn")) return;
  const button = document.createElement("button");
  button.id = "scrollTopBtn";
  button.type = "button";
  button.className = "scroll-top-btn";
  button.setAttribute("aria-label", "Scroll to top");
  button.innerHTML = `<span aria-hidden="true">\u2191</span>`;
  document.body.appendChild(button);
  button.addEventListener("click", () => window.scrollTo({ top: 0, behavior: "smooth" }));
  window.addEventListener("scroll", () => button.classList.toggle("visible", window.scrollY > 520), { passive: true });
}

function loadDraft() {
  const raw = safeGetStorage(DRAFT_KEY);
  if (!raw) return;
  try {
    const draft = JSON.parse(raw);
    if (!draft || !Array.isArray(draft.answers)) return;
    const total = totalQuestions();
    state.answers = Array(total).fill(null).map((_, index) => draft.answers[index] || null);
    state.startedAt = Number(draft.startedAt) || Date.now();
    const input = E("personName");
    if (input instanceof HTMLInputElement) input.value = draft.name || "";
  } catch (_) {
    safeRemoveStorage(DRAFT_KEY);
  }
}

function saveDraft() {
  const name = E("personName")?.value.trim() || "";
  const hasProgress = name || state.answers.some(Boolean);
  if (!hasProgress) {
    safeRemoveStorage(DRAFT_KEY);
    return;
  }
  safeSetStorage(DRAFT_KEY, JSON.stringify({ name, answers: state.answers, startedAt: state.startedAt, savedAt: Date.now() }));
}

function clearDraft() {
  safeRemoveStorage(DRAFT_KEY);
}

function renderQuiz() {
  const form = E("quizForm");
  if (!form) return;
  const items = questions();
  form.innerHTML = items.map((question, index) => {
    const selected = state.answers[index];
    const leftSelected = selected === question.codeLeft;
    const rightSelected = selected === question.codeRight;
    return `
      <section class="question fade-in" data-question="${index}">
        <h3>${index + 1}. ${esc(question.text)}</h3>
        <div class="options">
          <button type="button" class="option ${leftSelected ? "selected" : ""}" data-q="${index}" data-value="${esc(question.codeLeft)}" aria-pressed="${leftSelected}">
            <strong>${esc(question.left)}</strong>
          </button>
          <button type="button" class="option ${rightSelected ? "selected" : ""}" data-q="${index}" data-value="${esc(question.codeRight)}" aria-pressed="${rightSelected}">
            <strong>${esc(question.right)}</strong>
          </button>
        </div>
      </section>`;
  }).join("");
  setText("questionTotal", String(items.length));
  setText("leftCount", String(items.length - state.answers.filter(Boolean).length));
}

function updateAnswerSelection(index, value) {
  const question = document.querySelector(`[data-question="${index}"]`);
  if (!question) return;
  question.querySelectorAll(".option").forEach((option) => {
    const selected = option.dataset.value === value;
    option.classList.toggle("selected", selected);
    option.setAttribute("aria-pressed", String(selected));
  });
}

function updateProgress(options = {}) {
  const { persist = true } = options;
  const total = totalQuestions();
  const done = state.answers.filter(Boolean).length;
  const percent = total ? Math.round((done / total) * 100) : 0;

  setText("progressLabel", t("ui.progress.label", "Answered: {done} / {total}", { done, total }));
  setText("progressSubtitle", t("ui.progress.subtitle"));
  setText("progressPercent", t("ui.progress.percent", "{percent}%", { percent }));
  setText("doneCount", String(done));
  setText("leftCount", String(Math.max(0, total - done)));
  setText("questionTotal", String(total));

  const fill = E("barFill");
  if (fill) {
    fill.style.width = `${percent}%`;
  }
  document.documentElement.style.setProperty("--progress", String(percent));

  if (persist) saveDraft();
}

function validateQuiz() {
  const missing = state.answers.findIndex((answer) => !answer);
  if (missing === -1) return true;
  const question = document.querySelector(`[data-question="${missing}"]`);
  question?.scrollIntoView({ behavior: "smooth", block: "center" });
  question?.classList.add("input-bump");
  setTimeout(() => question?.classList.remove("input-bump"), 360);
  showToast(t("ui.notices.validation"), { title: t("ui.notices.testError"), tone: "error", duration: 3500 });
  return false;
}

function resetQuiz() {
  state.answers = Array(totalQuestions()).fill(null);
  state.startedAt = Date.now();
  state.lastResult = null;
  const name = E("personName");
  if (name instanceof HTMLInputElement) name.value = "";
  E("resultBox")?.classList.add("hidden");
  clearDraft();
  renderQuiz();
  updateProgress({ persist: false });
}

function translateDimension(dim) {
  const key = dim.key || `${dim.leftCode || ""}${dim.rightCode || ""}`;
  const labels = getContent().dimensions?.[key] || {};
  const winnerLabel = dim.winner === dim.leftCode ? (labels.left || dim.leftLabel) : (labels.right || dim.rightLabel);
  return {
    key,
    label: labels.label || dim.label || key,
    leftLabel: labels.left || dim.leftLabel || dim.leftCode,
    rightLabel: labels.right || dim.rightLabel || dim.rightCode,
    winnerLabel,
  };
}

function renderDimensionBreakdown(profile) {
  const dimensions = profile?.dimensions || [];
  if (!dimensions.length) return "";
  return `<div class="result-breakdown"><h3>${esc(t("ui.result.breakdownTitle", t("ui.result.why")))}</h3><div class="dimension-grid">${dimensions.map((dim) => {
    const labels = translateDimension(dim);
    const width = Math.max(0, Math.min(100, dim.percent || 0));
    return `<div class="dimension-card">
      <div class="dimension-card__top"><span>${esc(labels.label)}</span><strong>${esc(dim.winner)} - ${esc(labels.winnerLabel)}</strong></div>
      <div class="dimension-bar" aria-hidden="true"><span style="width:${width}%"></span></div>
      <div class="dimension-card__bottom"><span>${esc(dim.leftCode)} ${dim.leftScore}</span><span>${esc(dim.rightCode)} ${dim.rightScore}</span></div>
    </div>`;
  }).join("")}</div></div>`;
}

function dimensionMargin(dim) {
  return Math.abs(Number(dim?.leftScore || 0) - Number(dim?.rightScore || 0));
}

function nearestDimension(profile) {
  const dimensions = profile?.dimensions || [];
  return dimensions.reduce((nearest, dim) => {
    if (!nearest) return dim;
    return dimensionMargin(dim) < dimensionMargin(nearest) ? dim : nearest;
  }, null);
}

function resultConfidence(profile) {
  const dimensions = profile?.dimensions || [];
  if (!dimensions.length) return null;
  const margins = dimensions.map(dimensionMargin);
  const closeCount = margins.filter((margin) => margin <= 1).length;
  const softCount = margins.filter((margin) => margin <= 3).length;
  const level = closeCount >= 2 ? "close" : (closeCount === 1 || softCount >= 3 ? "medium" : "high");
  return { level, nearest: nearestDimension(profile) };
}

function renderResultConfidence(profile) {
  const confidence = resultConfidence(profile);
  if (!confidence) return "";
  const copy = resultInsightsText("confidence", null, {});
  const nearest = confidence.nearest;
  const labels = nearest ? translateDimension(nearest) : null;
  const levelLabel = copy[confidence.level] || confidence.level;
  const text = copy[`${confidence.level}Text`] || "";
  const closest = nearest && labels
    ? template(copy.closest || "", {
        axis: `${nearest.leftCode}/${nearest.rightCode}`,
        left: nearest.leftCode,
        leftScore: nearest.leftScore,
        right: nearest.rightCode,
        rightScore: nearest.rightScore,
      })
    : "";
  return `<section class="result-confidence result-confidence--${esc(confidence.level)}">
    <div class="result-confidence__top">
      <span>${esc(copy.title || "Result confidence")}</span>
      <strong>${esc(levelLabel)}</strong>
    </div>
    ${text ? `<p>${esc(text)}</p>` : ""}
    ${closest ? `<div class="result-confidence__meta">${esc(closest)}</div>` : ""}
  </section>`;
}

function renderWhyThisType(profile) {
  const dimensions = profile?.dimensions || [];
  if (!dimensions.length) return "";
  const copy = resultInsightsText("why", null, {});
  const cards = dimensions.map((dim) => {
    const labels = translateDimension(dim);
    const axisText = copy.axis?.[dim.winner] || "";
    const winnerLabel = labels.winnerLabel || dim.winner;
    return `<article class="axis-explanation-card">
      <div class="axis-explanation-card__top">
        <span>${esc(labels.key || dim.key)}</span>
        <strong>${esc(dim.winner)} - ${esc(winnerLabel)}</strong>
      </div>
      <p>${esc(axisText)}</p>
      <div class="axis-explanation-card__score">${esc(dim.leftCode)} ${esc(dim.leftScore)} / ${esc(dim.rightCode)} ${esc(dim.rightScore)}</div>
    </article>`;
  }).join("");
  return `<section class="result-why-type">
    <div class="result-section-card__head">
      <h3>${esc(copy.title || t("ui.result.why", "Why this type"))}</h3>
    </div>
    ${copy.intro ? `<p class="result-why-type__intro">${esc(copy.intro)}</p>` : ""}
    <div class="axis-explanation-grid">${cards}</div>
  </section>`;
}

function similarTypesFor(typeCode) {
  const code = String(typeCode || "").toUpperCase();
  if (!TYPE_GRID_ORDER.includes(code)) return [];
  const rules = window.RESULT_INSIGHTS?.similarAxisOrder || [];
  return rules.map((rule) => {
    const letters = code.split("");
    const current = letters[rule.index];
    const next = rule.flips?.[current];
    if (!next) return null;
    letters[rule.index] = next;
    const similarCode = letters.join("");
    return TYPE_GRID_ORDER.includes(similarCode) ? { code: similarCode, targetLetter: next, axis: rule.key } : null;
  }).filter(Boolean);
}

function renderSimilarTypes(typeCode) {
  const copy = resultInsightsText("similar", null, {});
  const diff = copy.diff || {};
  const cards = similarTypesFor(typeCode).map((item) => {
    const type = getTypeData(item.code);
    if (!type) return "";
    return `<button type="button" class="similar-type-card" data-open-type="${esc(type.code)}">
      <span class="similar-type-card__code">${esc(type.code)}</span>
      <strong>${esc(type.name)}</strong>
      <span class="similar-type-card__text">${esc(diff[item.targetLetter] || type.tagline || "")}</span>
      <span class="similar-type-card__link">${esc(copy.open || t("ui.types.open", "Open profile"))}</span>
    </button>`;
  }).filter(Boolean).join("");
  if (!cards) return "";
  return `<section class="similar-types">
    <div class="similar-types__head">
      <h3>${esc(copy.title || "Similar types")}</h3>
      ${copy.intro ? `<p>${esc(copy.intro)}</p>` : ""}
    </div>
    <div class="similar-types-grid">${cards}</div>
  </section>`;
}

function renderTypeSummary(type, options = {}) {
  const summary = type?.summary;
  if (!summary) return "";
  const { compact = false } = options;
  const cardClass = (key) => `summary-card summary-card--${esc(key)}${key === "shortSummary" ? " summary-card--featured" : ""}`;
  const renderCard = ({ key, title, text, items }) => {
    if (!text && !(items || []).length) return "";
    const body = (items || []).length
      ? `<ul>${items.map((item) => `<li>${esc(item)}</li>`).join("")}</ul>`
      : `<p>${esc(text)}</p>`;
    return `<article class="${cardClass(key)}">
      <div class="summary-card__head"><h4>${esc(title)}</h4></div>
      ${body}
    </article>`;
  };
  const fullCards = [
    { key: "image", title: t("ui.result.typeImage", "Короткий образ"), text: summary.image },
    { key: "thinking", title: t("ui.result.thinkingStyle", "Як мислить"), text: summary.thinkingStyle },
    { key: "strengths", title: t("ui.result.strengths"), items: summary.strengths },
    { key: "difficulties", title: t("ui.result.difficulties", "Можливі труднощі"), items: summary.difficulties },
    { key: "growth", title: t("ui.result.growthPoints"), items: summary.growth },
    { key: "work", title: t("ui.result.workStyle"), text: summary.workStyle },
    { key: "communication", title: t("ui.result.communicationStyle"), text: summary.communicationStyle },
    { key: "development", title: t("ui.result.development", "Що допомагає розвиватися"), text: summary.development },
    { key: "shortSummary", title: t("ui.result.shortSummary", "Підсумок"), text: summary.shortSummary },
  ];
  const compactCards = [
    { key: "image", title: t("ui.result.typeImage", "Короткий образ"), text: summary.image },
    { key: "strengths", title: t("ui.result.strengths"), items: (summary.strengths || []).slice(0, 3) },
    { key: "shortSummary", title: t("ui.result.shortSummary", "Підсумок"), text: summary.shortSummary },
  ];
  const cards = compact ? compactCards : fullCards;
  return `<section class="type-summary ${compact ? "type-summary--compact" : ""}" aria-label="${esc(t("ui.result.summaryTitle"))}">
    <h3>${esc(t("ui.result.summaryTitle"))}</h3>
    <div class="type-summary-grid">
      ${cards.map(renderCard).join("")}
    </div>
  </section>`;
}

function renderList(items = []) {
  const visible = items.filter(Boolean).slice(0, 5);
  if (!visible.length) return "";
  return `<ul>${visible.map((item) => `<li>${esc(item)}</li>`).join("")}</ul>`;
}

function renderResultSection({ key, title, text = "", items = [] }) {
  const list = renderList(items);
  if (!text && !list) return "";
  return `<article class="result-section-card result-section-card--${esc(key)}">
    <div class="result-section-card__head"><h3>${esc(title)}</h3></div>
    ${list || `<p>${esc(text)}</p>`}
  </article>`;
}

function renderResultOverview(type, profile = null) {
  if (!type) return "";
  const summary = type.summary || {};
  const socio = type.socioCode ? `<span>${esc(t("ui.types.socionics", "Socionics orientation"))}: ${esc(type.socioCode)}</span>` : "";
  const heroText = type.shareText || summary.shortSummary || type.tagline || "";
  const manifestation = [
    { key: "thinking", title: t("ui.result.thinkingStyle", "Thinking style"), text: summary.thinkingStyle },
    { key: "communication", title: t("ui.result.communicationStyle", t("ui.result.communication", "Communication")), text: summary.communicationStyle || sectionPreview(type, "communication") },
    { key: "work", title: t("ui.result.workStyle", t("ui.result.work", "Work and study")), text: summary.workStyle || sectionPreview(type, "work") },
  ].map(renderResultSection).join("");
  const listCards = [
    { key: "strengths", title: t("ui.result.strengths", "Strengths"), items: summary.strengths },
    { key: "difficulties", title: t("ui.result.difficulties", "Possible difficulties"), items: summary.difficulties || summary.growth },
  ].map(renderResultSection).join("");
  const development = renderResultSection({
    key: "development",
    title: t("ui.result.development", t("ui.result.growth", "What helps")),
    text: summary.development || sectionPreview(type, "growth"),
  });
  const finalSummary = summary.shortSummary || heroText || type.tagline;
  const confidence = renderResultConfidence(profile);
  const whyThisType = renderWhyThisType(profile);
  const similarTypes = renderSimilarTypes(type.code);

  return `<div class="result-overview">
    <section class="result-hero-card">
      <div class="result-hero-card__code">${esc(type.code)}</div>
      <div class="result-hero-card__body">
        <div class="result-hero-card__chips"><span>MBTI</span>${socio}</div>
        <h3>${esc(type.name)}</h3>
        <p>${esc(heroText)}</p>
      </div>
    </section>
    ${confidence}
    ${renderDimensionBreakdown(profile)}
    ${whyThisType}
    ${renderResultSection({ key: "image", title: t("ui.result.typeImage", "Snapshot"), text: summary.image || type.tagline })}
    ${manifestation ? `<section class="result-manifest"><h3>${esc(t("ui.result.manifestTitle", "How it shows up"))}</h3><div class="result-card-grid result-card-grid--three">${manifestation}</div></section>` : ""}
    ${listCards ? `<section class="result-card-grid result-card-grid--two">${listCards}</section>` : ""}
    ${development}
    ${finalSummary ? `<section class="result-final-summary">
      <div class="result-section-card__head"><h3>${esc(t("ui.result.shortSummary", "In short"))}</h3></div>
      <p>${esc(finalSummary)}</p>
    </section>` : ""}
    ${similarTypes}
  </div>`;
}

function renderTelegramCTA() {
  return `<aside class="telegram-cta">
    <div><strong>${esc(t("ui.telegram.title"))}</strong><p>${esc(t("ui.telegram.copy"))}</p></div>
    <a href="${esc(TELEGRAM_URL)}" target="_blank" rel="noreferrer">${esc(t("ui.telegram.link"))}</a>
  </aside>`;
}

function showResult(type, name, profile = null, options = {}) {
  const { scroll = true } = options;
  const data = getTypeData(type);
  state.lastResult = { type, name, profile };
  setText("resultType", type);
  setText("resultTitle", t("ui.result.title", "{name}, your type is {typeName}", { name, typeName: getTypeName(type) }));
  setText("resultDesc", getTypeTagline(type));
  E("resultPanel").innerHTML = `${renderResultOverview(data, profile)}${renderTelegramCTA()}<div class="result-actions"><button type="button" class="result-type-btn" data-open-type="${esc(type)}">${esc(t("ui.result.readProfile", t("ui.result.details")))} &rarr;</button><button type="button" class="result-type-btn" data-compare-my-type="${esc(type)}">${esc(t("ui.result.compare", "Compare my type"))}</button><button type="button" class="result-type-btn result-type-btn--share" data-share-result>${esc(t("ui.result.share", "Share result"))}</button><button type="button" class="result-type-btn result-type-btn--muted" data-copy-result>${esc(t("ui.result.copy"))}</button><button type="button" class="result-type-btn result-type-btn--muted" data-retake>${esc(t("ui.result.retake"))}</button></div>`;
  E("resultBox")?.classList.remove("hidden");
  if (scroll) E("resultBox")?.scrollIntoView({ behavior: prefersReducedMotion() ? "auto" : "smooth", block: "start" });
}

function prefersReducedMotion() {
  return Boolean(window.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches);
}

function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function demoAnswersForType(typeCode) {
  return questions().map((question) => {
    if (typeCode.includes(question.codeLeft)) return question.codeLeft;
    if (typeCode.includes(question.codeRight)) return question.codeRight;
    return question.codeLeft;
  });
}

function demoProfileFromAnswers(answers) {
  const axes = [
    { key: "EI", leftCode: "E", rightCode: "I" },
    { key: "SN", leftCode: "S", rightCode: "N" },
    { key: "TF", leftCode: "T", rightCode: "F" },
    { key: "JP", leftCode: "J", rightCode: "P" },
  ];
  return {
    dimensions: axes.map((axis) => {
      const axisAnswers = answers.filter((answer, index) => QUESTION_METADATA[index]?.axis === axis.key && answer);
      const leftScore = axisAnswers.filter((answer) => answer === axis.leftCode).length;
      const rightScore = axisAnswers.filter((answer) => answer === axis.rightCode).length;
      const total = Math.max(1, leftScore + rightScore);
      const winner = rightScore > leftScore ? axis.rightCode : axis.leftCode;
      return {
        key: axis.key,
        leftCode: axis.leftCode,
        rightCode: axis.rightCode,
        leftScore,
        rightScore,
        winner,
        percent: Math.round((Math.max(leftScore, rightScore) / total) * 100),
      };
    }),
  };
}

function resolveDemoType() {
  const selected = E("demoTypeSelect")?.value || "random";
  if (TYPE_GRID_ORDER.includes(selected)) return selected;
  return TYPE_GRID_ORDER[Math.floor(Math.random() * TYPE_GRID_ORDER.length)] || "INTJ";
}

function demoDisplayName(typeCode) {
  const typed = E("demoName")?.value.trim();
  return typed || `Demo ${typeCode}`;
}

function setDemoRunning(running) {
  state.demoRunning = running;
  const run = E("demoRunBtn");
  const cancel = E("demoCancelBtn");
  if (run) run.disabled = running;
  if (cancel) cancel.hidden = !running;
}

function setDemoState(message = "") {
  const el = E("demoState");
  if (!el) return;
  el.textContent = message;
  el.hidden = !message;
}

async function runDemoAutopass() {
  const typeCode = resolveDemoType();
  const name = demoDisplayName(typeCode);
  const answers = demoAnswersForType(typeCode);
  const previousDraft = safeGetStorage(DRAFT_KEY);
  const runId = state.demoRunId + 1;
  state.demoRunId = runId;
  setDemoRunning(true);
  setDemoState(t("ui.admin.demoRunning", "Filling the test for {type}...", { type: typeCode }));
  setTab("quiz", { updateHistory: false });
  if (E("personName")) E("personName").value = name;
  E("resultBox")?.classList.add("hidden");
  state.answers = Array(totalQuestions()).fill(null);
  renderQuiz();
  updateProgress({ persist: false });

  const reduced = prefersReducedMotion();
  for (let index = 0; index < answers.length; index += 1) {
    if (state.demoRunId !== runId) return;
    state.answers[index] = answers[index];
    updateAnswerSelection(index, answers[index]);
    updateProgress({ persist: false });
    const question = document.querySelector(`.question[data-question="${index}"]`);
    question?.classList.add("demo-autopass-highlight");
    if (!reduced && index % 3 === 0) question?.scrollIntoView({ behavior: "smooth", block: "center" });
    if (!reduced) await wait(58);
    question?.classList.remove("demo-autopass-highlight");
  }

  if (state.demoRunId !== runId) return;
  if (previousDraft) safeSetStorage(DRAFT_KEY, previousDraft);
  else safeRemoveStorage(DRAFT_KEY);
  setDemoRunning(false);
  setDemoState(t("ui.admin.demoDone", "Demo result {type} is ready. It was not saved to statistics.", { type: typeCode }));
  setAdminCardVisible(false);
  if (!reduced) await wait(180);
  showResult(typeCode, name, demoProfileFromAnswers(state.answers));
}

function cancelDemoAutopass() {
  state.demoRunId += 1;
  setDemoRunning(false);
  setDemoState("");
  if (safeGetStorage(DRAFT_KEY)) loadDraft();
  else {
    state.answers = Array(totalQuestions()).fill(null);
    if (E("personName")) E("personName").value = "";
  }
  renderQuiz();
  updateProgress({ persist: false });
}

async function submitQuiz() {
  let name = E("personName")?.value.trim() || "";
  if (!name) {
    name = await askName();
    if (!name) return;
    if (E("personName")) E("personName").value = name;
  }
  if (!validateQuiz()) return;

  const payload = {
    name,
    answers: state.answers.map((answer) => String(answer || "")),
    duration: Math.floor((Date.now() - state.startedAt) / 1000),
  };
  const res = await fetch(API.submit, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  let data = {};
  try { data = await res.json(); } catch (_) {}
  if (!res.ok) {
    showToast(data.error || t("ui.notices.submitFailed"), { title: t("ui.notices.testError"), tone: "error", duration: 3500 });
    return;
  }
  clearDraft();
  showResult(data.type, name, data.profile);
  refreshAdmin();
}

function getFilteredTypes() {
  const query = state.typeSearch.trim().toLowerCase();
  return allTypes().filter((type) => {
    const matchesFilter = state.typeFilter === "all" || type.quadra === state.typeFilter;
    const haystack = [type.code, type.name, type.socioCode, type.socioName, type.quadra, type.tagline].join(" ").toLowerCase();
    return matchesFilter && (!query || haystack.includes(query));
  });
}

function renderTypeCodes(type) {
  return `<span class="type-card__code type-card__code--mbti">${esc(type.code)}</span><span class="type-card__code type-card__code--socionics">${esc(type.socioCode)}</span>`;
}

function renderTypeChips(type) {
  return `<span class="type-chip type-chip--mbti">${esc(type.code)}</span><span class="type-chip type-chip--socionics">${esc(type.socioCode)}</span><span class="type-chip type-chip--quadra">${esc(type.quadra)}</span>`;
}

function renderTypeFilters() {
  const quadras = [...new Set(allTypes().map((type) => type.quadra).filter(Boolean))];
  const filters = [{ value: "all", label: t("ui.types.all") }, ...quadras.map((quadra) => ({ value: quadra, label: quadra }))];
  return filters.map((filter) => `<button type="button" class="type-filter-btn ${state.typeFilter === filter.value ? "active" : ""}" data-type-filter="${esc(filter.value)}" aria-pressed="${state.typeFilter === filter.value}">${esc(filter.label)}</button>`).join("");
}

function renderTypesGrid() {
  state.activeType = "";
  const visibleTypes = getFilteredTypes();
  const cards = visibleTypes.map((type) => `
    <button type="button" class="card type-card" data-type="${esc(type.code)}" aria-label="${esc(t("ui.types.open"))} ${esc(type.code)} ${esc(type.name)}">
      <div class="type-card__meta">${renderTypeCodes(type)}</div>
      <h3 class="type-card__title">${esc(type.name)}</h3>
      <p class="type-card__subtitle">${esc(type.socioName)} \u00b7 ${esc(type.quadra)} ${esc(t("ui.types.quadra"))}</p>
      <p class="type-card__text">${esc(type.tagline)}</p>
    </button>`).join("");

  E("typesSection").innerHTML = `
    <div class="types-layout">
      <section class="card types-tools" aria-label="${esc(t("ui.types.toolsLabel"))}">
        <div>
          <div class="types-tools__label">${esc(t("ui.types.toolsLabel"))}</div>
          <h2>${esc(t("ui.types.toolsTitle"))}</h2>
          <p>${esc(t("ui.types.toolsCopy"))}</p>
        </div>
        <div class="types-search-row">
          <label class="sr-only" for="typeSearch">${esc(t("ui.types.searchLabel"))}</label>
          <input id="typeSearch" type="text" value="${esc(state.typeSearch)}" placeholder="${esc(t("ui.types.searchPlaceholder"))}" autocomplete="off" />
          <span class="types-count">${visibleTypes.length} / ${TYPE_GRID_ORDER.length}</span>
        </div>
        <div class="type-filter-list" role="group" aria-label="${esc(t("ui.types.toolsLabel"))}">${renderTypeFilters()}</div>
      </section>
      ${cards ? `<div class="types-grid">${cards}</div>` : `<div class="card types-empty">${esc(t("ui.types.empty"))}</div>`}
    </div>`;
}
function renderTypeSection(section) {
  const paragraphs = (section.paragraphs || []).map((paragraph) => `<p>${esc(paragraph)}</p>`).join("");
  const items = (section.items || []).length ? `<ul class="type-section__list">${section.items.map((item) => `<li>${esc(item)}</li>`).join("")}</ul>` : "";
  if (!paragraphs && !items) return "";
  return `<article class="card type-section"><h3 class="type-section__title">${esc(section.title)}</h3><div class="type-section__body">${paragraphs}${items}</div></article>`;
}

function renderFullProfile(type) {
  const fullProfile = type?.fullProfile;
  if (!fullProfile?.sections?.length) {
    return `<section class="long-profile long-profile--notice" aria-label="${esc(t("ui.types.fullProfileTitle", "Повний профіль"))}">
      <div class="long-profile__head"><p>${esc(t("ui.types.detailsComing", ""))}</p></div>
    </section>`;
  }
  const sections = fullProfile.sections.map((section, index) => {
    const paragraphs = (section.paragraphs || []).map((paragraph) => `<p>${esc(paragraph)}</p>`).join("");
    const items = (section.items || []).length ? `<ul>${section.items.map((item) => `<li>${esc(item)}</li>`).join("")}</ul>` : "";
    return `<details class="profile-section" ${index < 2 ? "open" : ""}>
      <summary><span>${esc(section.title)}</span></summary>
      <div class="profile-section__body">${paragraphs}${items}</div>
    </details>`;
  }).join("");
  return `<section class="long-profile" id="fullProfile" aria-label="${esc(t("ui.types.fullProfileTitle", "Повний профіль"))}">
    <div class="long-profile__head">
      <div class="eyebrow">${esc(t("ui.types.fullProfileTitle", "Повний профіль"))}</div>
      <h2>${esc(fullProfile.title || t("ui.types.fullProfileTitle", "Повний профіль"))}</h2>
      <p>${esc(fullProfile.lead || t("ui.types.fullProfileLead", ""))}</p>
    </div>
    <div class="profile-sections">${sections}</div>
  </section>`;
}

function renderType(typeCode) {
  const data = getTypeData(typeCode);
  if (!data) {
    renderTypesGrid();
    return;
  }
  state.activeType = data.code;
  const sections = (data.sections || []).map(renderTypeSection).join("");
  const legacySections = sections ? `<div class="type-sections">${sections}</div>` : (!data.fullProfile && state.lang === "uk" ? `<div class="type-sections"><article class="card type-section"><p class="type-empty">${esc(t("ui.types.detailsComing", t("ui.types.translationNote")))}</p></article></div>` : "");
  const sourceMarkup = data.source ? `<p class="type-detail__source">${esc(data.source.note)} <a href="${esc(data.source.url)}" target="_blank" rel="noreferrer">${esc(data.source.label)}</a></p>` : "";
  const translationNote = state.lang === "uk" || data.fullProfile ? "" : `<p class="type-detail__note">${esc(t("ui.types.translationNote"))}</p>`;

  E("typesSection").innerHTML = `
    <div class="type-detail">
      <button type="button" data-back-types class="type-detail__back">${esc(t("ui.types.back"))}</button>
      <article class="card type-hero-card">
        <div class="type-detail__header-badge">${esc(t("ui.types.typeBadge"))} - ${esc(data.code)}</div>
        <div class="type-hero-card__top">
          <div class="type-hero-card__copy">
            <div class="eyebrow">${esc(t("ui.types.socionics"))} \u00b7 ${esc(data.quadra)} ${esc(t("ui.types.quadra"))}</div>
            <h1 class="type-detail__title">${esc(data.name)}</h1>
            <p class="type-detail__subtitle">MBTI ${esc(data.code)} \u00b7 ${esc(t("ui.types.socionics"))} ${esc(data.socioCode)} "${esc(data.socioName)}"</p>
            <p class="type-detail__lead">${esc(data.tagline)}</p>
          </div>
          <div class="type-detail__chips">${renderTypeChips(data)}</div>
        </div>
        ${translationNote}
        ${sourceMarkup}
      </article>
      ${renderTypeSummary(data)}
      ${renderFullProfile(data)}
      ${legacySections}
    </div>`;
  E("typesSection").scrollIntoView({ behavior: "smooth", block: "start" });
}

function openTypeDetail(typeCode, options = {}) {
  setTab("types", { skipRender: true, updateHistory: false });
  renderType(typeCode);
  if (options.updateHistory !== false) updateRoute("types", typeCode);
}

function updateRoute(tab, typeCode = "", replace = false) {
  const params = new URLSearchParams();
  if (tab === "types") params.set("tab", "types");
  if (tab === "compatibility") {
    params.set("tab", "compatibility");
    if (state.compatibility.typeA) params.set("typeA", state.compatibility.typeA);
    if (state.compatibility.typeB) params.set("typeB", state.compatibility.typeB);
    if (state.compatibility.context !== "friendship") params.set("context", state.compatibility.context);
  }
  if (typeCode) params.set("type", typeCode);
  const next = params.toString() ? `${location.pathname}?${params}` : location.pathname;
  if (replace) history.replaceState(null, "", next);
  else history.pushState(null, "", next);
}

function setTab(tab, options = {}) {
  const { skipRender = false, updateHistory = true } = options;
  const activeTab = tab === "types" || tab === "compatibility" ? tab : "quiz";
  const isQuiz = activeTab === "quiz";
  const isTypes = activeTab === "types";
  const isCompatibility = activeTab === "compatibility";
  E("tabQuiz")?.classList.toggle("active", isQuiz);
  E("tabTypes")?.classList.toggle("active", isTypes);
  E("tabCompatibility")?.classList.toggle("active", isCompatibility);
  E("tabQuiz")?.setAttribute("aria-selected", String(isQuiz));
  E("tabTypes")?.setAttribute("aria-selected", String(isTypes));
  E("tabCompatibility")?.setAttribute("aria-selected", String(isCompatibility));
  E("quizSection").hidden = !isQuiz;
  E("typesSection").hidden = !isTypes;
  E("compatibilitySection").hidden = !isCompatibility;
  E("quizHero").hidden = !isQuiz;
  E("typesHero").hidden = !isTypes;
  E("compatibilityHero").hidden = !isCompatibility;
  if (isTypes && !skipRender) renderTypesGrid();
  if (isCompatibility && !skipRender) renderCompatibility();
  if (updateHistory) updateRoute(activeTab);
}

function applyRoute(options = {}) {
  const params = new URLSearchParams(window.location.search);
  const typeCode = params.get("type")?.toUpperCase() || "";
  if (params.get("admin") === "1") setAdminCardVisible(true, { focus: false });
  if (params.get("tab") === "compatibility") {
    setCompatibilityState({
      typeA: params.get("typeA")?.toUpperCase() || "",
      typeB: params.get("typeB")?.toUpperCase() || "",
      context: params.get("context") || "friendship",
    });
    setTab("compatibility", { updateHistory: false });
    if (options.replace) updateRoute("compatibility", "", true);
    return;
  }
  if (params.get("tab") === "types") {
    if (typeCode && getTypeData(typeCode)) openTypeDetail(typeCode, { updateHistory: false });
    else setTab("types", { updateHistory: false });
    if (options.replace) updateRoute("types", typeCode && getTypeData(typeCode) ? typeCode : "", true);
    return;
  }
  if (typeCode && getTypeData(typeCode)) {
    openTypeDetail(typeCode, { updateHistory: false });
    if (options.replace) updateRoute("types", typeCode, true);
    return;
  }
  setTab("quiz", { updateHistory: false });
}

function setAdminCardVisible(visible, options = {}) {
  const card = E("adminCard");
  const access = E("adminAccessBtn");
  if (!card) return;
  state.adminCardVisible = Boolean(visible);
  if (state.adminCardVisible) setAdminAccessPopoverVisible(false);
  card.hidden = !state.adminCardVisible;
  E("quizSection")?.classList.toggle("layout--admin-hidden", !state.adminCardVisible);
  access?.setAttribute("aria-expanded", String(state.adminAccessOpen));
  const label = state.adminAccessOpen ? t("ui.admin.closeAccess", t("ui.admin.closePanel", "Hide admin tools")) : t("ui.admin.openPanel", "Open admin tools");
  access?.setAttribute("aria-label", label);
  access?.setAttribute("title", label);
  if (state.adminCardVisible && options.focus) {
    setTab("quiz", { updateHistory: false });
    setTimeout(() => E("adminPassword")?.focus({ preventScroll: false }), 0);
  }
}

function setAdminAccessPopoverVisible(visible, options = {}) {
  const popover = E("adminAccessPopover");
  const access = E("adminAccessBtn");
  if (!popover || !access) return;
  state.adminAccessOpen = Boolean(visible);
  if (adminPopoverHideTimer) {
    clearTimeout(adminPopoverHideTimer);
    adminPopoverHideTimer = null;
  }
  access.setAttribute("aria-expanded", String(state.adminAccessOpen));
  const label = state.adminAccessOpen ? t("ui.admin.closeAccess", t("ui.admin.closePanel", "Hide admin tools")) : t("ui.admin.openPanel", "Open admin tools");
  access.setAttribute("aria-label", label);
  access.setAttribute("title", label);

  if (state.adminAccessOpen) {
    popover.hidden = false;
    requestAnimationFrame(() => popover.classList.add("visible"));
    if (options.focus) setTimeout(() => E("adminOpenBtn")?.focus({ preventScroll: true }), 0);
    return;
  }

  popover.classList.remove("visible");
  adminPopoverHideTimer = setTimeout(() => {
    if (!state.adminAccessOpen) popover.hidden = true;
  }, prefersReducedMotion() ? 0 : 180);
}

function toggleAdminAccessPopover() {
  setAdminAccessPopoverVisible(!state.adminAccessOpen, { focus: !state.adminAccessOpen });
}

function openAdminPanelFromAccess() {
  setAdminAccessPopoverVisible(false);
  setAdminCardVisible(true, { focus: true });
}

function setAdminState(message = "", tone = "info") {
  const el = E("adminState");
  if (!el) return;
  el.textContent = message;
  el.className = `admin-state admin-state--${tone}`;
  el.hidden = !message;
}

function renderAdminResults() {
  const tbody = E("resultsTable");
  if (!tbody) return;
  const query = E("adminSearch")?.value.trim().toLowerCase() || "";
  const items = state.adminResults.filter((result) => [result.name, result.type, result.id].join(" ").toLowerCase().includes(query));
  setText("statsCounter", String(state.adminResults.length));
  tbody.innerHTML = "";
  if (!state.adminResults.length) {
    setAdminState(t("ui.admin.empty"));
    return;
  }
  if (!items.length) {
    setAdminState(t("ui.admin.notFound"));
    return;
  }
  setAdminState("");
  for (const result of items) {
    const row = document.createElement("tr");
    row.innerHTML = `<td>${esc(result.name)}</td><td><strong class="type-cell">${esc(result.type)}</strong><div class="small">${formatDuration(result.duration)}</div></td><td>${formatDate(result.created)}</td><td><span class="small">${esc(result.id)}</span><button type="button" class="table-action" data-del="${esc(result.id)}">${esc(t("ui.admin.table.delete"))}</button></td>`;
    tbody.appendChild(row);
  }
}

async function refreshAdmin() {
  const panelVisible = E("adminPanel")?.classList.contains("visible");
  if (panelVisible) setAdminState(t("ui.admin.loading"));
  try {
    const res = await fetch(API.results);
    if (res.status === 401) {
      if (panelVisible) setAdminState(t("ui.admin.needLogin"), "error");
      return;
    }
    if (!res.ok) throw new Error(t("ui.admin.needLogin"));
    const data = await res.json();
    state.adminResults = data.results || [];
    renderAdminResults();
  } catch (error) {
    if (panelVisible) setAdminState(error.message || t("ui.admin.needLogin"), "error");
  }
}

async function loginAdmin() {
  const input = E("adminPassword");
  const password = input?.value.trim() || "";
  if (!password) {
    setInputInvalidState(input, true);
    showInlineNotice({ title: t("ui.notices.enterPassword"), message: t("ui.notices.passwordRequired") });
    input?.focus();
    return;
  }
  const res = await fetch(API.login, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ password }) });
  if (!res.ok) {
    if (input) input.value = "";
    setInputInvalidState(input, true);
    showInlineNotice({ title: t("ui.notices.passwordWrong"), message: t("ui.notices.passwordWrongCopy"), tone: "error" });
    input?.focus();
    return;
  }
  setInputInvalidState(input, false);
  if (input) input.value = "";
  E("adminPanel")?.classList.add("visible");
  showInlineNotice({ title: t("ui.notices.done"), message: t("ui.notices.adminOpened"), tone: "success", duration: 2200 });
  refreshAdmin();
}

async function logoutAdmin() {
  await fetch(API.logout, { method: "POST" });
  E("adminPanel")?.classList.remove("visible");
  state.adminResults = [];
  renderAdminResults();
  showInlineNotice({ title: t("ui.notices.done"), message: t("ui.notices.logoutDone"), tone: "success", duration: 2200 });
}

async function clearResults() {
  if (!await askConfirm(t("ui.modal.clearAll"))) return;
  const res = await fetch(API.results, { method: "DELETE" });
  if (!res.ok) {
    showToast(t("ui.notices.exportError"), { title: t("ui.notices.error"), tone: "error" });
    return;
  }
  state.adminResults = [];
  renderAdminResults();
  showToast(t("ui.notices.clearDone"), { title: t("ui.notices.done"), duration: 2200 });
}

async function deleteResult(id) {
  if (!await askConfirm(t("ui.modal.deleteOne"))) return;
  const res = await fetch(`${API.results}/${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!res.ok) {
    showToast(t("ui.notices.exportError"), { title: t("ui.notices.error"), tone: "error" });
    return;
  }
  showToast(t("ui.notices.deleteDone"), { title: t("ui.notices.done"), duration: 2200 });
  refreshAdmin();
}

async function exportResults(format = "csv") {
  setAdminState(format === "json" ? t("ui.admin.exportJson") : t("ui.admin.exportCsv"));
  try {
    const res = await fetch(format === "json" ? `${API.export}?format=json` : API.export);
    if (!res.ok) throw new Error(t("ui.notices.exportError"));
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `mbti-results.${format}`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    setAdminState("");
    showToast(format === "json" ? t("ui.notices.jsonReady") : t("ui.notices.csvReady"), { title: t("ui.notices.done"), duration: 2200 });
  } catch (error) {
    setAdminState(error.message || t("ui.notices.exportError"), "error");
  }
}

function drawRoundRect(ctx, x, y, width, height, radius) {
  const r = Math.min(radius, width / 2, height / 2);
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + width - r, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + r);
  ctx.lineTo(x + width, y + height - r);
  ctx.quadraticCurveTo(x + width, y + height, x + width - r, y + height);
  ctx.lineTo(x + r, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - r);
  ctx.lineTo(x, y + r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.closePath();
}

function drawWrappedText(ctx, text, x, y, maxWidth, lineHeight, maxLines) {
  const words = String(text || "").split(/\s+/).filter(Boolean);
  const lines = [];
  let line = "";
  words.forEach((word) => {
    const test = line ? `${line} ${word}` : word;
    if (ctx.measureText(test).width <= maxWidth || !line) {
      line = test;
      return;
    }
    lines.push(line);
    line = word;
  });
  if (line) lines.push(line);
  const visible = lines.slice(0, maxLines);
  if (lines.length > maxLines && visible.length) {
    let last = visible[visible.length - 1];
    while (last.length && ctx.measureText(`${last}...`).width > maxWidth) last = last.slice(0, -1);
    visible[visible.length - 1] = `${last.trim()}...`;
  }
  visible.forEach((row, index) => ctx.fillText(row, x, y + index * lineHeight));
  return y + visible.length * lineHeight;
}

function drawShareCard(payload) {
  const canvas = document.createElement("canvas");
  canvas.width = 1200;
  canvas.height = 630;
  const ctx = canvas.getContext("2d");
  const bg = ctx.createLinearGradient(0, 0, 1200, 630);
  bg.addColorStop(0, "#090a13");
  bg.addColorStop(0.46, "#111625");
  bg.addColorStop(1, "#06070c");
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, 1200, 630);

  const glowA = ctx.createRadialGradient(210, 120, 20, 210, 120, 430);
  glowA.addColorStop(0, "rgba(201,187,255,.25)");
  glowA.addColorStop(1, "rgba(201,187,255,0)");
  ctx.fillStyle = glowA;
  ctx.fillRect(0, 0, 1200, 630);

  const glowB = ctx.createRadialGradient(980, 520, 20, 980, 520, 390);
  glowB.addColorStop(0, "rgba(244,238,223,.15)");
  glowB.addColorStop(1, "rgba(244,238,223,0)");
  ctx.fillStyle = glowB;
  ctx.fillRect(0, 0, 1200, 630);

  drawRoundRect(ctx, 36, 36, 1128, 558, 34);
  ctx.fillStyle = "rgba(12,15,24,.68)";
  ctx.fill();
  ctx.strokeStyle = "rgba(244,238,223,.18)";
  ctx.lineWidth = 1.5;
  ctx.stroke();

  ctx.fillStyle = "rgba(245,247,255,.72)";
  ctx.font = "700 28px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(t("ui.shareCard.brand"), 76, 96);
  ctx.fillStyle = "rgba(201,187,255,.82)";
  ctx.font = "700 18px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(t("ui.shareCard.title"), 76, 130);

  ctx.fillStyle = "#f8f5ff";
  ctx.font = "900 132px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(payload.type, 76, 260);

  ctx.fillStyle = "#f4eedf";
  ctx.font = "800 40px Inter, Segoe UI, Arial, sans-serif";
  drawWrappedText(ctx, payload.name, 76, 314, 620, 46, 2);

  if (payload.socioCode) {
    drawRoundRect(ctx, 76, 348, 330, 46, 23);
    ctx.fillStyle = "rgba(201,187,255,.12)";
    ctx.fill();
    ctx.strokeStyle = "rgba(201,187,255,.26)";
    ctx.stroke();
    ctx.fillStyle = "rgba(238,232,255,.9)";
    ctx.font = "700 20px Inter, Segoe UI, Arial, sans-serif";
    ctx.fillText(`${t("ui.shareCard.socionics")}: ${payload.socioCode}`, 98, 378);
  }

  ctx.fillStyle = "rgba(232,236,255,.84)";
  ctx.font = "500 30px Inter, Segoe UI, Arial, sans-serif";
  drawWrappedText(ctx, payload.shareText, 76, 452, 710, 42, 3);

  ctx.fillStyle = "rgba(201,187,255,.78)";
  ctx.font = "800 20px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(t("ui.shareCard.traits"), 806, 178);

  let chipY = 212;
  (payload.traits || []).forEach((trait) => {
    drawRoundRect(ctx, 806, chipY, 306, 58, 18);
    ctx.fillStyle = "rgba(255,255,255,.055)";
    ctx.fill();
    ctx.strokeStyle = "rgba(255,255,255,.1)";
    ctx.stroke();
    ctx.fillStyle = "rgba(245,247,255,.86)";
    ctx.font = "700 21px Inter, Segoe UI, Arial, sans-serif";
    drawWrappedText(ctx, trait, 828, chipY + 35, 264, 24, 1);
    chipY += 72;
  });

  drawRoundRect(ctx, 806, 468, 306, 68, 20);
  ctx.fillStyle = "rgba(244,238,223,.12)";
  ctx.fill();
  ctx.strokeStyle = "rgba(244,238,223,.2)";
  ctx.stroke();
  ctx.fillStyle = "#f4eedf";
  ctx.font = "800 22px Inter, Segoe UI, Arial, sans-serif";
  drawWrappedText(ctx, t("ui.shareCard.cta"), 828, 500, 264, 26, 2);

  ctx.fillStyle = "rgba(232,236,255,.58)";
  ctx.font = "700 20px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(payload.displayUrl, 76, 552);

  return canvas;
}

function canvasToBlob(canvas) {
  return new Promise((resolve) => canvas.toBlob(resolve, "image/png", 0.95));
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function openSharePreview(payload, rendered) {
  if (activeModal) closeActiveModal(null);
  const previousFocus = document.activeElement;
  const backdrop = document.createElement("div");
  const canNativeShare = Boolean(navigator.share);
  backdrop.className = "modal-backdrop share-backdrop";
  backdrop.innerHTML = `
    <div class="modal-panel share-modal" role="dialog" aria-modal="true" aria-labelledby="shareModalTitle">
      <div class="modal-head"><h3 id="shareModalTitle">${esc(t("ui.shareCard.preview"))}</h3></div>
      <p class="modal-copy share-modal__copy">${esc(t("ui.shareCard.hint", "The PNG card is ready. Share it, copy the link, or download the image."))}</p>
      <img class="share-card-preview" src="${rendered.previewUrl}" alt="${esc(t("ui.shareCard.preview"))}" data-share-preview data-share-source="${esc(rendered.source || "canvas")}" />
      <div class="modal-actions share-modal__actions">
        ${canNativeShare ? `<button type="button" class="modal-btn modal-btn--primary" data-share-web>${esc(t("ui.shareCard.share"))}</button>` : ""}
        <button type="button" class="modal-btn" data-share-link>${esc(t("ui.shareCard.copyLink", "Copy link"))}</button>
        <button type="button" class="modal-btn" data-share-download>${esc(t("ui.shareCard.download"))}</button>
        <button type="button" class="modal-btn" data-share-close>${esc(t("ui.shareCard.close"))}</button>
      </div>
    </div>`;
  const keyHandler = (event) => {
    if (!activeModal || activeModal.backdrop !== backdrop) return;
    if (event.key === "Escape") {
      event.preventDefault();
      closeActiveModal(false);
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = getFocusable(backdrop);
    if (!focusable.length) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  };
  document.body.appendChild(backdrop);
  activeModal = { backdrop, resolve: () => {}, previousFocus, keyHandler, cleanup: rendered.cleanup };
  document.addEventListener("keydown", keyHandler, true);
  requestAnimationFrame(() => backdrop.classList.add("visible"));
  setTimeout(() => backdrop.querySelector("button")?.focus(), 0);

  backdrop.addEventListener("mousedown", (event) => {
    if (event.target === backdrop) closeActiveModal(false);
  });
  backdrop.addEventListener("click", async (event) => {
    const target = event.target instanceof Element ? event.target : null;
    if (target?.closest("[data-share-close]")) {
      closeActiveModal(false);
      return;
    }
    if (target?.closest("[data-share-download]")) {
      downloadBlob(rendered.blob, `personality-type-${payload.type.toLowerCase()}.png`);
      showToast(t("ui.shareCard.downloadReady"), { title: t("ui.notices.done"), duration: 2200 });
      return;
    }
    if (target?.closest("[data-share-link]")) {
      try {
        if (!navigator.clipboard?.writeText) throw new Error("clipboard unavailable");
        await navigator.clipboard.writeText(payload.url);
        showToast(t("ui.shareCard.linkCopied", t("ui.shareCard.copied")), { title: t("ui.shareCard.copyLink", "Copy link"), duration: 2200 });
      } catch (_) {
        showToast(payload.url, { title: t("ui.shareCard.copyLink", "Copy link"), duration: 4200 });
      }
      return;
    }
    if (target?.closest("[data-share-web]")) {
      try {
        const file = new File([rendered.blob], `personality-type-${payload.type.toLowerCase()}.png`, { type: "image/png" });
        if (navigator.canShare?.({ files: [file] })) {
          await navigator.share({ title: payload.title, text: shareTextFor(payload.type), files: [file] });
        } else {
          await navigator.share({ title: payload.title, text: shareTextFor(payload.type), url: payload.url });
        }
      } catch (error) {
        if (error?.name !== "AbortError") showToast(shareTextFor(payload.type), { title: t("ui.shareCard.copy"), duration: 4200 });
      }
    }
  });
}

async function openShareCard() {
  if (!state.lastResult) return;
  const payload = sharePayload(state.lastResult.type);
  if (!payload) return;
  const staticCard = await loadStaticShareCard(payload.type);
  if (staticCard) {
    openSharePreview(payload, staticCard);
    return;
  }
  const canvas = drawShareCard(payload);
  const blob = await canvasToBlob(canvas);
  if (!blob) {
    showToast(shareTextFor(payload.type), { title: t("ui.shareCard.copy"), duration: 4200 });
    return;
  }
  openSharePreview(payload, { blob, previewUrl: canvas.toDataURL("image/png"), source: "canvas" });
}

function copyResult() {
  if (!state.lastResult) return;
  const text = shareTextFor(state.lastResult.type);
  if (!navigator.clipboard?.writeText) {
    showToast(text, { title: t("ui.result.copy"), duration: 4200 });
    return;
  }
  navigator.clipboard.writeText(text)
    .then(() => showToast(t("ui.notices.copied"), { title: t("ui.notices.done"), duration: 2200 }))
    .catch(() => showToast(text, { title: t("ui.result.copy"), duration: 4200 }));
}

function handleTabKeydown(event) {
  const tabs = [E("tabQuiz"), E("tabTypes"), E("tabCompatibility")].filter(Boolean);
  const currentIndex = tabs.indexOf(document.activeElement);
  if (currentIndex === -1) return;
  let nextIndex = currentIndex;
  if (event.key === "ArrowRight") nextIndex = (currentIndex + 1) % tabs.length;
  else if (event.key === "ArrowLeft") nextIndex = (currentIndex - 1 + tabs.length) % tabs.length;
  else if (event.key === "Home") nextIndex = 0;
  else if (event.key === "End") nextIndex = tabs.length - 1;
  else return;
  event.preventDefault();
  tabs[nextIndex].focus();
  tabs[nextIndex].click();
}

function wireEvents() {
  E("tabQuiz")?.addEventListener("click", () => setTab("quiz"));
  E("tabTypes")?.addEventListener("click", () => setTab("types"));
  E("tabCompatibility")?.addEventListener("click", () => setTab("compatibility"));
  E("tabQuiz")?.addEventListener("keydown", handleTabKeydown);
  E("tabTypes")?.addEventListener("keydown", handleTabKeydown);
  E("tabCompatibility")?.addEventListener("keydown", handleTabKeydown);
  E("adminAccessBtn")?.addEventListener("click", toggleAdminAccessPopover);
  E("adminOpenBtn")?.addEventListener("click", openAdminPanelFromAccess);
  E("submitBtn")?.addEventListener("click", () => submitQuiz().catch((error) => showToast(error.message, { title: t("ui.notices.error"), tone: "error" })));
  E("resetBtn")?.addEventListener("click", resetQuiz);
  E("loginBtn")?.addEventListener("click", () => loginAdmin().catch((error) => showToast(error.message, { title: t("ui.notices.error"), tone: "error" })));
  E("logoutBtn")?.addEventListener("click", () => logoutAdmin().catch((error) => showToast(error.message, { title: t("ui.notices.error"), tone: "error" })));
  E("clearBtn")?.addEventListener("click", () => clearResults().catch((error) => showToast(error.message, { title: t("ui.notices.error"), tone: "error" })));
  E("exportBtn")?.addEventListener("click", () => exportResults("csv"));
  E("exportJsonBtn")?.addEventListener("click", () => exportResults("json"));
  E("demoRunBtn")?.addEventListener("click", () => runDemoAutopass().catch((error) => {
    setDemoRunning(false);
    showToast(error.message || t("ui.notices.error"), { title: t("ui.notices.error"), tone: "error" });
  }));
  E("demoCancelBtn")?.addEventListener("click", cancelDemoAutopass);
  E("personName")?.addEventListener("input", saveDraft);
  E("adminSearch")?.addEventListener("input", renderAdminResults);

  document.querySelectorAll("[data-lang]").forEach((button) => {
    button.addEventListener("click", () => setLanguage(button.dataset.lang));
  });

  E("quizForm")?.addEventListener("click", (event) => {
    const button = event.target instanceof Element ? event.target.closest(".option") : null;
    if (!button) return;
    const index = Number(button.dataset.q);
    const value = button.dataset.value || null;
    state.answers[index] = value;
    updateAnswerSelection(index, value);
    updateProgress();
    button.focus({ preventScroll: true });
  });

  document.addEventListener("input", (event) => {
    const target = event.target instanceof HTMLInputElement ? event.target : null;
    if (target?.id !== "typeSearch") return;
    state.typeSearch = target.value;
    const selection = target.selectionStart || target.value.length;
    renderTypesGrid();
    const next = E("typeSearch");
    next?.focus();
    next?.setSelectionRange(selection, selection);
  });

  document.addEventListener("change", (event) => {
    const target = event.target instanceof HTMLSelectElement ? event.target : null;
    if (target?.id === "compatTypeA") {
      setCompatibilityState({ typeA: target.value });
      runCompatibilityFromControls({ updateHistory: false });
      return;
    }
    if (target?.id === "compatTypeB") {
      setCompatibilityState({ typeB: target.value });
      runCompatibilityFromControls({ updateHistory: false });
    }
  });

  document.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    if (state.adminAccessOpen && target && !target.closest("#adminAccessPopover") && !target.closest("#adminAccessBtn")) {
      setAdminAccessPopoverVisible(false);
    }
    const filter = target?.closest("[data-type-filter]");
    if (filter?.dataset.typeFilter) {
      state.typeFilter = filter.dataset.typeFilter;
      renderTypesGrid();
      return;
    }
    const typeButton = target?.closest("[data-type]");
    if (typeButton?.dataset.type) {
      openTypeDetail(typeButton.dataset.type);
      return;
    }
    const openType = target?.closest("[data-open-type]");
    if (openType?.dataset.openType) {
      openTypeDetail(openType.dataset.openType);
      return;
    }
    const compareMyType = target?.closest("[data-compare-my-type]");
    if (compareMyType?.dataset.compareMyType) {
      openCompatibilityWithType(compareMyType.dataset.compareMyType);
      return;
    }
    const contextButton = target?.closest("[data-compat-context]");
    if (contextButton?.dataset.compatContext) {
      setCompatibilityState({ context: contextButton.dataset.compatContext });
      renderCompatibility();
      if (!E("compatibilitySection")?.hidden) updateRoute("compatibility");
      return;
    }
    if (target?.closest("[data-run-compatibility]")) {
      runCompatibilityFromControls();
      return;
    }
    if (target?.closest("[data-copy-compatibility]")) {
      copyCompatibilityResult();
      return;
    }
    if (target?.closest("[data-back-types]")) {
      renderTypesGrid();
      updateRoute("types");
      return;
    }
    if (target?.closest("[data-retake]")) {
      resetQuiz();
      setTab("quiz");
      E("personName")?.focus();
      return;
    }
    if (target?.closest("[data-copy-result]")) {
      copyResult();
      return;
    }
    if (target?.closest("[data-share-result]")) {
      openShareCard().catch(() => showToast(shareTextFor(state.lastResult?.type), { title: t("ui.shareCard.copy"), duration: 4200 }));
      return;
    }
    const del = target?.closest("[data-del]");
    if (del?.dataset.del) deleteResult(del.dataset.del).catch((error) => showToast(error.message, { title: t("ui.notices.error"), tone: "error" }));
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && state.adminAccessOpen) {
      event.preventDefault();
      setAdminAccessPopoverVisible(false);
      E("adminAccessBtn")?.focus({ preventScroll: true });
    }
  });

  window.addEventListener("popstate", () => applyRoute());
}

function init() {
  state.lang = detectLanguage();
  state.answers = Array(totalQuestions()).fill(null);
  initScrollTopButton();
  setLanguage(state.lang, { persist: false, render: false });
  renderQuiz();
  loadDraft();
  renderQuiz();
  updateProgress({ persist: false });
  wireEvents();
  setAdminCardVisible(false);
  applyRoute({ replace: true });
}

init();

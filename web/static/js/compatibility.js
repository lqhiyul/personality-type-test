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


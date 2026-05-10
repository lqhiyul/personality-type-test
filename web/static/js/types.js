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
  const currentParams = new URLSearchParams(window.location.search);
  if (currentParams.get("admin") === "1") params.set("admin", "1");
  else if (currentParams.get("qa") === "1") params.set("qa", "1");
  if (tab === "types") params.set("tab", "types");
  if (tab === "compatibility") {
    params.set("tab", "compatibility");
    if (state.compatibility.typeA) params.set("typeA", state.compatibility.typeA);
    if (state.compatibility.typeB) params.set("typeB", state.compatibility.typeB);
    if (state.compatibility.context !== "friendship") params.set("context", state.compatibility.context);
    if (state.compatibility.checked && state.compatibility.typeA && state.compatibility.typeB) params.set("checked", "1");
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
  if (E("profileSection")) E("profileSection").hidden = true;
  E("quizHero").hidden = !isQuiz;
  E("typesHero").hidden = !isTypes;
  E("compatibilityHero").hidden = !isCompatibility;
  if (E("profileHero")) E("profileHero").hidden = true;
  if (isTypes && !skipRender) renderTypesGrid();
  if (isCompatibility && !skipRender) renderCompatibility();
  if (updateHistory) updateRoute(activeTab);
}

function applyRoute(options = {}) {
  const params = new URLSearchParams(window.location.search);
  const typeCode = params.get("type")?.toUpperCase() || "";
  const adminRequested = params.get("admin") === "1";
  const qaRequested = params.get("qa") === "1";
  setAdminAccessAvailable(adminRequested || qaRequested);
  if (adminRequested) setAdminCardVisible(true, { focus: false });
  const profileUsername = params.get("profile") || "";
  if (profileUsername && typeof openPublicProfile === "function") {
    openPublicProfile(profileUsername, { updateHistory: false });
    if (options.replace) updatePublicProfileRoute(profileUsername, true);
    return;
  }
  if (params.get("tab") === "compatibility") {
    setCompatibilityState({
      typeA: params.get("typeA")?.toUpperCase() || "",
      typeB: params.get("typeB")?.toUpperCase() || "",
      context: params.get("context") || "friendship",
      checked: params.get("checked") === "1",
      validation: false,
      result: null,
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


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

function dimensionLeftPercent(dim) {
  if (Number.isFinite(Number(dim?.leftPercent))) return Number(dim.leftPercent);
  const left = Number(dim?.leftScore || 0);
  const right = Number(dim?.rightScore || 0);
  const total = left + right;
  return total > 0 ? Math.round((left / total) * 100) : 0;
}

function dimensionRightPercent(dim) {
  if (Number.isFinite(Number(dim?.rightPercent))) return Number(dim.rightPercent);
  const left = Number(dim?.leftScore || 0);
  const right = Number(dim?.rightScore || 0);
  const total = left + right;
  return total > 0 ? Math.round((right / total) * 100) : 0;
}

function dimensionBalanceText(dim, labels) {
  const margin = dimensionMargin(dim);
  if ((dim?.balanceLevel || "") === "balanced" || margin <= 5) {
    return t("ui.slider.balanced", "Balanced");
  }
  const target = dim?.winner === dim?.leftCode ? labels.leftLabel : labels.rightLabel;
  const level = dim?.balanceLevel || (margin <= 15 ? "slight" : (margin <= 30 ? "moderate" : "strong"));
  const key = `${level}Lean`;
  const fallback = {
    slightLean: "Slight lean toward: {label}",
    moderateLean: "Moderate lean toward: {label}",
    strongLean: "Strong lean toward: {label}",
  }[key] || "Lean toward: {label}";
  return t(`ui.slider.${key}`, fallback, { label: target });
}

function renderDimensionBreakdown(profile) {
  const dimensions = profile?.dimensions || [];
  if (!dimensions.length) return "";
  return `<div class="result-breakdown"><h3>${esc(t("ui.result.breakdownTitle", t("ui.result.why")))}</h3><div class="dimension-grid">${dimensions.map((dim) => {
    const labels = translateDimension(dim);
    const leftPercent = Math.max(0, Math.min(100, dimensionLeftPercent(dim)));
    const rightPercent = Math.max(0, Math.min(100, dimensionRightPercent(dim)));
    return `<div class="dimension-card">
      <div class="dimension-card__top"><span>${esc(labels.label)}</span><strong>${esc(dim.winner)} - ${esc(labels.winnerLabel)}</strong></div>
      <div class="dimension-bar dimension-bar--split" aria-hidden="true"><span class="dimension-bar__left" style="width:${leftPercent}%"></span><span class="dimension-bar__right" style="width:${rightPercent}%"></span></div>
      <div class="dimension-card__bottom"><span>${esc(labels.leftLabel)} ${leftPercent}%</span><span>${esc(labels.rightLabel)} ${rightPercent}%</span></div>
      <div class="dimension-card__lean">${esc(dimensionBalanceText(dim, labels))}</div>
    </div>`;
  }).join("")}</div></div>`;
}

function dimensionMargin(dim) {
  if (Number.isFinite(Number(dim?.margin))) return Math.abs(Number(dim.margin));
  if (dim?.leftPercent !== undefined || dim?.rightPercent !== undefined) {
    return Math.abs(dimensionLeftPercent(dim) - dimensionRightPercent(dim));
  }
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
  const closeCount = margins.filter((margin) => margin <= 5).length;
  const softCount = margins.filter((margin) => margin <= 15).length;
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
        leftScore: `${dimensionLeftPercent(nearest)}%`,
        right: nearest.rightCode,
        rightScore: `${dimensionRightPercent(nearest)}%`,
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
      <div class="axis-explanation-card__score">${esc(dim.leftCode)} ${esc(dimensionLeftPercent(dim))}% / ${esc(dim.rightCode)} ${esc(dimensionRightPercent(dim))}%</div>
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

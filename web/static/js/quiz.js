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

function sliderLabel(key, fallback = "", params = {}) {
  return t(`ui.slider.${key}`, fallback, params);
}

function isValidSliderValue(value) {
  return Number.isInteger(value) && value >= SLIDER_MIN && value <= SLIDER_MAX;
}

function clampSliderValue(value) {
  const numeric = Math.round(Number(value));
  if (!Number.isFinite(numeric)) return SLIDER_CENTER;
  return Math.max(SLIDER_MIN, Math.min(SLIDER_MAX, numeric));
}

function normalizeAnswerEntry(entry) {
  if (entry && typeof entry === "object" && !Array.isArray(entry)) {
    const raw = Number(entry.value);
    if (!Number.isFinite(raw)) return null;
    const value = clampSliderValue(raw);
    return { value, touched: Boolean(entry.touched) };
  }
  if (typeof entry === "number" && Number.isFinite(entry) && isValidSliderValue(clampSliderValue(entry))) {
    return { value: clampSliderValue(entry), touched: true };
  }
  return null;
}

function answerState(index) {
  return normalizeAnswerEntry(state.answers[index]) || { value: SLIDER_CENTER, touched: false };
}

function isAnswered(answer) {
  const normalized = normalizeAnswerEntry(answer);
  return Boolean(normalized?.touched);
}

function answeredCount() {
  return state.answers.filter(isAnswered).length;
}

function sliderPercentages(value) {
  const right = clampSliderValue(value);
  return { left: SLIDER_MAX - right, right };
}

function formatSliderPercent(percentages) {
  return `${percentages.left}% / ${percentages.right}%`;
}

function sliderLevel(value) {
  const margin = Math.abs(SLIDER_MAX - clampSliderValue(value) * 2);
  if (margin === 0) return "balanced";
  if (margin <= 5) return "almost";
  if (margin <= 15) return "slight";
  if (margin <= 30) return "moderate";
  return "strong";
}

function sliderDirection(value) {
  const normalized = clampSliderValue(value);
  if (normalized === SLIDER_CENTER) return "balanced";
  return normalized < SLIDER_CENTER ? "left" : "right";
}

function sliderVisualClass(answer) {
  if (!answer.touched) return "";
  const direction = sliderDirection(answer.value);
  if (direction === "balanced") return "slider-balanced";
  return direction === "left" ? "slider-left-active" : "slider-right-active";
}

function getLeanLabel(value) {
  const level = sliderLevel(value);
  if (level === "balanced") return sliderLabel("balanced", "Balanced");
  if (level === "almost") return sliderLabel("almostBalanced", "Almost balanced");
  const side = sliderDirection(value) === "left" ? "Left" : "Right";
  const key = `${level}${side}`;
  const fallback = {
    slightLeft: "Slight lean left",
    slightRight: "Slight lean right",
    moderateLeft: "Moderate lean left",
    moderateRight: "Moderate lean right",
    strongLeft: "Strong lean left",
    strongRight: "Strong lean right",
  }[key] || "Lean selected";
  return sliderLabel(key, fallback);
}

function sliderInterpretation(_question, value) {
  return getLeanLabel(value);
}

function sliderAriaValueText(question, value) {
  const percentages = sliderPercentages(value);
  return sliderLabel(
    "ariaValue",
    "{leftPercent}% toward {leftLabel}, {rightPercent}% toward {rightLabel}. {balance}",
    {
      leftPercent: percentages.left,
      rightPercent: percentages.right,
      leftLabel: question.left,
      rightLabel: question.right,
      balance: sliderInterpretation(question, value),
    },
  );
}

function loadDraft() {
  const raw = safeGetStorage(DRAFT_KEY);
  if (!raw) return;
  try {
    const draft = JSON.parse(raw);
    if (!draft || !Array.isArray(draft.answers)) return;
    const total = totalQuestions();
    state.answers = Array(total).fill(null).map((_, index) => normalizeAnswerEntry(draft.answers[index]));
    state.startedAt = Number(draft.startedAt) || Date.now();
    const input = E("personName");
    if (input instanceof HTMLInputElement) input.value = draft.name || "";
  } catch (_) {
    safeRemoveStorage(DRAFT_KEY);
  }
}

function saveDraft() {
  const name = E("personName")?.value.trim() || "";
  const hasProgress = name || state.answers.some(isAnswered);
  if (!hasProgress) {
    safeRemoveStorage(DRAFT_KEY);
    return;
  }
  safeSetStorage(DRAFT_KEY, JSON.stringify({ name, answers: state.answers, startedAt: state.startedAt, savedAt: Date.now() }));
}

function clearDraft() {
  safeRemoveStorage(DRAFT_KEY);
}

function renderSlider(question, index, answer) {
  const value = answer.value;
  const touched = answer.touched;
  const percentages = sliderPercentages(value);
  const visualClass = sliderVisualClass(answer);
  const isLeftActive = visualClass === "slider-left-active";
  const isRightActive = visualClass === "slider-right-active";
  const isBalanced = visualClass === "slider-balanced";
  const sliderID = `questionSlider${index}`;
  const valueID = `sliderValue${index}`;
  const balanceID = `sliderBalance${index}`;
  const ariaLabel = sliderLabel("ariaLabel", "Question {number}: choose between {leftLabel} and {rightLabel}", {
    number: index + 1,
    leftLabel: question.left,
    rightLabel: question.right,
  });
  return `<div class="question-slider ${touched ? "slider-touched" : "slider-unanswered"} ${visualClass}" data-slider-question="${index}" style="--slider-value:${value}%">
    <div class="slider-options">
      <div class="slider-option slider-option--left ${isLeftActive ? "is-active" : ""} ${isBalanced ? "is-balanced" : ""}">
        <span class="slider-option__side">${esc(sliderLabel("leftSide", "Left"))}</span>
        <strong>${esc(question.left)}</strong>
      </div>
      <div class="slider-option slider-option--right ${isRightActive ? "is-active" : ""} ${isBalanced ? "is-balanced" : ""}">
        <span class="slider-option__side">${esc(sliderLabel("rightSide", "Right"))}</span>
        <strong>${esc(question.right)}</strong>
      </div>
    </div>
    <div class="slider-shell">
      <span class="slider-center-mark" aria-hidden="true"></span>
      <input id="${sliderID}" class="slider-input" type="range" min="${SLIDER_MIN}" max="${SLIDER_MAX}" step="1" value="${value}" data-q="${index}" aria-label="${esc(ariaLabel)}" aria-valuetext="${esc(sliderAriaValueText(question, value))}" aria-describedby="${valueID} ${balanceID}" />
    </div>
    <div class="slider-readout">
      <div class="slider-value" id="${valueID}">${formatSliderPercent(percentages)}</div>
      <button type="button" class="slider-confirm-center ${touched && value === SLIDER_CENTER ? "active" : ""}" data-confirm-center="${index}">${esc(sliderLabel("keepCenter", "Keep 50/50"))}</button>
      <div class="slider-balance" id="${balanceID}">${esc(touched ? sliderInterpretation(question, value) : sliderLabel("unanswered", "Move the slider or keep 50/50"))}</div>
    </div>
  </div>`;
}

function renderQuiz() {
  const form = E("quizForm");
  if (!form) return;
  const items = questions();
  form.innerHTML = items.map((question, index) => {
    const answer = answerState(index);
    const visualClass = sliderVisualClass(answer);
    return `<section class="question fade-in ${answer.touched ? "slider-touched" : "slider-unanswered"} ${visualClass}" data-question="${index}">
      <div class="question-header">
        <span class="question-number">${index + 1}</span>
        <h3>${esc(question.text)}</h3>
      </div>
      ${renderSlider(question, index, answer)}
    </section>`;
  }).join("");
  setText("questionTotal", String(items.length));
  setText("leftCount", String(items.length - answeredCount()));
}

function updateSliderUI(index) {
  const item = questions()[index];
  const question = document.querySelector(`[data-question="${index}"]`);
  if (!item || !question) return;
  const answer = answerState(index);
  const percentages = sliderPercentages(answer.value);
  question.classList.toggle("slider-touched", answer.touched);
  question.classList.toggle("slider-unanswered", !answer.touched);

  const slider = question.querySelector(".question-slider");
  slider?.classList.toggle("slider-touched", answer.touched);
  slider?.classList.toggle("slider-unanswered", !answer.touched);
  slider?.style.setProperty("--slider-value", `${answer.value}%`);
  updateSliderVisualState(question, answer);

  const input = question.querySelector(".slider-input");
  if (input instanceof HTMLInputElement) {
    input.value = String(answer.value);
    input.setAttribute("aria-valuetext", sliderAriaValueText(item, answer.value));
  }
  const center = question.querySelector(".slider-confirm-center");
  center?.classList.toggle("active", answer.touched && answer.value === SLIDER_CENTER);
  const value = question.querySelector(".slider-value");
  const balance = question.querySelector(".slider-balance");
  if (value) value.textContent = formatSliderPercent(percentages);
  if (balance) balance.textContent = answer.touched ? sliderInterpretation(item, answer.value) : sliderLabel("unanswered", "Move the slider or keep 50/50");
}

function updateSliderVisualState(question, answer) {
  const visualClass = sliderVisualClass(answer);
  const visualClasses = ["slider-left-active", "slider-right-active", "slider-balanced"];
  const slider = question.querySelector(".question-slider");
  visualClasses.forEach((className) => {
    question.classList.toggle(className, className === visualClass);
    slider?.classList.toggle(className, className === visualClass);
  });

  const left = question.querySelector(".slider-option--left");
  const right = question.querySelector(".slider-option--right");
  left?.classList.toggle("is-active", visualClass === "slider-left-active");
  right?.classList.toggle("is-active", visualClass === "slider-right-active");
  left?.classList.toggle("is-balanced", visualClass === "slider-balanced");
  right?.classList.toggle("is-balanced", visualClass === "slider-balanced");
}

function setSliderAnswer(index, value, options = {}) {
  const { persist = true } = options;
  state.answers[index] = { value: clampSliderValue(value), touched: true };
  updateSliderUI(index);
  updateProgress({ persist });
}

function updateAnswerSelection(index, value) {
  state.answers[index] = { value: clampSliderValue(value), touched: true };
  updateSliderUI(index);
}

function updateProgress(options = {}) {
  const { persist = true } = options;
  const total = totalQuestions();
  const done = answeredCount();
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
  const missing = state.answers.findIndex((answer) => !isAnswered(answer));
  if (missing === -1) return true;
  const question = document.querySelector(`[data-question="${missing}"]`);
  question?.scrollIntoView({ behavior: prefersReducedMotion() ? "auto" : "smooth", block: "center" });
  question?.classList.add("input-bump");
  setTimeout(() => question?.classList.remove("input-bump"), 360);
  const focusTarget = question?.querySelector(".slider-input, .slider-confirm-center");
  focusTarget?.focus({ preventScroll: true });
  showToast(t("ui.slider.validation", t("ui.notices.validation")), { title: t("ui.notices.testError"), tone: "error", duration: 3500 });
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
    answers: state.answers.map((answer) => answerStateFromEntry(answer).value),
    duration: Math.floor((Date.now() - state.startedAt) / 1000),
  };
  const { response: res, data } = await submitResult(payload);
  if (!res.ok) {
    showToast(data.error || t("ui.notices.submitFailed"), { title: t("ui.notices.testError"), tone: "error", duration: 3500 });
    return;
  }
  clearDraft();
  showResult(data.type, name, data.profile);
  if (data.savedToAccount) {
    showToast(t("ui.auth.profile.saved", "Saved to your account"), { title: t("ui.auth.done", "Done"), duration: 2400 });
    if (typeof loadProfileResults === "function") {
      loadProfileResults({ silent: true }).catch(() => {});
    }
  }
  refreshAdmin();
}

function answerStateFromEntry(entry) {
  return normalizeAnswerEntry(entry) || { value: SLIDER_CENTER, touched: false };
}

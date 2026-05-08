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
  const { response: res, data } = await submitResult(payload);
  if (!res.ok) {
    showToast(data.error || t("ui.notices.submitFailed"), { title: t("ui.notices.testError"), tone: "error", duration: 3500 });
    return;
  }
  clearDraft();
  showResult(data.type, name, data.profile);
  refreshAdmin();
}

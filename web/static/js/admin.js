function demoAnswersForType(typeCode) {
  return questions().map((question) => {
    if (typeCode.includes(question.codeLeft)) return SLIDER_MIN;
    if (typeCode.includes(question.codeRight)) return SLIDER_MAX;
    return SLIDER_CENTER;
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
    type: "",
    code: "",
    dimensions: axes.map((axis) => {
      const axisAnswers = answers
        .map(answerStateFromEntry)
        .filter((answer, index) => QUESTION_METADATA[index]?.axis === axis.key && answer.touched);
      const leftScore = axisAnswers.reduce((total, answer) => total + (SLIDER_MAX - answer.value), 0);
      const rightScore = axisAnswers.reduce((total, answer) => total + answer.value, 0);
      const total = Math.max(1, leftScore + rightScore);
      const leftPercent = Math.round((leftScore / total) * 100);
      const rightPercent = Math.round((rightScore / total) * 100);
      const winner = rightScore > leftScore ? axis.rightCode : axis.leftCode;
      const percent = Math.max(leftPercent, rightPercent);
      const margin = Math.abs(leftPercent - rightPercent);
      const labels = getContent().dimensions?.[axis.key] || {};
      return {
        key: axis.key,
        label: labels.label || axis.key,
        leftCode: axis.leftCode,
        leftLabel: labels.left || axis.leftCode,
        rightCode: axis.rightCode,
        rightLabel: labels.right || axis.rightCode,
        leftScore,
        leftPercent,
        rightScore,
        rightPercent,
        winner,
        percent,
        margin,
        balanceLevel: margin <= 5 ? "balanced" : (margin <= 15 ? "slight" : (margin <= 30 ? "moderate" : "strong")),
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
    state.answers[index] = { value: answers[index], touched: true };
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

function setAdminAccessAvailable(available) {
  const access = E("adminAccessBtn");
  if (!access) return;
  access.hidden = !available;
  if (!available) {
    setAdminAccessPopoverVisible(false);
    setAdminCardVisible(false);
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
    const res = await fetchAdminResults();
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
  const { response: res, data } = await loginWithPassword(password);
  if (!res.ok) {
    if (input) input.value = "";
    setInputInvalidState(input, true);
    showInlineNotice({ title: t("ui.notices.passwordWrong"), message: data.error || t("ui.notices.passwordWrongCopy"), tone: "error" });
    input?.focus();
    return;
  }
  setInputInvalidState(input, false);
  if (input) input.value = "";
  E("adminPanel")?.classList.add("visible");
  showInlineNotice({ title: t("ui.notices.done"), message: t("ui.notices.adminOpened"), tone: "success", duration: 2200 });
  refreshAdmin();
  if (typeof loadAdminReports === "function") loadAdminReports().catch(() => {});
}

async function logoutAdmin() {
  await logoutRequest();
  E("adminPanel")?.classList.remove("visible");
  state.adminResults = [];
  state.adminReports = [];
  renderAdminResults();
  if (typeof renderAdminReportsPanel === "function") renderAdminReportsPanel();
  showInlineNotice({ title: t("ui.notices.done"), message: t("ui.notices.logoutDone"), tone: "success", duration: 2200 });
}

async function clearResults() {
  if (!await askConfirm(t("ui.modal.clearAll"))) return;
  const res = await deleteAllResults();
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
  const res = await deleteStoredResult(id);
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
    const res = await fetchResultsExport(format);
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

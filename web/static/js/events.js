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
  if (typeof wireAuthEvents === "function") wireAuthEvents();
  if (typeof wireProfileEvents === "function") wireProfileEvents();
  if (typeof wireFriendsEvents === "function") wireFriendsEvents();
  if (typeof wireSafetyEvents === "function") wireSafetyEvents();
  if (typeof wireMessagesEvents === "function") wireMessagesEvents();

  document.querySelectorAll("[data-lang]").forEach((button) => {
    button.addEventListener("click", () => setLanguage(button.dataset.lang));
  });

  E("quizForm")?.addEventListener("click", (event) => {
    const button = event.target instanceof Element ? event.target.closest(".option") : null;
    if (!button) return;
    const index = Number(button.dataset.q);
    const value = normalizeSliderAnswer(button.dataset.value);
    state.answers[index] = value;
    updateAnswerSelection(index, value);
    updateProgress();
    button.focus({ preventScroll: true });
  });

  E("quizForm")?.addEventListener("pointerdown", (event) => {
    const slider = event.target instanceof HTMLInputElement && event.target.matches("[data-answer-slider]") ? event.target : null;
    if (!slider) return;
    const index = Number(slider.dataset.q);
    const value = normalizeSliderAnswer(slider.value);
    state.answers[index] = value;
    updateAnswerSelection(index, value);
    updateProgress();
  });

  document.addEventListener("input", (event) => {
    const target = event.target instanceof HTMLInputElement ? event.target : null;
    if (target?.matches("[data-answer-slider]")) {
      const index = Number(target.dataset.q);
      const value = normalizeSliderAnswer(target.value);
      state.answers[index] = value;
      updateAnswerSelection(index, value);
      updateProgress();
      return;
    }
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
      setCompatibilityState({ typeA: target.value, context: "friendship", checked: false, validation: false, result: null });
      renderCompatibility();
      if (!E("compatibilitySection")?.hidden) updateRoute("compatibility", "", true);
      return;
    }
    if (target?.id === "compatTypeB") {
      setCompatibilityState({ typeB: target.value, context: "friendship", checked: false, validation: false, result: null });
      renderCompatibility();
      if (!E("compatibilitySection")?.hidden) updateRoute("compatibility", "", true);
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
      setCompatibilityState({ context: contextButton.dataset.compatContext, validation: false });
      renderCompatibility();
      if (!E("compatibilitySection")?.hidden) updateRoute("compatibility", "", true);
      return;
    }
    if (target?.closest("[data-run-compatibility]")) {
      runCompatibilityFromControls();
      return;
    }
    const copyCompatibility = target?.closest("[data-copy-compatibility]");
    if (copyCompatibility) {
      copyCompatibilityResult(copyCompatibility.dataset.copyCompatibility || state.compatibility.context);
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

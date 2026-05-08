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

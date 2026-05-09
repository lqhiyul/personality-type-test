function authLabel(key, fallback, params = {}) {
  return t(`ui.auth.${key}`, fallback, params);
}

function profileLabel(key, fallback, params = {}) {
  return t(`ui.auth.profile.${key}`, fallback, params);
}

function setAuthNotice(message = "", tone = "info") {
  const notice = E("authNotice");
  if (!notice) return;
  notice.textContent = message;
  notice.className = `auth-notice auth-notice--${tone}`;
}

function setAuthPanelOpen(open) {
  state.authPanelOpen = Boolean(open);
  const panel = E("accountPanel");
  const button = E("accountBtn");
  if (panel) panel.hidden = !state.authPanelOpen;
  button?.setAttribute("aria-expanded", String(state.authPanelOpen));
  if (state.authPanelOpen) {
    setTimeout(() => {
      const target = state.currentUser ? E("authLogoutBtn") : E(state.authMode === "register" ? "authUsername" : "authLogin");
      target?.focus({ preventScroll: true });
    }, 0);
  }
}

function setAuthMode(mode) {
  state.authMode = mode === "register" ? "register" : "login";
  renderAuthPanel();
}

function applyAuthStaticText() {
  setText("accountPanelEyebrow", authLabel("eyebrow", "Account"));
  setText("accountPanelTitle", authLabel("title", "My account"));
  E("accountCloseBtn")?.setAttribute("aria-label", authLabel("close", "Close account panel"));
  setText("authLoginModeBtn", authLabel("login", "Log in"));
  setText("authRegisterModeBtn", authLabel("register", "Register"));
  setText("authUsernameLabel", authLabel("username", "Username"));
  setText("authLoginLabel", authLabel("emailOrUsername", "Email or username"));
  setText("authEmailLabel", authLabel("email", "Email"));
  setText("authPasswordLabel", authLabel("password", "Password"));
  setText("authLogoutBtn", authLabel("logout", "Log out"));
  setText("viewPublicProfileBtn", profileLabel("viewPublicProfile", "View public profile"));
  setText("copyPublicProfileBtn", profileLabel("copyPublicProfile", "Copy profile link"));
  setText("profilePrimaryLabel", profileLabel("primaryLabel", "Primary type"));
  setText("profileRefreshBtn", profileLabel("refresh", "Refresh"));
  setText("profileResultsTitle", profileLabel("historyTitle", "Result history"));
  renderAuthPanel();
}

function renderAuthPanel() {
  const user = state.currentUser;
  const signedOut = E("accountSignedOut");
  const signedIn = E("accountSignedIn");
  const button = E("accountBtn");
  const label = E("accountBtnLabel");
  const loginMode = E("authLoginModeBtn");
  const registerMode = E("authRegisterModeBtn");
  const usernameField = E("authUsernameField");
  const loginField = E("authLoginField");
  const emailField = E("authEmailField");
  const password = E("authPassword");
  const submit = E("authSubmitBtn");

  if (signedOut) signedOut.hidden = Boolean(user);
  if (signedIn) signedIn.hidden = !user;
  button?.classList.toggle("is-authenticated", Boolean(user));
  if (label) label.textContent = user?.username || authLabel("button", "Account");

  loginMode?.classList.toggle("active", state.authMode === "login");
  registerMode?.classList.toggle("active", state.authMode === "register");
  if (usernameField) usernameField.hidden = state.authMode !== "register";
  if (loginField) loginField.hidden = state.authMode !== "login";
  if (emailField) emailField.hidden = state.authMode !== "register";
  if (password) password.autocomplete = state.authMode === "register" ? "new-password" : "current-password";
  if (submit) submit.textContent = state.authMode === "register" ? authLabel("register", "Register") : authLabel("login", "Log in");

  if (user) {
    setText("accountName", user.displayName || user.username);
    setText("accountEmail", user.email || "");
    const avatar = E("accountAvatar");
    if (avatar) {
      const key = typeof avatarKeyOrDefault === "function" ? avatarKeyOrDefault(user.avatarKey) : "";
      avatar.className = `account-avatar ${key && typeof avatarClass === "function" ? avatarClass(key) : ""}`.trim();
      avatar.textContent = typeof avatarSymbolFor === "function" ? avatarSymbolFor(key, user.username) : (user.username || "A").slice(0, 1).toUpperCase();
    }
    renderProfileResults();
  }
  if (typeof renderPublicProfile === "function" && state.publicProfile && !E("profileSection")?.hidden) renderPublicProfile();
}

function renderProfileResults() {
  const list = E("profileResultsList");
  const latest = E("profileLatestResult");
  const primaryType = E("profilePrimaryType");
  if (!list || !latest || !primaryType) return;

  const results = Array.isArray(state.profileResults) ? state.profileResults : [];
  const primary = results.find((result) => result.isPrimary);
  const newest = results[0];

  primaryType.textContent = primary
    ? `${primary.mbtiType} - ${getTypeName(primary.mbtiType)}`
    : profileLabel("primaryEmpty", "Not selected");

  if (state.profileLoading) {
    latest.textContent = profileLabel("loading", "Loading saved results...");
    list.innerHTML = "";
    return;
  }

  if (!results.length) {
    latest.textContent = profileLabel("emptyLatest", "No saved results yet.");
    list.innerHTML = `<div class="profile-results__empty">${esc(profileLabel("emptyHistory", "Complete the quiz while logged in to save your result here."))}</div>`;
    return;
  }

  latest.innerHTML = `${esc(profileLabel("latest", "Latest"))}: <strong>${esc(newest.mbtiType)}</strong> <span>${esc(formatDate(newest.createdAt))}</span>`;
  list.innerHTML = results.map((result) => {
    const isPrimary = Boolean(result.isPrimary);
    return `<article class="profile-result ${isPrimary ? "profile-result--primary" : ""}">
      <div class="profile-result__main">
        <div class="profile-result__type">${esc(result.mbtiType)}</div>
        <div>
          <strong>${esc(getTypeName(result.mbtiType))}</strong>
          <span>${esc(formatDate(result.createdAt))} - ${esc(formatDuration(result.durationSeconds))}</span>
        </div>
      </div>
      <div class="profile-result__actions">
        ${isPrimary ? `<span class="profile-result__badge">${esc(profileLabel("primary", "Primary"))}</span>` : `<button type="button" class="profile-result__btn" data-profile-primary="${esc(result.id)}">${esc(profileLabel("makePrimary", "Make primary"))}</button>`}
        <button type="button" class="profile-result__btn profile-result__btn--danger" data-profile-delete="${esc(result.id)}">${esc(profileLabel("delete", "Delete"))}</button>
      </div>
    </article>`;
  }).join("");
}

async function loadProfileResults(options = {}) {
  if (!state.currentUser) return;
  const { silent = false } = options;
  state.profileLoading = !silent;
  renderProfileResults();
  try {
    const response = await fetchMyResults();
    if (response.status === 401) {
      state.currentUser = null;
      state.profileResults = [];
      renderAuthPanel();
      return;
    }
    if (!response.ok) throw new Error(profileLabel("loadFailed", "Could not load saved results"));
    const data = await response.json();
    state.profileResults = Array.isArray(data.results) ? data.results : [];
  } finally {
    state.profileLoading = false;
    renderProfileResults();
  }
}

async function initAuth() {
  renderAuthPanel();
  try {
    const response = await fetchCurrentAccount();
    if (response.status === 401) {
      state.currentUser = null;
      renderAuthPanel();
      return;
    }
    if (!response.ok) return;
    state.currentUser = await response.json();
    renderAuthPanel();
    await loadProfileResults({ silent: true });
  } catch (_) {}
}

async function submitAuthForm() {
  const mode = state.authMode;
  const password = E("authPassword")?.value || "";
  setAuthNotice("");

  const payload = mode === "register"
    ? {
        username: E("authUsername")?.value || "",
        email: E("authEmail")?.value || "",
        password,
      }
    : {
        emailOrUsername: E("authLogin")?.value || "",
        password,
      };

  const { response, data } = mode === "register" ? await registerAccount(payload) : await loginAccount(payload);
  if (!response.ok) {
    setAuthNotice(data.error || authLabel("authFailed", "Authentication failed"), "error");
    showToast(data.error || authLabel("authFailed", "Authentication failed"), { title: authLabel("error", "Account error"), tone: "error" });
    return;
  }

  state.currentUser = data;
  state.profileResults = [];
  const passwordInput = E("authPassword");
  if (passwordInput) passwordInput.value = "";
  setAuthNotice(mode === "register" ? authLabel("registered", "Account created") : authLabel("loggedIn", "Logged in"), "success");
  renderAuthPanel();
  await loadProfileResults({ silent: true });
  showToast(mode === "register" ? authLabel("registered", "Account created") : authLabel("loggedIn", "Logged in"), { title: authLabel("done", "Done"), duration: 2200 });
}

async function logoutUserAccount() {
  const response = await logoutAccount();
  if (!response.ok) {
    showToast(authLabel("logoutFailed", "Could not log out"), { title: authLabel("error", "Account error"), tone: "error" });
    return;
  }
  state.currentUser = null;
  state.profileResults = [];
  setAuthNotice(authLabel("loggedOut", "Logged out"), "success");
  renderAuthPanel();
  showToast(authLabel("loggedOut", "Logged out"), { title: authLabel("done", "Done"), duration: 2200 });
}

async function makeProfileResultPrimary(id) {
  const { response, data } = await setPrimaryMyResult(id);
  if (!response.ok) {
    throw new Error(data.error || profileLabel("primaryFailed", "Could not set primary result"));
  }
  await loadProfileResults({ silent: true });
  showToast(profileLabel("primarySaved", "Primary result updated"), { title: authLabel("done", "Done"), duration: 2200 });
}

async function removeProfileResult(id) {
  const confirmed = await askConfirm(profileLabel("deleteConfirm", "Delete this saved result?"));
  if (!confirmed) return;
  const response = await deleteMyResult(id);
  if (!response.ok) {
    let data = {};
    try {
      data = await response.json();
    } catch (_) {}
    throw new Error(data.error || profileLabel("deleteFailed", "Could not delete saved result"));
  }
  state.profileResults = state.profileResults.filter((result) => String(result.id) !== String(id));
  renderProfileResults();
  showToast(profileLabel("deleted", "Saved result deleted"), { title: authLabel("done", "Done"), duration: 2200 });
}

function wireAuthEvents() {
  E("accountBtn")?.addEventListener("click", () => setAuthPanelOpen(!state.authPanelOpen));
  E("accountCloseBtn")?.addEventListener("click", () => setAuthPanelOpen(false));
  E("authLoginModeBtn")?.addEventListener("click", () => setAuthMode("login"));
  E("authRegisterModeBtn")?.addEventListener("click", () => setAuthMode("register"));
  E("viewPublicProfileBtn")?.addEventListener("click", () => {
    if (!state.currentUser?.username || typeof openPublicProfile !== "function") return;
    setAuthPanelOpen(false);
    openPublicProfile(state.currentUser.username);
  });
  E("copyPublicProfileBtn")?.addEventListener("click", () => {
    if (!state.currentUser?.username || typeof copyProfileLink !== "function") return;
    copyProfileLink(state.currentUser.username);
  });
  E("profileRefreshBtn")?.addEventListener("click", () => loadProfileResults().catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" })));
  E("authLogoutBtn")?.addEventListener("click", () => logoutUserAccount().catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" })));
  E("authForm")?.addEventListener("submit", (event) => {
    event.preventDefault();
    submitAuthForm().catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" }));
  });

  E("profileResultsList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const primary = target?.closest("[data-profile-primary]");
    if (primary?.dataset.profilePrimary) {
      makeProfileResultPrimary(primary.dataset.profilePrimary).catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" }));
      return;
    }
    const del = target?.closest("[data-profile-delete]");
    if (del?.dataset.profileDelete) {
      removeProfileResult(del.dataset.profileDelete).catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" }));
    }
  });

  document.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    if (!state.authPanelOpen || !target) return;
    if (target.closest("#accountPanel") || target.closest("#accountBtn")) return;
    setAuthPanelOpen(false);
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape" || !state.authPanelOpen) return;
    setAuthPanelOpen(false);
    E("accountBtn")?.focus({ preventScroll: true });
  });
}

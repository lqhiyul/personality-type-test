function authLabel(key, fallback, params = {}) {
  return t(`ui.auth.${key}`, fallback, params);
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
    setText("accountAvatar", (user.username || "A").slice(0, 1).toUpperCase());
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
  const passwordInput = E("authPassword");
  if (passwordInput) passwordInput.value = "";
  setAuthNotice(mode === "register" ? authLabel("registered", "Account created") : authLabel("loggedIn", "Logged in"), "success");
  renderAuthPanel();
  showToast(mode === "register" ? authLabel("registered", "Account created") : authLabel("loggedIn", "Logged in"), { title: authLabel("done", "Done"), duration: 2200 });
}

async function logoutUserAccount() {
  const response = await logoutAccount();
  if (!response.ok) {
    showToast(authLabel("logoutFailed", "Could not log out"), { title: authLabel("error", "Account error"), tone: "error" });
    return;
  }
  state.currentUser = null;
  setAuthNotice(authLabel("loggedOut", "Logged out"), "success");
  renderAuthPanel();
  showToast(authLabel("loggedOut", "Logged out"), { title: authLabel("done", "Done"), duration: 2200 });
}

function wireAuthEvents() {
  E("accountBtn")?.addEventListener("click", () => setAuthPanelOpen(!state.authPanelOpen));
  E("accountCloseBtn")?.addEventListener("click", () => setAuthPanelOpen(false));
  E("authLoginModeBtn")?.addEventListener("click", () => setAuthMode("login"));
  E("authRegisterModeBtn")?.addEventListener("click", () => setAuthMode("register"));
  E("authLogoutBtn")?.addEventListener("click", () => logoutUserAccount().catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" })));
  E("authForm")?.addEventListener("submit", (event) => {
    event.preventDefault();
    submitAuthForm().catch((error) => showToast(error.message, { title: authLabel("error", "Account error"), tone: "error" }));
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

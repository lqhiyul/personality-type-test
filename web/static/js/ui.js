function formatDate(value) {
  return new Date(value).toLocaleDateString(getContent().locale || "uk-UA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatDuration(seconds = 0) {
  const safe = Math.max(0, Number(seconds) || 0);
  const minutes = Math.floor(safe / 60);
  const rest = safe % 60;
  if (state.lang === "en") return minutes <= 0 ? `${rest}s` : `${minutes}m ${String(rest).padStart(2, "0")}s`;
  return minutes <= 0 ? `${rest} \u0441` : `${minutes} \u0445\u0432 ${String(rest).padStart(2, "0")} \u0441`;
}

function ensureToastRoot() {
  let root = E("uiToast");
  if (root) return root;
  root = document.createElement("div");
  root.id = "uiToast";
  root.className = "soft-toast-host";
  root.setAttribute("role", "status");
  root.setAttribute("aria-live", "polite");
  document.body.appendChild(root);
  return root;
}

function toastCloseLabel() {
  const labels = {
    uk: "Закрити повідомлення",
    ru: "Закрыть сообщение",
    en: "Close notification",
  };
  return t("ui.notices.toastClose", labels[state.lang] || labels.en);
}

function showToast(message, options = {}) {
  const { title = t("ui.notices.message"), tone = "info", duration = 2800 } = options;
  const root = ensureToastRoot();
  const toast = document.createElement("div");
  toast.className = `soft-toast soft-toast--${tone}`;
  toast.innerHTML = `
      <span aria-hidden="true" class="soft-toast__icon">${tone === "error" ? "!" : "i"}</span>
      <div class="soft-toast__copy"><strong>${esc(title)}</strong><span>${esc(message)}</span></div>
      <button type="button" class="soft-toast__close" aria-label="${esc(toastCloseLabel())}">
        <span aria-hidden="true">&times;</span>
      </button>`;
  root.appendChild(toast);

  let hidden = false;
  const hideToast = () => {
    if (hidden) return;
    hidden = true;
    toast.classList.remove("visible");
    setTimeout(() => {
      toast.remove();
      if (!root.children.length) root.classList.remove("visible");
    }, 220);
  };

  toast.querySelector(".soft-toast__close")?.addEventListener("click", hideToast, { once: true });
  requestAnimationFrame(() => root.classList.add("visible"));
  requestAnimationFrame(() => toast.classList.add("visible"));
  setTimeout(hideToast, duration);
}

function showInlineNotice({ title, message, tone = "info", duration = 3600 }) {
  const notice = E("adminNotice");
  if (!notice) return;
  notice.className = `soft-notice soft-notice--${tone}`;
  notice.innerHTML = `
    <span aria-hidden="true" class="soft-notice__icon">${tone === "error" ? "!" : tone === "success" ? "\u2713" : "i"}</span>
    <div class="soft-notice__copy"><strong>${esc(title)}</strong><span>${esc(message)}</span></div>`;
  requestAnimationFrame(() => notice.classList.add("visible"));
  if (inlineNoticeTimer) clearTimeout(inlineNoticeTimer);
  if (duration > 0) inlineNoticeTimer = setTimeout(() => notice.classList.remove("visible"), duration);
}

function setInputInvalidState(input, invalid) {
  if (!(input instanceof HTMLElement)) return;
  input.classList.toggle("input-invalid", invalid);
  input.setAttribute("aria-invalid", invalid ? "true" : "false");
  if (!invalid) return;
  input.classList.remove("input-bump");
  void input.offsetWidth;
  input.classList.add("input-bump");
}

function getFocusable(container) {
  return [...container.querySelectorAll(focusableSelector)].filter((el) => el.offsetParent !== null || el === document.activeElement);
}

function closeActiveModal(value) {
  if (!activeModal) return;
  const { backdrop, resolve, previousFocus, keyHandler, cleanup } = activeModal;
  activeModal = null;
  document.removeEventListener("keydown", keyHandler, true);
  backdrop.classList.remove("visible");
  setTimeout(() => {
    backdrop.remove();
    if (typeof cleanup === "function") cleanup();
  }, 220);
  if (previousFocus instanceof HTMLElement) previousFocus.focus({ preventScroll: true });
  resolve(value);
}

function openModal({ title, copy, confirmLabel, cancelLabel, input = false, initialValue = "" }) {
  if (activeModal) closeActiveModal(null);
  return new Promise((resolve) => {
    const previousFocus = document.activeElement;
    const backdrop = document.createElement("div");
    backdrop.className = "modal-backdrop";
    backdrop.innerHTML = `
      <div class="modal-panel" role="dialog" aria-modal="true" aria-labelledby="modalTitle">
        <div class="modal-head"><h3 id="modalTitle">${esc(title)}</h3></div>
        <p class="modal-copy">${esc(copy || "")}</p>
        ${input ? `<input id="modalInput" class="modal-input" type="text" autocomplete="name" maxlength="64" value="${esc(initialValue)}" />` : ""}
        <div class="modal-actions">
          <button type="button" id="modalCancel" class="modal-btn">${esc(cancelLabel || t("ui.modal.cancel"))}</button>
          <button type="button" id="modalConfirm" class="modal-btn modal-btn--primary">${esc(confirmLabel || t("ui.modal.confirm"))}</button>
        </div>
      </div>`;

    const keyHandler = (event) => {
      if (!activeModal || activeModal.backdrop !== backdrop) return;
      if (event.key === "Escape") {
        event.preventDefault();
        closeActiveModal(input ? "" : false);
        return;
      }
      if (event.key !== "Tab") return;
      const focusable = getFocusable(backdrop);
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.body.appendChild(backdrop);
    activeModal = { backdrop, resolve, previousFocus, keyHandler };
    document.addEventListener("keydown", keyHandler, true);
    E("modalCancel")?.addEventListener("click", () => closeActiveModal(input ? "" : false));
    E("modalConfirm")?.addEventListener("click", () => {
      const value = input ? E("modalInput")?.value.trim() || "" : true;
      closeActiveModal(value);
    });
    backdrop.addEventListener("mousedown", (event) => {
      if (event.target === backdrop) closeActiveModal(input ? "" : false);
    });
    requestAnimationFrame(() => backdrop.classList.add("visible"));
    setTimeout(() => (E("modalInput") || E("modalConfirm"))?.focus(), 0);
  });
}

function askName() {
  return openModal({
    title: t("ui.modal.nameTitle"),
    copy: t("ui.modal.nameCopy"),
    input: true,
    initialValue: E("personName")?.value.trim() || "",
    confirmLabel: t("ui.modal.showResult"),
  });
}

function askConfirm(message) {
  return openModal({ title: message, copy: "", confirmLabel: t("ui.modal.confirm") });
}

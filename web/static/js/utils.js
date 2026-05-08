function safeGetStorage(key) {
  try {
    return window.localStorage.getItem(key);
  } catch (_) {
    return null;
  }
}

function safeSetStorage(key, value) {
  try {
    window.localStorage.setItem(key, value);
    return true;
  } catch (_) {
    return false;
  }
}

function safeRemoveStorage(key) {
  try {
    window.localStorage.removeItem(key);
  } catch (_) {}
}

function esc(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function template(value, params = {}) {
  return String(value ?? "").replace(/\{(\w+)\}/g, (_, key) => String(params[key] ?? ""));
}

function prefersReducedMotion() {
  return Boolean(window.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches);
}

function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

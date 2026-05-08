const API = {
  submit: "/api/submit",
  login: "/api/login",
  logout: "/api/logout",
  results: "/api/results",
  export: "/api/results/export",
};

async function requestJSON(url, options = {}) {
  const { body, headers, ...rest } = options;
  const response = await fetch(url, {
    ...rest,
    headers: { "Content-Type": "application/json", ...(headers || {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  let data = {};
  try {
    data = await response.json();
  } catch (_) {}
  return { response, data };
}

function submitResult(payload) {
  return requestJSON(API.submit, { method: "POST", body: payload });
}

function loginWithPassword(password) {
  return requestJSON(API.login, { method: "POST", body: { password } });
}

function logoutRequest() {
  return fetch(API.logout, { method: "POST" });
}

function fetchAdminResults() {
  return fetch(API.results);
}

function deleteAllResults() {
  return fetch(API.results, { method: "DELETE" });
}

function deleteStoredResult(id) {
  return fetch(`${API.results}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function fetchResultsExport(format = "csv") {
  return fetch(format === "json" ? `${API.export}?format=json` : API.export);
}

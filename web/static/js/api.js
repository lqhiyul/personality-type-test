const API = {
  submit: "/api/submit",
  login: "/api/login",
  logout: "/api/logout",
  results: "/api/results",
  export: "/api/results/export",
  authRegister: "/api/auth/register",
  authLogin: "/api/auth/login",
  authLogout: "/api/auth/logout",
  authMe: "/api/auth/me",
  myResults: "/api/me/results",
  myProfile: "/api/me/profile",
  users: "/api/users",
  profileComments: "/api/profile-comments",
  friends: "/api/friends",
  friendRequest: "/api/friends/request",
  friendRequests: "/api/friends/requests",
  blocks: "/api/blocks",
  reports: "/api/reports",
  adminReports: "/api/admin/reports",
  messagesStart: "/api/messages/start",
  messagesConversations: "/api/messages/conversations",
  messages: "/api/messages",
};

const CSRF_COOKIE_NAME = "csrf_token";
const CSRF_HEADER_NAME = "X-CSRF-Token";

function readCookie(name) {
  const prefix = `${name}=`;
  return document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix))
    ?.slice(prefix.length) || "";
}

function unsafeMethod(method = "GET") {
  return !["GET", "HEAD", "OPTIONS", "TRACE"].includes(String(method).toUpperCase());
}

async function ensureCSRFToken() {
  let token = readCookie(CSRF_COOKIE_NAME);
  if (token) return token;
  await fetch("/healthz", { cache: "no-store", credentials: "same-origin" });
  return readCookie(CSRF_COOKIE_NAME);
}

async function request(url, options = {}) {
  const method = options.method || "GET";
  const headers = { ...(options.headers || {}) };
  if (unsafeMethod(method)) {
    const token = await ensureCSRFToken();
    if (token) headers[CSRF_HEADER_NAME] = token;
  }
  return fetch(url, {
    ...options,
    method,
    credentials: "same-origin",
    headers,
  });
}

async function requestJSON(url, options = {}) {
  const { body, headers, ...rest } = options;
  const response = await request(url, {
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
  return request(API.logout, { method: "POST" });
}

function fetchAdminResults() {
  return request(API.results);
}

function deleteAllResults() {
  return request(API.results, { method: "DELETE" });
}

function deleteStoredResult(id) {
  return request(`${API.results}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function fetchResultsExport(format = "csv") {
  return request(format === "json" ? `${API.export}?format=json` : API.export);
}

function registerAccount(payload) {
  return requestJSON(API.authRegister, { method: "POST", body: payload });
}

function loginAccount(payload) {
  return requestJSON(API.authLogin, { method: "POST", body: payload });
}

function logoutAccount() {
  return request(API.authLogout, { method: "POST" });
}

function fetchCurrentAccount() {
  return request(API.authMe);
}

function fetchMyResults() {
  return request(API.myResults);
}

function setPrimaryMyResult(id) {
  return requestJSON(`${API.myResults}/${encodeURIComponent(id)}/primary`, { method: "POST" });
}

function deleteMyResult(id) {
  return request(`${API.myResults}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function fetchPublicProfile(username) {
  return request(`${API.users}/${encodeURIComponent(username)}`);
}

function fetchPublicProfileComments(username) {
  return request(`${API.users}/${encodeURIComponent(username)}/comments`);
}

function postPublicProfileComment(username, body) {
  return requestJSON(`${API.users}/${encodeURIComponent(username)}/comments`, { method: "POST", body: { body } });
}

function deletePublicProfileComment(id) {
  return request(`${API.profileComments}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function updateMyProfile(payload) {
  return requestJSON(API.myProfile, { method: "PATCH", body: payload });
}

function sendFriendRequest(username) {
  return requestJSON(API.friendRequest, { method: "POST", body: { username } });
}

function fetchFriends() {
  return request(API.friends);
}

function fetchFriendRequests() {
  return request(API.friendRequests);
}

function acceptFriendRequest(id) {
  return requestJSON(`${API.friendRequests}/${encodeURIComponent(id)}/accept`, { method: "POST" });
}

function deleteFriendship(id) {
  return request(`${API.friends}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function fetchBlocks() {
  return request(API.blocks);
}

function blockUser(username) {
  return requestJSON(API.blocks, { method: "POST", body: { username } });
}

function unblockUser(username) {
  return request(`${API.blocks}/${encodeURIComponent(username)}`, { method: "DELETE" });
}

function createReport(payload) {
  return requestJSON(API.reports, { method: "POST", body: payload });
}

function fetchAdminReports(status = "") {
  const suffix = status ? `?status=${encodeURIComponent(status)}` : "";
  return request(`${API.adminReports}${suffix}`);
}

function updateAdminReportStatus(id, status) {
  return requestJSON(`${API.adminReports}/${encodeURIComponent(id)}/status`, { method: "POST", body: { status } });
}

function startMessageConversation(username) {
  return requestJSON(API.messagesStart, { method: "POST", body: { username } });
}

function fetchMessageConversations() {
  return request(API.messagesConversations);
}

function fetchMessageConversation(id) {
  return request(`${API.messagesConversations}/${encodeURIComponent(id)}`);
}

function sendConversationMessage(id, body) {
  return requestJSON(`${API.messagesConversations}/${encodeURIComponent(id)}`, { method: "POST", body: { body } });
}

function deleteConversationMessage(id) {
  return request(`${API.messages}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

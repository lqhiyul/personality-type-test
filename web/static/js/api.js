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
  messagesStart: "/api/messages/start",
  messageConversations: "/api/messages/conversations",
  messages: "/api/messages",
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

function registerAccount(payload) {
  return requestJSON(API.authRegister, { method: "POST", body: payload });
}

function loginAccount(payload) {
  return requestJSON(API.authLogin, { method: "POST", body: payload });
}

function logoutAccount() {
  return fetch(API.authLogout, { method: "POST" });
}

function fetchCurrentAccount() {
  return fetch(API.authMe);
}

function fetchMyResults() {
  return fetch(API.myResults);
}

function setPrimaryMyResult(id) {
  return requestJSON(`${API.myResults}/${encodeURIComponent(id)}/primary`, { method: "POST" });
}

function deleteMyResult(id) {
  return fetch(`${API.myResults}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function fetchPublicProfile(username) {
  return fetch(`${API.users}/${encodeURIComponent(username)}`);
}

function fetchPublicProfileComments(username) {
  return fetch(`${API.users}/${encodeURIComponent(username)}/comments`);
}

function postPublicProfileComment(username, body) {
  return requestJSON(`${API.users}/${encodeURIComponent(username)}/comments`, { method: "POST", body: { body } });
}

function deletePublicProfileComment(id) {
  return fetch(`${API.profileComments}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function updateMyProfile(payload) {
  return requestJSON(API.myProfile, { method: "PATCH", body: payload });
}

function sendFriendRequest(username) {
  return requestJSON(API.friendRequest, { method: "POST", body: { username } });
}

function fetchFriends() {
  return fetch(API.friends);
}

function fetchFriendRequests() {
  return fetch(API.friendRequests);
}

function acceptFriendRequest(id) {
  return requestJSON(`${API.friendRequests}/${encodeURIComponent(id)}/accept`, { method: "POST" });
}

function deleteFriendship(id) {
  return fetch(`${API.friends}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

function startMessageConversation(username) {
  return requestJSON(API.messagesStart, { method: "POST", body: { username } });
}

function fetchMessageConversations() {
  return fetch(API.messageConversations);
}

function fetchMessageConversation(id) {
  return fetch(`${API.messageConversations}/${encodeURIComponent(id)}`);
}

function sendConversationMessage(id, body) {
  return requestJSON(`${API.messageConversations}/${encodeURIComponent(id)}`, { method: "POST", body: { body } });
}

function deleteConversationMessage(id) {
  return fetch(`${API.messages}/${encodeURIComponent(id)}`, { method: "DELETE" });
}

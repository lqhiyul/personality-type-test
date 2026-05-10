const maxMessageBodyLength = 1000;

function messagesLabel(key, fallback, params = {}) {
  return t(`ui.messages.${key}`, fallback, params);
}

function resetMessagesState() {
  state.messageConversations = [];
  state.selectedMessageConversationID = null;
  state.selectedMessageConversation = null;
  state.selectedMessageMessages = [];
  state.messageConversationsLoading = false;
  state.selectedMessageLoading = false;
  state.messageSending = false;
  renderMessagesPanel();
}

function applyMessagesStaticText() {
  setText("profileMessagesEyebrow", messagesLabel("eyebrow", "Inbox"));
  setText("profileMessagesTitle", messagesLabel("title", "Messages"));
  setText("profileMessagesRefreshBtn", messagesLabel("refresh", "Refresh"));
  renderMessagesPanel();
}

function otherMessageParticipants(conversation) {
  const participants = Array.isArray(conversation?.participants) ? conversation.participants : [];
  const currentID = state.currentUser?.id;
  return participants.filter((participant) => String(participant.id) !== String(currentID));
}

function conversationTitle(conversation) {
  const others = otherMessageParticipants(conversation);
  const visible = others.length ? others : (Array.isArray(conversation?.participants) ? conversation.participants : []);
  return visible.map((participant) => participant.displayName || participant.username).filter(Boolean).join(", ") || messagesLabel("unknownUser", "User");
}

function renderMessagesPanel() {
  const list = E("messageConversationsList");
  const thread = E("messageThread");
  if (!list || !thread) return;

  if (!state.currentUser) {
    list.innerHTML = "";
    thread.innerHTML = "";
    return;
  }

  if (state.messageConversationsLoading) {
    list.innerHTML = `<div class="profile-messages__empty">${esc(messagesLabel("loading", "Loading conversations..."))}</div>`;
  } else if (!state.messageConversations.length) {
    list.innerHTML = `<div class="profile-messages__empty">${esc(messagesLabel("noConversations", "No conversations yet"))}</div>`;
  } else {
    list.innerHTML = state.messageConversations.map(renderMessageConversationItem).join("");
  }

  thread.innerHTML = renderMessageThread();
}

function renderMessageConversationItem(conversation) {
  const active = String(conversation.id) === String(state.selectedMessageConversationID);
  const last = conversation.lastMessage;
  const preview = last?.body
    ? `${last.sender?.username ? `${last.sender.username}: ` : ""}${last.body}`
    : messagesLabel("noMessages", "No messages yet");
  const time = last?.createdAt ? formatDate(last.createdAt) : formatDate(conversation.updatedAt);
  return `<button type="button" class="profile-message-conversation ${active ? "active" : ""}" data-message-conversation="${esc(conversation.id)}" aria-pressed="${active}">
    <span class="profile-message-conversation__title">${esc(conversationTitle(conversation))}</span>
    <span class="profile-message-conversation__preview">${esc(preview)}</span>
    <span class="profile-message-conversation__time">${esc(time)}</span>
  </button>`;
}

function renderMessageThread() {
  if (!state.selectedMessageConversationID) {
    return `<div class="profile-messages__empty">${esc(messagesLabel("selectConversation", "Select a conversation"))}</div>`;
  }
  if (state.selectedMessageLoading) {
    return `<div class="profile-messages__empty">${esc(messagesLabel("loadingMessages", "Loading messages..."))}</div>`;
  }

  const conversation = state.selectedMessageConversation;
  const messages = Array.isArray(state.selectedMessageMessages) ? state.selectedMessageMessages : [];
  const items = messages.length
    ? `<div class="profile-message-list" id="messageThreadList">${messages.map(renderMessageBubble).join("")}</div>`
    : `<div class="profile-messages__empty">${esc(messagesLabel("noMessages", "No messages yet"))}</div>`;

  return `<div class="profile-message-thread__head">
      <strong>${esc(conversationTitle(conversation))}</strong>
      <button type="button" class="profile-result__btn profile-result__btn--muted" data-message-thread-refresh="${esc(state.selectedMessageConversationID)}">${esc(messagesLabel("refresh", "Refresh"))}</button>
    </div>
    ${items}
    <form id="messageThreadForm" class="profile-message-form">
      <label for="messageBody">${esc(messagesLabel("write", "Write a message"))}</label>
      <textarea id="messageBody" maxlength="${maxMessageBodyLength}" rows="3" placeholder="${esc(messagesLabel("write", "Write a message"))}" ${state.messageSending ? "disabled" : ""}></textarea>
      <div class="profile-message-form__actions">
        <span>${esc(messagesLabel("limit", "1000 characters max"))}</span>
        <button type="submit" class="btn primary btn-compact" ${state.messageSending ? "disabled" : ""}>${esc(messagesLabel("send", "Send"))}</button>
      </div>
    </form>`;
}

function renderMessageBubble(message) {
  const own = String(message.sender?.id) === String(state.currentUser?.id);
  const sender = message.sender?.displayName || message.sender?.username || messagesLabel("unknownUser", "User");
  return `<article class="profile-message ${own ? "profile-message--own" : ""}">
    <div class="profile-message__meta">
      <strong>${esc(sender)}</strong>
      <span>${esc(formatDate(message.createdAt))}</span>
    </div>
    <p>${esc(message.body || "")}</p>
    ${own ? `<button type="button" class="profile-message__delete" data-message-delete="${esc(message.id)}">${esc(messagesLabel("delete", "Delete message"))}</button>` : ""}
  </article>`;
}

async function loadMessageConversations(options = {}) {
  if (!state.currentUser) {
    resetMessagesState();
    return;
  }
  const { silent = false } = options;
  state.messageConversationsLoading = !silent;
  renderMessagesPanel();
  try {
    const response = await fetchMessageConversations();
    if (response.status === 401) {
      state.currentUser = null;
      resetMessagesState();
      renderAuthPanel();
      return;
    }
    if (!response.ok) throw new Error(messagesLabel("loadFailed", "Could not load conversations"));
    const data = await response.json();
    state.messageConversations = Array.isArray(data.conversations) ? data.conversations : [];
    if (state.selectedMessageConversationID && !state.messageConversations.some((conversation) => String(conversation.id) === String(state.selectedMessageConversationID))) {
      state.selectedMessageConversationID = null;
      state.selectedMessageConversation = null;
      state.selectedMessageMessages = [];
    }
  } finally {
    state.messageConversationsLoading = false;
    renderMessagesPanel();
  }
}

async function selectMessageConversation(id, options = {}) {
  if (!state.currentUser || !id) return;
  const { silent = false } = options;
  state.selectedMessageConversationID = id;
  state.selectedMessageLoading = !silent;
  renderMessagesPanel();
  try {
    const response = await fetchMessageConversation(id);
    if (response.status === 401) {
      state.currentUser = null;
      resetMessagesState();
      renderAuthPanel();
      return;
    }
    if (!response.ok) throw new Error(messagesLabel("threadLoadFailed", "Could not load conversation"));
    const data = await response.json();
    state.selectedMessageConversation = data.conversation || null;
    state.selectedMessageMessages = Array.isArray(data.messages) ? data.messages : [];
  } finally {
    state.selectedMessageLoading = false;
    renderMessagesPanel();
  }
}

async function submitConversationMessage() {
  if (!state.currentUser || !state.selectedMessageConversationID) return;
  const input = E("messageBody");
  const body = String(input?.value || "").trim();
  if (!body) {
    showToast(messagesLabel("empty", "Message cannot be empty"), { title: messagesLabel("error", "Messages error"), tone: "error" });
    return;
  }
  if ([...body].length > maxMessageBodyLength) {
    showToast(messagesLabel("tooLong", "Message is too long"), { title: messagesLabel("error", "Messages error"), tone: "error" });
    return;
  }

  state.messageSending = true;
  renderMessagesPanel();
  const { response, data } = await sendConversationMessage(state.selectedMessageConversationID, body);
  state.messageSending = false;
  if (!response.ok) {
    renderMessagesPanel();
    throw new Error(data.error || messagesLabel("sendFailed", "Could not send message"));
  }
  state.selectedMessageMessages = [...state.selectedMessageMessages, data];
  if (input) input.value = "";
  await loadMessageConversations({ silent: true });
  renderMessagesPanel();
}

async function removeConversationMessage(id) {
  const confirmed = await askConfirm(messagesLabel("deleteConfirm", "Delete this message?"));
  if (!confirmed) return;
  const response = await deleteConversationMessage(id);
  if (!response.ok) {
    let data = {};
    try {
      data = await response.json();
    } catch (_) {}
    throw new Error(data.error || messagesLabel("deleteFailed", "Could not delete message"));
  }
  state.selectedMessageMessages = state.selectedMessageMessages.filter((message) => String(message.id) !== String(id));
  await loadMessageConversations({ silent: true });
  renderMessagesPanel();
}

async function openMessageConversationFromProfile(username) {
  if (!state.currentUser) {
    showToast(messagesLabel("loginPrompt", "Log in to send messages"), { title: messagesLabel("title", "Messages"), tone: "info" });
    setAuthPanelOpen(true);
    return;
  }
  const { response, data } = await startMessageConversation(username);
  if (!response.ok) {
    throw new Error(data.error || messagesLabel("startFailed", "Could not start conversation"));
  }
  state.selectedMessageConversationID = data.id;
  state.selectedMessageConversation = data;
  setAuthPanelOpen(true);
  await loadMessageConversations({ silent: true });
  await selectMessageConversation(data.id, { silent: true });
  showToast(messagesLabel("opened", "Conversation opened"), { title: messagesLabel("title", "Messages"), duration: 2200 });
}

function renderPublicProfileMessageAction(profile, ownProfile) {
  if (ownProfile) return "";
  if (!state.currentUser) {
    return `<div class="public-profile-message-state">${esc(messagesLabel("loginPrompt", "Log in to send messages"))}</div>`;
  }
  return `<div class="public-profile-message-actions">
    <button type="button" class="result-type-btn" data-public-message-start="${esc(profile.username)}">${esc(messagesLabel("messageUser", "Message this user"))}</button>
  </div>`;
}

function wireMessagesEvents() {
  E("profileMessagesRefreshBtn")?.addEventListener("click", () => loadMessageConversations().catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" })));

  E("messageConversationsList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const conversation = target?.closest("[data-message-conversation]");
    if (conversation?.dataset.messageConversation) {
      selectMessageConversation(conversation.dataset.messageConversation).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
    }
  });

  E("messageThread")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const refresh = target?.closest("[data-message-thread-refresh]");
    if (refresh?.dataset.messageThreadRefresh) {
      selectMessageConversation(refresh.dataset.messageThreadRefresh).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
      return;
    }
    const del = target?.closest("[data-message-delete]");
    if (del?.dataset.messageDelete) {
      removeConversationMessage(del.dataset.messageDelete).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
    }
  });

  E("messageThread")?.addEventListener("submit", (event) => {
    if (!(event.target instanceof HTMLFormElement) || event.target.id !== "messageThreadForm") return;
    event.preventDefault();
    submitConversationMessage().catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
  });

  E("profileSection")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const start = target?.closest("[data-public-message-start]");
    if (start?.dataset.publicMessageStart) {
      openMessageConversationFromProfile(start.dataset.publicMessageStart).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
    }
  });
}

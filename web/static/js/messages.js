function messagesLabel(key, fallback, params = {}) {
  return t(`ui.messages.${key}`, fallback, params);
}

function resetMessagesState() {
  state.messagesConversations = [];
  state.messagesConversation = null;
  state.messages = [];
  state.messagesLoading = false;
  state.messagesSending = false;
  state.selectedConversationId = "";
  renderMessagesPanel();
}

function applyMessagesStaticText() {
  setText("profileMessagesEyebrow", messagesLabel("eyebrow", "Messages"));
  setText("profileMessagesTitle", messagesLabel("title", "Inbox"));
  setText("profileMessagesRefreshBtn", messagesLabel("refresh", "Refresh"));
  renderMessagesPanel();
}

function messageParticipants(conversation) {
  return Array.isArray(conversation?.participants) ? conversation.participants : [];
}

function otherMessageParticipants(conversation) {
  const currentID = Number(state.currentUser?.id);
  return messageParticipants(conversation).filter((participant) => Number(participant.id) !== currentID);
}

function conversationTitle(conversation) {
  const others = otherMessageParticipants(conversation);
  const visible = others.length ? others : messageParticipants(conversation);
  return visible.map((participant) => participant.displayName || participant.username).filter(Boolean).join(", ") || messagesLabel("unknown", "User");
}

function messageSenderName(conversation, senderID) {
  if (Number(senderID) === Number(state.currentUser?.id)) return messagesLabel("you", "You");
  const sender = messageParticipants(conversation).find((participant) => Number(participant.id) === Number(senderID));
  return sender?.displayName || sender?.username || messagesLabel("unknown", "User");
}

function renderMessagesPanel() {
  const conversationsEl = E("messagesConversationList");
  const threadEl = E("messagesThread");
  if (!conversationsEl || !threadEl) return;

  if (!state.currentUser) {
    conversationsEl.innerHTML = "";
    threadEl.innerHTML = "";
    return;
  }

  const conversations = Array.isArray(state.messagesConversations) ? state.messagesConversations : [];
  if (state.messagesLoading && !conversations.length) {
    conversationsEl.innerHTML = `<div class="profile-friends__empty">${esc(messagesLabel("loading", "Loading messages..."))}</div>`;
  } else if (!conversations.length) {
    conversationsEl.innerHTML = `<div class="profile-friends__empty">${esc(messagesLabel("noConversations", "No conversations yet"))}</div>`;
  } else {
    conversationsEl.innerHTML = conversations.map(renderConversationListItem).join("");
  }

  if (!state.selectedConversationId) {
    threadEl.innerHTML = `<div class="messages-empty">${esc(messagesLabel("selectConversation", "Select a conversation"))}</div>`;
    return;
  }

  const conversation = state.messagesConversation || conversations.find((item) => String(item.id) === String(state.selectedConversationId));
  if (!conversation) {
    threadEl.innerHTML = `<div class="messages-empty">${esc(messagesLabel("loadingThread", "Loading conversation..."))}</div>`;
    return;
  }

  const messages = Array.isArray(state.messages) ? state.messages : [];
  const messagesHTML = state.messagesLoading
    ? `<div class="messages-empty">${esc(messagesLabel("loadingThread", "Loading conversation..."))}</div>`
    : messages.length
      ? messages.map((message) => renderMessageBubble(message, conversation)).join("")
      : `<div class="messages-empty">${esc(messagesLabel("noMessages", "No messages yet"))}</div>`;
  const disabled = Boolean(conversation.blocked || state.messagesSending);

  threadEl.innerHTML = `<div class="messages-thread__head">
      <strong>${esc(conversationTitle(conversation))}</strong>
      ${conversation.blocked ? `<span>${esc(messagesLabel("blocked", "Cannot interact with blocked user"))}</span>` : ""}
    </div>
    <div class="messages-thread__body">${messagesHTML}</div>
    <form id="messagesSendForm" class="messages-compose">
      <label class="sr-only" for="messagesBody">${esc(messagesLabel("write", "Write a message"))}</label>
      <textarea id="messagesBody" maxlength="1000" rows="3" placeholder="${esc(messagesLabel("write", "Write a message"))}" ${disabled ? "disabled" : ""}></textarea>
      <div class="messages-compose__actions">
        <span>${esc(messagesLabel("limit", "1000 characters max"))}</span>
        <button type="submit" class="btn primary btn-compact" ${disabled ? "disabled" : ""}>${esc(messagesLabel("send", "Send"))}</button>
      </div>
    </form>`;
}

function renderConversationListItem(conversation) {
  const active = String(conversation.id) === String(state.selectedConversationId);
  const last = conversation.lastMessage;
  const preview = last?.body
    ? `${messageSenderName(conversation, last.senderId)}: ${last.body}`
    : messagesLabel("noMessagesShort", "No messages yet");
  const time = last?.createdAt || conversation.updatedAt;
  return `<button type="button" class="messages-conversation ${active ? "active" : ""}" data-message-conversation="${esc(conversation.id)}" aria-pressed="${active}">
    <span>${esc(conversationTitle(conversation))}</span>
    <small>${esc(preview)}${time ? ` - ${esc(formatDate(time))}` : ""}</small>
  </button>`;
}

function renderMessageBubble(message, conversation) {
  const own = Number(message.senderId) === Number(state.currentUser?.id);
  const reportLabel = typeof safetyLabel === "function" ? safetyLabel("report", "Report") : "Report";
  return `<article class="message-bubble ${own ? "message-bubble--own" : ""}">
    <div class="message-bubble__meta">
      <span>${esc(messageSenderName(conversation, message.senderId))} - ${esc(formatDate(message.createdAt))}</span>
      <span>
        ${own ? `<button type="button" class="profile-result__btn profile-result__btn--danger" data-message-delete="${esc(message.id)}">${esc(messagesLabel("delete", "Delete"))}</button>` : ""}
        <button type="button" class="profile-result__btn profile-result__btn--muted" data-report-message="${esc(message.id)}">${esc(reportLabel)}</button>
      </span>
    </div>
    <p>${esc(message.body || "")}</p>
  </article>`;
}

async function loadMessageConversations(options = {}) {
  if (!state.currentUser) {
    resetMessagesState();
    return;
  }
  const { silent = false } = options;
  state.messagesLoading = !silent;
  renderMessagesPanel();
  try {
    const response = await fetchMessageConversations();
    if (response.status === 401) {
      state.currentUser = null;
      resetMessagesState();
      renderAuthPanel();
      return;
    }
    if (!response.ok) throw new Error(messagesLabel("loadFailed", "Could not load messages"));
    const data = await response.json();
    state.messagesConversations = Array.isArray(data.conversations) ? data.conversations : [];
    if (state.selectedConversationId && !state.messagesConversations.some((item) => String(item.id) === String(state.selectedConversationId))) {
      state.selectedConversationId = "";
      state.messagesConversation = null;
      state.messages = [];
    }
  } finally {
    state.messagesLoading = false;
    renderMessagesPanel();
  }
}

async function openMessageConversation(id) {
  if (!state.currentUser || !id) return;
  state.selectedConversationId = String(id);
  state.messagesLoading = true;
  renderMessagesPanel();
  try {
    const response = await fetchMessageConversation(id);
    if (response.status === 401) {
      state.currentUser = null;
      resetMessagesState();
      renderAuthPanel();
      return;
    }
    if (!response.ok) throw new Error(messagesLabel("threadFailed", "Could not load conversation"));
    const data = await response.json();
    state.messagesConversation = data.conversation || null;
    state.messages = Array.isArray(data.messages) ? data.messages : [];
  } finally {
    state.messagesLoading = false;
    renderMessagesPanel();
  }
}

async function startMessageFromProfile(username) {
  if (!state.currentUser) {
    showToast(messagesLabel("loginPrompt", "Log in to send messages"), { title: messagesLabel("title", "Inbox"), tone: "error" });
    setAuthPanelOpen(true);
    return;
  }
  const { response, data } = await startMessageConversation(username);
  if (!response.ok) {
    throw new Error(data.error || messagesLabel("startFailed", "Could not start conversation"));
  }
  await loadMessageConversations({ silent: true });
  await openMessageConversation(data.id);
  setAuthPanelOpen(true);
  showToast(messagesLabel("opened", "Conversation opened"), { title: messagesLabel("title", "Inbox"), duration: 2200 });
}

async function sendSelectedConversationMessage() {
  if (!state.selectedConversationId) return;
  const input = E("messagesBody");
  const body = String(input?.value || "").trim();
  if (!body) {
    showToast(messagesLabel("empty", "Message cannot be empty"), { title: messagesLabel("error", "Messages error"), tone: "error" });
    return;
  }
  if ([...body].length > 1000) {
    showToast(messagesLabel("tooLong", "Message is too long"), { title: messagesLabel("error", "Messages error"), tone: "error" });
    return;
  }

  state.messagesSending = true;
  renderMessagesPanel();
  const { response, data } = await sendConversationMessage(state.selectedConversationId, body);
  state.messagesSending = false;
  if (!response.ok) {
    renderMessagesPanel();
    throw new Error(data.error || messagesLabel("sendFailed", "Could not send message"));
  }
  state.messages = [...state.messages, data];
  if (input) input.value = "";
  await loadMessageConversations({ silent: true });
  await openMessageConversation(state.selectedConversationId);
}

async function removeMessage(id) {
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
  state.messages = state.messages.filter((message) => String(message.id) !== String(id));
  renderMessagesPanel();
  await loadMessageConversations({ silent: true });
}

function renderPublicProfileMessageAction(profile, ownProfile) {
  if (ownProfile) return "";
  const viewerBlock = profile.viewerBlock || {};
  if (viewerBlock.targetBlockedViewer || viewerBlock.viewerBlockedTarget) return "";
  if (!state.currentUser) {
    return `<span class="public-profile-friend-state public-profile-friend-state--muted">${esc(messagesLabel("loginPrompt", "Log in to send messages"))}</span>`;
  }
  return `<button type="button" class="result-type-btn result-type-btn--muted" data-message-profile="${esc(profile.username)}">${esc(messagesLabel("messageUser", "Message"))}</button>`;
}

function wireMessagesEvents() {
  E("profileMessagesRefreshBtn")?.addEventListener("click", () => loadMessageConversations().catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" })));

  E("messagesConversationList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest("[data-message-conversation]");
    if (button?.dataset.messageConversation) {
      openMessageConversation(button.dataset.messageConversation).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
    }
  });

  E("messagesThread")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const del = target?.closest("[data-message-delete]");
    if (del?.dataset.messageDelete) {
      removeMessage(del.dataset.messageDelete).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
      return;
    }
    const report = target?.closest("[data-report-message]");
    if (report?.dataset.reportMessage && typeof promptReport === "function") {
      promptReport({ targetType: "message", targetId: Number(report.dataset.reportMessage) }).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
    }
  });

  E("messagesThread")?.addEventListener("submit", (event) => {
    if (!(event.target instanceof HTMLFormElement) || event.target.id !== "messagesSendForm") return;
    event.preventDefault();
    sendSelectedConversationMessage().catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
  });

  E("profileSection")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest("[data-message-profile]");
    if (button?.dataset.messageProfile) {
      startMessageFromProfile(button.dataset.messageProfile).catch((error) => showToast(error.message, { title: messagesLabel("error", "Messages error"), tone: "error" }));
    }
  });
}

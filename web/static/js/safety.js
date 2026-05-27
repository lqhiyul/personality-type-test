function safetyLabel(key, fallback, params = {}) {
  return t(`ui.safety.${key}`, fallback, params);
}

function resetSafetyState() {
  state.blockedUsers = [];
  state.blocksLoading = false;
  renderSafetyPanel();
}

function applySafetyStaticText() {
  setText("profileBlocksEyebrow", safetyLabel("eyebrow", "Safety"));
  setText("profileBlocksTitle", safetyLabel("blockedUsers", "Blocked users"));
  setText("profileBlocksRefreshBtn", safetyLabel("refresh", "Refresh"));
  setText("adminReportsTitle", safetyLabel("reports", "Reports"));
  setText("adminReportsRefreshBtn", safetyLabel("refresh", "Refresh"));
  setText("adminReportsStatusLabel", safetyLabel("status", "Status"));
  const statusFilter = E("adminReportsStatusFilter");
  if (statusFilter) {
    const labels = {
      open: safetyLabel("open", "Open"),
      reviewed: safetyLabel("reviewed", "Reviewed"),
      dismissed: safetyLabel("dismissed", "Dismissed"),
      "": safetyLabel("all", "All"),
    };
    Array.from(statusFilter.options).forEach((option) => {
      option.textContent = labels[option.value] || option.textContent;
    });
  }
  renderSafetyPanel();
  renderAdminReportsPanel();
}

function renderSafetyPanel() {
  const list = E("blockedUsersList");
  if (!list) return;
  if (!state.currentUser) {
    list.innerHTML = "";
    return;
  }
  if (state.blocksLoading) {
    list.innerHTML = `<div class="profile-friends__empty">${esc(safetyLabel("loadingBlocks", "Loading blocked users..."))}</div>`;
    return;
  }
  const blocks = Array.isArray(state.blockedUsers) ? state.blockedUsers : [];
  if (!blocks.length) {
    list.innerHTML = `<div class="profile-friends__empty">${esc(safetyLabel("noBlockedUsers", "No blocked users"))}</div>`;
    return;
  }
  list.innerHTML = blocks.map((block) => `<article class="profile-friend-card">
    <div class="profile-friend-card__top">
      <div class="${esc(avatarClass(avatarKeyOrDefault(block.avatarKey)))}" aria-hidden="true">${esc(avatarSymbolFor(block.avatarKey, block.username))}</div>
      <div class="profile-friend-card__identity">
        <strong>${esc(block.displayName || block.username)}</strong>
        <span>@${esc(block.username)}</span>
      </div>
    </div>
    <div class="profile-friend-card__actions">
      <button type="button" class="profile-result__btn profile-result__btn--muted" data-blocked-profile="${esc(block.username)}">${esc(safetyLabel("viewProfile", "Profile"))}</button>
      <button type="button" class="profile-result__btn" data-unblock-user="${esc(block.username)}">${esc(safetyLabel("unblockUser", "Unblock user"))}</button>
    </div>
  </article>`).join("");
}

async function loadBlocksData(options = {}) {
  if (!state.currentUser) {
    resetSafetyState();
    return;
  }
  const { silent = false } = options;
  state.blocksLoading = !silent;
  renderSafetyPanel();
  try {
    const response = await fetchBlocks();
    if (response.status === 401) {
      state.currentUser = null;
      resetSafetyState();
      renderAuthPanel();
      return;
    }
    if (!response.ok) throw new Error(safetyLabel("loadBlocksFailed", "Could not load blocked users"));
    const data = await response.json();
    state.blockedUsers = Array.isArray(data.blocks) ? data.blocks : [];
  } finally {
    state.blocksLoading = false;
    renderSafetyPanel();
  }
}

async function blockProfileUser(username) {
  const confirmed = await askConfirm(safetyLabel("blockConfirm", "Block this user?"));
  if (!confirmed) return;
  const { response, data } = await blockUser(username);
  if (!response.ok) {
    throw new Error(data.error || safetyLabel("blockFailed", "Could not block user"));
  }
  await afterSafetyRelationshipChange(username);
  showToast(safetyLabel("blocked", "User blocked"), { title: safetyLabel("done", "Done"), duration: 2200 });
}

async function unblockProfileUser(username) {
  const response = await unblockUser(username);
  if (!response.ok) {
    let data = {};
    try {
      data = await response.json();
    } catch (_) {}
    throw new Error(data.error || safetyLabel("unblockFailed", "Could not unblock user"));
  }
  await afterSafetyRelationshipChange(username);
  showToast(safetyLabel("unblocked", "User unblocked"), { title: safetyLabel("done", "Done"), duration: 2200 });
}

async function afterSafetyRelationshipChange(username) {
  await loadBlocksData({ silent: true });
  if (typeof loadFriendsData === "function") await loadFriendsData({ silent: true });
  if (typeof loadMessageConversations === "function") await loadMessageConversations({ silent: true });
  if (state.publicProfileUsername && !E("profileSection")?.hidden && typeof openPublicProfile === "function") {
    await openPublicProfile(state.publicProfileUsername || username, { updateHistory: false });
  }
}

function profileInteractionBlocked(profile) {
  const block = profile?.viewerBlock || {};
  return Boolean(block.viewerBlockedTarget || block.targetBlockedViewer);
}

function renderPublicProfileSafetyActions(profile, ownProfile) {
  if (ownProfile) return "";
  if (!state.currentUser) {
    return `<div class="public-profile-friend-state public-profile-friend-state--muted">${esc(safetyLabel("loginToUseSafety", "Log in to report or block users"))}</div>`;
  }
  const block = profile.viewerBlock || {};
  if (block.targetBlockedViewer) {
    return `<div class="public-profile-safety-actions">
      <span class="public-profile-friend-state public-profile-friend-state--muted">${esc(safetyLabel("userUnavailable", "This user is unavailable"))}</span>
      <button type="button" class="result-type-btn result-type-btn--muted" data-report-profile="${esc(profile.username)}">${esc(safetyLabel("reportProfile", "Report profile"))}</button>
    </div>`;
  }
  if (block.viewerBlockedTarget) {
    return `<div class="public-profile-safety-actions">
      <span class="public-profile-friend-state">${esc(safetyLabel("youBlocked", "You blocked this user"))}</span>
      <button type="button" class="result-type-btn" data-unblock-profile="${esc(profile.username)}">${esc(safetyLabel("unblockUser", "Unblock user"))}</button>
      <button type="button" class="result-type-btn result-type-btn--muted" data-report-profile="${esc(profile.username)}">${esc(safetyLabel("reportProfile", "Report profile"))}</button>
    </div>`;
  }
  return `<div class="public-profile-safety-actions">
    <button type="button" class="result-type-btn result-type-btn--muted" data-block-profile="${esc(profile.username)}">${esc(safetyLabel("blockUser", "Block user"))}</button>
    <button type="button" class="result-type-btn result-type-btn--muted" data-report-profile="${esc(profile.username)}">${esc(safetyLabel("reportProfile", "Report profile"))}</button>
  </div>`;
}

async function promptReport({ targetType, targetId = 0, username = "" }) {
  if (!state.currentUser) {
    showToast(safetyLabel("loginToReport", "Log in to report"), { title: safetyLabel("report", "Report"), tone: "error" });
    return;
  }
  const reason = window.prompt(safetyLabel("reasonPrompt", "Reason"));
  if (reason === null) return;
  const trimmedReason = reason.trim();
  if (!trimmedReason) {
    showToast(safetyLabel("reasonRequired", "Reason is required"), { title: safetyLabel("report", "Report"), tone: "error" });
    return;
  }
  const details = window.prompt(safetyLabel("detailsPrompt", "Details optional")) || "";
  const { response, data } = await createReport({
    targetType,
    targetId,
    username,
    reason: trimmedReason,
    details,
  });
  if (!response.ok) {
    throw new Error(data.error || safetyLabel("reportFailed", "Could not submit report"));
  }
  showToast(safetyLabel("reportSubmitted", "Report submitted"), { title: safetyLabel("report", "Report"), duration: 2200 });
}

function renderAdminReportsPanel() {
  const list = E("adminReportsList");
  const filter = E("adminReportsStatusFilter");
  if (!list) return;
  if (filter) filter.value = state.adminReportStatusFilter || "open";
  if (state.adminReportsLoading) {
    list.innerHTML = `<div class="admin-state">${esc(safetyLabel("loadingReports", "Loading reports..."))}</div>`;
    return;
  }
  const reports = Array.isArray(state.adminReports) ? state.adminReports : [];
  if (!reports.length) {
    list.innerHTML = `<div class="admin-state">${esc(safetyLabel("noReports", "No reports"))}</div>`;
    return;
  }
  list.innerHTML = reports.map((report) => {
    const reporter = report.reporter ? `@${report.reporter.username}` : safetyLabel("unknownUser", "Unknown user");
    const target = report.targetUser ? `@${report.targetUser.username}` : safetyLabel("unknownTarget", "Unknown target");
    return `<article class="admin-report-card">
      <div class="admin-report-card__top">
        <strong>${esc(report.targetType)} #${esc(report.targetId || "")}</strong>
        <span>${esc(report.status)}</span>
      </div>
      <p>${esc(report.reason || "")}</p>
      ${report.details ? `<p class="admin-report-card__details">${esc(report.details)}</p>` : ""}
      <div class="admin-report-card__meta">${esc(reporter)} -> ${esc(target)} - ${esc(formatDate(report.createdAt))}</div>
      <div class="admin-report-card__actions">
        <button type="button" class="table-action" data-report-status="${esc(report.id)}" data-status-value="open">${esc(safetyLabel("open", "Open"))}</button>
        <button type="button" class="table-action" data-report-status="${esc(report.id)}" data-status-value="reviewed">${esc(safetyLabel("reviewed", "Reviewed"))}</button>
        <button type="button" class="table-action" data-report-status="${esc(report.id)}" data-status-value="dismissed">${esc(safetyLabel("dismissed", "Dismissed"))}</button>
      </div>
    </article>`;
  }).join("");
}

async function loadAdminReports() {
  state.adminReportsLoading = true;
  renderAdminReportsPanel();
  try {
    const response = await fetchAdminReports(state.adminReportStatusFilter || "");
    if (response.status === 401) {
      state.adminReports = [];
      return;
    }
    if (!response.ok) throw new Error(safetyLabel("loadReportsFailed", "Could not load reports"));
    const data = await response.json();
    state.adminReports = Array.isArray(data.reports) ? data.reports : [];
  } finally {
    state.adminReportsLoading = false;
    renderAdminReportsPanel();
  }
}

async function setAdminReportStatus(id, status) {
  const { response, data } = await updateAdminReportStatus(id, status);
  if (!response.ok) {
    throw new Error(data.error || safetyLabel("reportStatusFailed", "Could not update report"));
  }
  await loadAdminReports();
}

function wireSafetyEvents() {
  E("profileBlocksRefreshBtn")?.addEventListener("click", () => loadBlocksData().catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" })));
  E("blockedUsersList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const unblock = target?.closest("[data-unblock-user]");
    if (unblock?.dataset.unblockUser) {
      unblockProfileUser(unblock.dataset.unblockUser).catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
      return;
    }
    const profile = target?.closest("[data-blocked-profile]");
    if (profile?.dataset.blockedProfile && typeof openPublicProfile === "function") {
      setAuthPanelOpen(false);
      openPublicProfile(profile.dataset.blockedProfile);
    }
  });

  E("profileSection")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const block = target?.closest("[data-block-profile]");
    if (block?.dataset.blockProfile) {
      blockProfileUser(block.dataset.blockProfile).catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
      return;
    }
    const unblock = target?.closest("[data-unblock-profile]");
    if (unblock?.dataset.unblockProfile) {
      unblockProfileUser(unblock.dataset.unblockProfile).catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
      return;
    }
    const profileReport = target?.closest("[data-report-profile]");
    if (profileReport?.dataset.reportProfile) {
      promptReport({ targetType: "profile", username: profileReport.dataset.reportProfile }).catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
      return;
    }
    const commentReport = target?.closest("[data-report-comment]");
    if (commentReport?.dataset.reportComment) {
      promptReport({ targetType: "comment", targetId: Number(commentReport.dataset.reportComment) }).catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
    }
  });

  E("adminReportsRefreshBtn")?.addEventListener("click", () => loadAdminReports().catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" })));
  E("adminReportsStatusFilter")?.addEventListener("change", (event) => {
    state.adminReportStatusFilter = event.target?.value || "open";
    loadAdminReports().catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
  });
  E("adminReportsList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const status = target?.closest("[data-report-status]");
    if (status?.dataset.reportStatus && status.dataset.statusValue) {
      setAdminReportStatus(status.dataset.reportStatus, status.dataset.statusValue).catch((error) => showToast(error.message, { title: safetyLabel("error", "Safety error"), tone: "error" }));
    }
  });
}

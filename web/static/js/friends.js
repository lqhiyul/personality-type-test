function friendsLabel(key, fallback, params = {}) {
  return t(`ui.friends.${key}`, fallback, params);
}

function resetFriendsState() {
  state.friends = [];
  state.friendRequests = [];
  state.friendsLoading = false;
  state.friendRequestsLoading = false;
  renderFriendsPanel();
}

function applyFriendsStaticText() {
  setText("profileFriendsEyebrow", friendsLabel("eyebrow", "Friends"));
  setText("profileFriendsTitle", friendsLabel("title", "Friends"));
  setText("profileFriendsRefreshBtn", friendsLabel("refresh", "Refresh"));
  setText("profileFriendRequestsTitle", friendsLabel("requestsTitle", "Friend requests"));
  setText("profileFriendListTitle", friendsLabel("listTitle", "Friends list"));
  renderFriendsPanel();
}

function renderFriendsPanel() {
  const requestsList = E("friendRequestsList");
  const friendsList = E("friendsList");
  if (!requestsList || !friendsList) return;

  if (!state.currentUser) {
    requestsList.innerHTML = "";
    friendsList.innerHTML = "";
    return;
  }

  if (state.friendRequestsLoading) {
    requestsList.innerHTML = `<div class="profile-friends__empty">${esc(friendsLabel("loadingRequests", "Loading requests..."))}</div>`;
  } else if (!state.friendRequests.length) {
    requestsList.innerHTML = `<div class="profile-friends__empty">${esc(friendsLabel("noRequests", "No incoming requests."))}</div>`;
  } else {
    requestsList.innerHTML = state.friendRequests.map(renderIncomingFriendRequest).join("");
  }

  if (state.friendsLoading) {
    friendsList.innerHTML = `<div class="profile-friends__empty">${esc(friendsLabel("loadingFriends", "Loading friends..."))}</div>`;
  } else if (!state.friends.length) {
    friendsList.innerHTML = `<div class="profile-friends__empty">${esc(friendsLabel("noFriends", "Accepted friends will appear here."))}</div>`;
  } else {
    friendsList.innerHTML = state.friends.map(renderFriendListItem).join("");
  }
}

function renderIncomingFriendRequest(request) {
  const requester = request.requester || {};
  const key = avatarKeyOrDefault(requester.avatarKey);
  const primary = requester.primaryType ? `${requester.primaryType} - ${getTypeName(requester.primaryType)}` : friendsLabel("noPrimaryShort", "No primary type");
  return `<article class="profile-friend-card">
    <div class="profile-friend-card__top">
      <div class="${esc(avatarClass(key))}" aria-hidden="true">${esc(avatarSymbolFor(key, requester.username))}</div>
      <div class="profile-friend-card__identity">
        <strong>${esc(requester.displayName || requester.username || friendsLabel("unknownUser", "User"))}</strong>
        <span>@${esc(requester.username || "")}</span>
      </div>
    </div>
    <div class="profile-friend-card__meta">${esc(primary)}</div>
    <div class="profile-friend-card__actions">
      <button type="button" class="profile-result__btn" data-friend-accept="${esc(request.id)}">${esc(friendsLabel("accept", "Accept"))}</button>
      ${requester.username ? `<button type="button" class="profile-result__btn profile-result__btn--muted" data-friend-profile="${esc(requester.username)}">${esc(friendsLabel("viewProfile", "Profile"))}</button>` : ""}
    </div>
  </article>`;
}

function renderFriendListItem(friend) {
  const key = avatarKeyOrDefault(friend.avatarKey);
  const primary = friend.primaryType ? `${friend.primaryType} - ${getTypeName(friend.primaryType)}` : friendsLabel("noPrimaryShort", "No primary type");
  return `<article class="profile-friend-card">
    <div class="profile-friend-card__top">
      <div class="${esc(avatarClass(key))}" aria-hidden="true">${esc(avatarSymbolFor(key, friend.username))}</div>
      <div class="profile-friend-card__identity">
        <strong>${esc(friend.displayName || friend.username)}</strong>
        <span>@${esc(friend.username)}</span>
      </div>
    </div>
    <div class="profile-friend-card__meta">${esc(primary)}</div>
    ${renderFriendCompatibility(friend.compatibility)}
    <div class="profile-friend-card__actions">
      <button type="button" class="profile-result__btn profile-result__btn--muted" data-friend-profile="${esc(friend.username)}">${esc(friendsLabel("viewProfile", "Profile"))}</button>
      <button type="button" class="profile-result__btn profile-result__btn--danger" data-friend-remove="${esc(friend.friendshipId)}">${esc(friendsLabel("remove", "Remove"))}</button>
    </div>
  </article>`;
}

function renderFriendCompatibility(compatibility = {}) {
  if (!compatibility.available) {
    return `<div class="profile-friend-compat profile-friend-compat--empty">${esc(compatibility.reason || friendsLabel("compatUnavailable", "Compatibility unavailable"))}</div>`;
  }
  const items = [
    ["friendship", friendsLabel("friendship", "Friendship"), compatibility.friendship],
    ["relationship", friendsLabel("relationship", "Relationship"), compatibility.relationship],
    ["work", friendsLabel("work", "Work"), compatibility.work],
  ];
  return `<div class="profile-friend-compat">${items.map(([, label, value]) => `<span><strong>${esc(value)}%</strong>${esc(label)}</span>`).join("")}</div>`;
}

async function loadFriendsData(options = {}) {
  if (!state.currentUser) {
    resetFriendsState();
    return;
  }
  const { silent = false } = options;
  state.friendsLoading = !silent;
  state.friendRequestsLoading = !silent;
  renderFriendsPanel();

  try {
    const [friendsResponse, requestsResponse] = await Promise.all([fetchFriends(), fetchFriendRequests()]);
    if (friendsResponse.status === 401 || requestsResponse.status === 401) {
      state.currentUser = null;
      resetFriendsState();
      renderAuthPanel();
      return;
    }
    if (!friendsResponse.ok) throw new Error(friendsLabel("loadFriendsFailed", "Could not load friends"));
    if (!requestsResponse.ok) throw new Error(friendsLabel("loadRequestsFailed", "Could not load friend requests"));
    const friendsData = await friendsResponse.json();
    const requestsData = await requestsResponse.json();
    state.friends = Array.isArray(friendsData.friends) ? friendsData.friends : [];
    state.friendRequests = Array.isArray(requestsData.requests) ? requestsData.requests : [];
  } finally {
    state.friendsLoading = false;
    state.friendRequestsLoading = false;
    renderFriendsPanel();
  }
}

async function requestFriendFromProfile(username) {
  const { response, data } = await sendFriendRequest(username);
  if (!response.ok) {
    throw new Error(data.error || friendsLabel("requestFailed", "Could not send friend request"));
  }
  await loadFriendsData({ silent: true });
  await refreshVisiblePublicProfile();
  showToast(friendsLabel("requestSent", "Friend request sent"), { title: friendsLabel("done", "Done"), duration: 2200 });
}

async function acceptIncomingFriendRequest(id) {
  const { response, data } = await acceptFriendRequest(id);
  if (!response.ok) {
    throw new Error(data.error || friendsLabel("acceptFailed", "Could not accept friend request"));
  }
  await loadFriendsData({ silent: true });
  await refreshVisiblePublicProfile();
  showToast(friendsLabel("accepted", "Friend request accepted"), { title: friendsLabel("done", "Done"), duration: 2200 });
}

async function removeFriendshipByID(id) {
  const confirmed = await askConfirm(friendsLabel("removeConfirm", "Remove this friend?"));
  if (!confirmed) return;
  const response = await deleteFriendship(id);
  if (!response.ok) {
    let data = {};
    try {
      data = await response.json();
    } catch (_) {}
    throw new Error(data.error || friendsLabel("removeFailed", "Could not remove friend"));
  }
  state.friends = state.friends.filter((friend) => String(friend.friendshipId) !== String(id));
  renderFriendsPanel();
  await refreshVisiblePublicProfile();
  showToast(friendsLabel("removed", "Friend removed"), { title: friendsLabel("done", "Done"), duration: 2200 });
}

async function refreshVisiblePublicProfile() {
  if (!state.publicProfileUsername || E("profileSection")?.hidden || typeof openPublicProfile !== "function") return;
  await openPublicProfile(state.publicProfileUsername, { updateHistory: false });
}

function renderPublicProfileFriendActions(profile, ownProfile) {
  if (ownProfile) return "";
  if (!state.currentUser) {
    return `<div class="public-profile-friend-state public-profile-friend-state--muted">${esc(friendsLabel("loginPrompt", "Log in to add friends."))}</div>`;
  }

  const relationship = profile.viewerFriendship || { status: "none" };
  const status = relationship.status || "none";
  if (status === "friends") {
    return `<div class="public-profile-friend-actions">
      <span class="public-profile-friend-state">${esc(friendsLabel("stateFriends", "Friends"))}</span>
      <button type="button" class="result-type-btn result-type-btn--muted" data-friend-remove="${esc(relationship.friendshipId)}">${esc(friendsLabel("remove", "Remove friend"))}</button>
    </div>`;
  }
  if (status === "request_sent") {
    return `<div class="public-profile-friend-actions">
      <span class="public-profile-friend-state">${esc(friendsLabel("stateSent", "Request sent"))}</span>
    </div>`;
  }
  if (status === "request_received") {
    return `<div class="public-profile-friend-actions">
      <span class="public-profile-friend-state">${esc(friendsLabel("stateReceived", "Request received"))}</span>
    </div>`;
  }
  return `<div class="public-profile-friend-actions">
    <span class="public-profile-friend-state public-profile-friend-state--muted">${esc(friendsLabel("stateNone", "Not friends"))}</span>
    <button type="button" class="result-type-btn" data-public-friend-request="${esc(profile.username)}">${esc(friendsLabel("addFriend", "Add friend"))}</button>
  </div>`;
}

function wireFriendsEvents() {
  E("profileFriendsRefreshBtn")?.addEventListener("click", () => loadFriendsData().catch((error) => showToast(error.message, { title: friendsLabel("error", "Friends error"), tone: "error" })));

  E("friendRequestsList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const accept = target?.closest("[data-friend-accept]");
    if (accept?.dataset.friendAccept) {
      acceptIncomingFriendRequest(accept.dataset.friendAccept).catch((error) => showToast(error.message, { title: friendsLabel("error", "Friends error"), tone: "error" }));
      return;
    }
    const profile = target?.closest("[data-friend-profile]");
    if (profile?.dataset.friendProfile && typeof openPublicProfile === "function") {
      setAuthPanelOpen(false);
      openPublicProfile(profile.dataset.friendProfile);
    }
  });

  E("friendsList")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const remove = target?.closest("[data-friend-remove]");
    if (remove?.dataset.friendRemove) {
      removeFriendshipByID(remove.dataset.friendRemove).catch((error) => showToast(error.message, { title: friendsLabel("error", "Friends error"), tone: "error" }));
      return;
    }
    const profile = target?.closest("[data-friend-profile]");
    if (profile?.dataset.friendProfile && typeof openPublicProfile === "function") {
      setAuthPanelOpen(false);
      openPublicProfile(profile.dataset.friendProfile);
    }
  });

  E("profileSection")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    const request = target?.closest("[data-public-friend-request]");
    if (request?.dataset.publicFriendRequest) {
      requestFriendFromProfile(request.dataset.publicFriendRequest).catch((error) => showToast(error.message, { title: friendsLabel("error", "Friends error"), tone: "error" }));
      return;
    }
    const remove = target?.closest("[data-friend-remove]");
    if (remove?.dataset.friendRemove) {
      removeFriendshipByID(remove.dataset.friendRemove).catch((error) => showToast(error.message, { title: friendsLabel("error", "Friends error"), tone: "error" }));
    }
  });
}

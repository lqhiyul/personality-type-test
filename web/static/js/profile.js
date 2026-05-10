const AVATAR_PRESETS = [
  "gradient-violet",
  "gradient-blue",
  "gradient-gold",
  "gradient-green",
  "gradient-red",
  "symbol-analyst",
  "symbol-explorer",
  "symbol-guardian",
  "symbol-creator",
];

function publicProfileLabel(key, fallback, params = {}) {
  return t(`ui.publicProfile.${key}`, fallback, params);
}

function avatarKeyOrDefault(key) {
  return AVATAR_PRESETS.includes(key) ? key : "gradient-violet";
}

function avatarSymbolFor(key, username = "") {
  const symbols = {
    "symbol-analyst": "A",
    "symbol-explorer": "E",
    "symbol-guardian": "G",
    "symbol-creator": "C",
  };
  return symbols[key] || String(username || "U").slice(0, 1).toUpperCase();
}

function avatarClass(key) {
  return `avatar-preset avatar-preset--${avatarKeyOrDefault(key)}`;
}

function profileURLForUsername(username) {
  const params = new URLSearchParams();
  params.set("profile", String(username || "").trim().toLowerCase());
  return `${location.origin}${location.pathname}?${params}`;
}

function setPublicProfileViewVisible() {
  E("tabQuiz")?.classList.remove("active");
  E("tabTypes")?.classList.remove("active");
  E("tabCompatibility")?.classList.remove("active");
  E("tabQuiz")?.setAttribute("aria-selected", "false");
  E("tabTypes")?.setAttribute("aria-selected", "false");
  E("tabCompatibility")?.setAttribute("aria-selected", "false");
  E("quizSection").hidden = true;
  E("typesSection").hidden = true;
  E("compatibilitySection").hidden = true;
  E("profileSection").hidden = false;
  E("quizHero").hidden = true;
  E("typesHero").hidden = true;
  E("compatibilityHero").hidden = true;
  E("profileHero").hidden = false;
}

function updatePublicProfileRoute(username, replace = false) {
  const next = profileURLForUsername(username).replace(location.origin, "");
  if (replace) history.replaceState(null, "", next);
  else history.pushState(null, "", next);
}

async function openPublicProfile(username, options = {}) {
  const normalized = String(username || "").trim().toLowerCase();
  if (!normalized) return;
  const { updateHistory = true, replace = false } = options;
  state.publicProfileUsername = normalized;
  state.publicProfileLoading = true;
  state.publicProfileError = "";
  state.publicProfile = null;
  state.profileEditOpen = false;
  setPublicProfileViewVisible();
  renderPublicProfile();
  if (updateHistory) updatePublicProfileRoute(normalized, replace);

  try {
    const response = await fetchPublicProfile(normalized);
    if (response.status === 404) {
      state.publicProfileError = publicProfileLabel("notFound", "Profile not found");
      return;
    }
    if (!response.ok) throw new Error(publicProfileLabel("loadFailed", "Could not load profile"));
    state.publicProfile = await response.json();
    state.profileEditAvatarKey = avatarKeyOrDefault(state.publicProfile.avatarKey);
  } catch (error) {
    state.publicProfileError = error.message || publicProfileLabel("loadFailed", "Could not load profile");
  } finally {
    state.publicProfileLoading = false;
    renderPublicProfile();
  }
}

function renderPublicProfile() {
  const section = E("profileSection");
  if (!section) return;
  setPublicProfileViewVisible();

  if (state.publicProfileLoading) {
    section.innerHTML = `<article class="public-profile-card"><p class="public-profile-state">${esc(publicProfileLabel("loading", "Loading profile..."))}</p></article>`;
    return;
  }

  if (state.publicProfileError) {
    section.innerHTML = `<article class="public-profile-card public-profile-card--state">
      <h2>${esc(state.publicProfileError)}</h2>
      <p>${esc(publicProfileLabel("notFoundCopy", "The profile may have been renamed or is unavailable."))}</p>
      <button type="button" class="result-type-btn" data-profile-home>${esc(publicProfileLabel("backHome", "Back to quiz"))}</button>
    </article>`;
    return;
  }

  const profile = state.publicProfile;
  if (!profile) {
    section.innerHTML = "";
    return;
  }

  const ownProfile = Boolean(state.currentUser && state.currentUser.username === profile.username);
  const key = avatarKeyOrDefault(profile.avatarKey);
  if (profile.isPrivate || profile.profileVisibility === "private") {
    section.innerHTML = `<article class="public-profile-card public-profile-card--state">
      <div class="public-profile-head">
        <div class="${esc(avatarClass(key))}" aria-hidden="true">${esc(avatarSymbolFor(key, profile.username))}</div>
        <div class="public-profile-title">
          <h2>${esc(profile.displayName || profile.username)}</h2>
          <p>@${esc(profile.username)}</p>
        </div>
      </div>
      <p class="public-profile-note">${esc(ownProfile ? publicProfileLabel("privateOwn", "This is how other people see your private profile.") : publicProfileLabel("privateProfile", "This profile is private."))}</p>
      <div class="public-profile-actions">
        <button type="button" class="result-type-btn result-type-btn--muted" data-profile-copy="${esc(profile.username)}">${esc(publicProfileLabel("copyLink", "Copy profile link"))}</button>
        <button type="button" class="result-type-btn" data-profile-home>${esc(publicProfileLabel("backHome", "Back to quiz"))}</button>
      </div>
    </article>`;
    return;
  }

  const type = profile.primaryType ? getTypeData(profile.primaryType) : null;
  const typeSummary = type?.summary?.shortSummary || type?.tagline || "";
  const primaryDate = profile.primaryResultDate ? formatDate(profile.primaryResultDate) : "";
  const edit = ownProfile && state.profileEditOpen ? renderProfileEditForm(profile) : "";
  const completedLabel = profile.showCompletedCount === false
    ? publicProfileLabel("completedHidden", "Hidden")
    : String(Number(profile.completedTestsCount || 0));

  section.innerHTML = `<div class="public-profile-layout">
    <article class="public-profile-card">
      <div class="public-profile-head">
        <div class="${esc(avatarClass(key))}" aria-hidden="true">${esc(avatarSymbolFor(key, profile.username))}</div>
        <div class="public-profile-title">
          <h2>${esc(profile.displayName || profile.username)}</h2>
          <p>@${esc(profile.username)}</p>
        </div>
      </div>
      ${profile.bio ? `<p class="public-profile-bio">${esc(profile.bio)}</p>` : `<p class="public-profile-bio public-profile-bio--empty">${esc(publicProfileLabel("emptyBio", "No bio yet."))}</p>`}
      <div class="public-profile-meta">
        <span>${esc(publicProfileLabel("completed", "Completed tests"))}: <strong>${esc(completedLabel)}</strong></span>
        ${profile.showPrimaryResult === false ? `<span>${esc(publicProfileLabel("primarySince", "Primary result"))}: <strong>${esc(publicProfileLabel("primaryHidden", "Hidden"))}</strong></span>` : (primaryDate ? `<span>${esc(publicProfileLabel("primarySince", "Primary result"))}: <strong>${esc(primaryDate)}</strong></span>` : "")}
      </div>
      <div class="public-profile-actions">
        ${ownProfile ? `<button type="button" class="result-type-btn" data-profile-edit>${esc(state.profileEditOpen ? publicProfileLabel("closeEdit", "Close edit") : publicProfileLabel("edit", "Edit profile"))}</button>` : ""}
        <button type="button" class="result-type-btn result-type-btn--muted" data-profile-copy="${esc(profile.username)}">${esc(publicProfileLabel("copyLink", "Copy profile link"))}</button>
      </div>
      ${typeof renderPublicProfileFriendActions === "function" ? renderPublicProfileFriendActions(profile, ownProfile) : ""}
    </article>
    ${renderPublicProfileTypeCard(profile, type, typeSummary)}
    ${renderPublicProfileFriends(profile)}
    ${edit}
  </div>`;
}

function renderPublicProfileTypeCard(profile, type, typeSummary) {
  if (profile.showPrimaryResult === false) {
    return `<article class="public-profile-card public-profile-type public-profile-type--muted">
      <div class="public-profile-type__badge">${esc(publicProfileLabel("hiddenBadge", "Hidden"))}</div>
      <h3>${esc(publicProfileLabel("hiddenPrimaryTitle", "Primary MBTI is hidden"))}</h3>
      <p>${esc(publicProfileLabel("hiddenPrimaryCopy", "This user chose not to show their primary result publicly."))}</p>
    </article>`;
  }
  if (!type) {
    return `<article class="public-profile-card public-profile-type public-profile-type--muted">
      <div class="public-profile-type__badge">${esc(publicProfileLabel("emptyBadge", "Empty"))}</div>
      <h3>${esc(publicProfileLabel("noPrimaryTitle", "No public type yet"))}</h3>
      <p>${esc(publicProfileLabel("noPrimaryCopy", "This user has not selected a primary result."))}</p>
    </article>`;
  }
  return `<article class="public-profile-card public-profile-type">
    <div class="public-profile-type__badge">${esc(type.code)}</div>
    <h3>${esc(type.name)}</h3>
    <p>${esc(typeSummary)}</p>
    <button type="button" class="result-type-btn" data-open-type="${esc(type.code)}">${esc(publicProfileLabel("readType", "Read type profile"))}</button>
  </article>`;
}

function renderPublicProfileFriends(profile) {
  if (profile.showFriends === false) {
    return `<article class="public-profile-card public-profile-friends-card">
      <h3>${esc(publicProfileLabel("friendsTitle", "Friends"))}</h3>
      <p class="public-profile-state">${esc(publicProfileLabel("friendsHidden", "Friends list is hidden."))}</p>
    </article>`;
  }

  const friends = Array.isArray(profile.friends) ? profile.friends : [];
  if (!friends.length) {
    return `<article class="public-profile-card public-profile-friends-card">
      <h3>${esc(publicProfileLabel("friendsTitle", "Friends"))}</h3>
      <p class="public-profile-state">${esc(publicProfileLabel("friendsEmpty", "No public friends to show yet."))}</p>
    </article>`;
  }

  const cards = friends.map((friend) => {
    const key = avatarKeyOrDefault(friend.avatarKey);
    const primary = friend.primaryType ? `${friend.primaryType} - ${getTypeName(friend.primaryType)}` : publicProfileLabel("friendPrimaryHidden", "Primary type hidden");
    return `<article class="public-profile-friend">
      <div class="${esc(avatarClass(key))}" aria-hidden="true">${esc(avatarSymbolFor(key, friend.username))}</div>
      <div>
        <strong>${esc(friend.displayName || friend.username)}</strong>
        <span>@${esc(friend.username)} - ${esc(primary)}</span>
      </div>
    </article>`;
  }).join("");

  return `<article class="public-profile-card public-profile-friends-card">
    <h3>${esc(publicProfileLabel("friendsTitle", "Friends"))}</h3>
    <div class="public-profile-friends-list">${cards}</div>
  </article>`;
}

function renderProfileEditForm(profile) {
  const selected = avatarKeyOrDefault(state.profileEditAvatarKey || profile.avatarKey);
  const buttons = AVATAR_PRESETS.map((key) => {
    const active = key === selected;
    return `<button type="button" class="avatar-choice ${active ? "active" : ""}" data-profile-avatar="${esc(key)}" aria-pressed="${active}">
      <span class="${esc(avatarClass(key))}" aria-hidden="true">${esc(avatarSymbolFor(key, profile.username))}</span>
      <span>${esc(key.replace("-", " "))}</span>
    </button>`;
  }).join("");
  return `<article class="public-profile-card public-profile-edit">
    <h3>${esc(publicProfileLabel("editTitle", "Edit public profile"))}</h3>
    <form id="publicProfileForm">
      <div class="field">
        <label for="profileDisplayName">${esc(publicProfileLabel("displayName", "Display name"))}</label>
        <input id="profileDisplayName" type="text" maxlength="64" value="${esc(profile.displayName || profile.username)}" />
      </div>
      <div class="field">
        <label for="profileBio">${esc(publicProfileLabel("bio", "Bio"))}</label>
        <textarea id="profileBio" maxlength="280">${esc(profile.bio || "")}</textarea>
      </div>
      <div class="avatar-choice-grid" role="group" aria-label="${esc(publicProfileLabel("avatar", "Avatar preset"))}">${buttons}</div>
      <div class="public-profile-edit__actions">
        <button type="submit" class="btn primary">${esc(publicProfileLabel("save", "Save profile"))}</button>
        <button type="button" class="btn ghost" data-profile-cancel>${esc(publicProfileLabel("cancel", "Cancel"))}</button>
      </div>
    </form>
  </article>`;
}

async function submitPublicProfileForm() {
  if (!state.currentUser || !state.publicProfile) return;
  const payload = {
    displayName: E("profileDisplayName")?.value || "",
    bio: E("profileBio")?.value || "",
    avatarKey: avatarKeyOrDefault(state.profileEditAvatarKey),
  };
  const { response, data } = await updateMyProfile(payload);
  if (!response.ok) {
    showToast(data.error || publicProfileLabel("saveFailed", "Could not save profile"), { title: publicProfileLabel("error", "Profile error"), tone: "error" });
    return;
  }
  state.currentUser = { ...state.currentUser, ...data };
  state.profileEditOpen = false;
  renderAuthPanel();
  await openPublicProfile(state.publicProfileUsername || data.username, { updateHistory: false });
  showToast(publicProfileLabel("saved", "Profile updated"), { title: publicProfileLabel("done", "Done"), duration: 2200 });
}

async function copyProfileLink(username) {
  const url = profileURLForUsername(username);
  try {
    await navigator.clipboard.writeText(url);
    showToast(publicProfileLabel("linkCopied", "Profile link copied"), { title: publicProfileLabel("done", "Done"), duration: 2200 });
  } catch (_) {
    showToast(url, { title: publicProfileLabel("copyLink", "Copy profile link"), duration: 4200 });
  }
}

function wireProfileEvents() {
  E("profileSection")?.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target : null;
    if (target?.closest("[data-profile-home]")) {
      setTab("quiz");
      return;
    }
    if (target?.closest("[data-profile-edit]")) {
      state.profileEditOpen = !state.profileEditOpen;
      state.profileEditAvatarKey = avatarKeyOrDefault(state.publicProfile?.avatarKey);
      renderPublicProfile();
      return;
    }
    if (target?.closest("[data-profile-cancel]")) {
      state.profileEditOpen = false;
      renderPublicProfile();
      return;
    }
    const avatar = target?.closest("[data-profile-avatar]");
    if (avatar?.dataset.profileAvatar) {
      state.profileEditAvatarKey = avatarKeyOrDefault(avatar.dataset.profileAvatar);
      renderPublicProfile();
      return;
    }
    const copy = target?.closest("[data-profile-copy]");
    if (copy?.dataset.profileCopy) {
      copyProfileLink(copy.dataset.profileCopy);
    }
  });

  E("profileSection")?.addEventListener("submit", (event) => {
    if (!(event.target instanceof HTMLFormElement) || event.target.id !== "publicProfileForm") return;
    event.preventDefault();
    submitPublicProfileForm().catch((error) => showToast(error.message, { title: publicProfileLabel("error", "Profile error"), tone: "error" }));
  });
}

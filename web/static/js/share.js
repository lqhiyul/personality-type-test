function drawRoundRect(ctx, x, y, width, height, radius) {
  const r = Math.min(radius, width / 2, height / 2);
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + width - r, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + r);
  ctx.lineTo(x + width, y + height - r);
  ctx.quadraticCurveTo(x + width, y + height, x + width - r, y + height);
  ctx.lineTo(x + r, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - r);
  ctx.lineTo(x, y + r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.closePath();
}

function drawWrappedText(ctx, text, x, y, maxWidth, lineHeight, maxLines) {
  const words = String(text || "").split(/\s+/).filter(Boolean);
  const lines = [];
  let line = "";
  words.forEach((word) => {
    const test = line ? `${line} ${word}` : word;
    if (ctx.measureText(test).width <= maxWidth || !line) {
      line = test;
      return;
    }
    lines.push(line);
    line = word;
  });
  if (line) lines.push(line);
  const visible = lines.slice(0, maxLines);
  if (lines.length > maxLines && visible.length) {
    let last = visible[visible.length - 1];
    while (last.length && ctx.measureText(`${last}...`).width > maxWidth) last = last.slice(0, -1);
    visible[visible.length - 1] = `${last.trim()}...`;
  }
  visible.forEach((row, index) => ctx.fillText(row, x, y + index * lineHeight));
  return y + visible.length * lineHeight;
}

function drawShareCard(payload) {
  const canvas = document.createElement("canvas");
  canvas.width = 1200;
  canvas.height = 630;
  const ctx = canvas.getContext("2d");
  const bg = ctx.createLinearGradient(0, 0, 1200, 630);
  bg.addColorStop(0, "#090a13");
  bg.addColorStop(0.46, "#111625");
  bg.addColorStop(1, "#06070c");
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, 1200, 630);

  const glowA = ctx.createRadialGradient(210, 120, 20, 210, 120, 430);
  glowA.addColorStop(0, "rgba(201,187,255,.25)");
  glowA.addColorStop(1, "rgba(201,187,255,0)");
  ctx.fillStyle = glowA;
  ctx.fillRect(0, 0, 1200, 630);

  const glowB = ctx.createRadialGradient(980, 520, 20, 980, 520, 390);
  glowB.addColorStop(0, "rgba(244,238,223,.15)");
  glowB.addColorStop(1, "rgba(244,238,223,0)");
  ctx.fillStyle = glowB;
  ctx.fillRect(0, 0, 1200, 630);

  drawRoundRect(ctx, 36, 36, 1128, 558, 34);
  ctx.fillStyle = "rgba(12,15,24,.68)";
  ctx.fill();
  ctx.strokeStyle = "rgba(244,238,223,.18)";
  ctx.lineWidth = 1.5;
  ctx.stroke();

  ctx.fillStyle = "rgba(245,247,255,.72)";
  ctx.font = "700 28px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(t("ui.shareCard.brand"), 76, 96);
  ctx.fillStyle = "rgba(201,187,255,.82)";
  ctx.font = "700 18px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(t("ui.shareCard.title"), 76, 130);

  ctx.fillStyle = "#f8f5ff";
  ctx.font = "900 132px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(payload.type, 76, 260);

  ctx.fillStyle = "#f4eedf";
  ctx.font = "800 40px Inter, Segoe UI, Arial, sans-serif";
  drawWrappedText(ctx, payload.name, 76, 314, 620, 46, 2);

  if (payload.socioCode) {
    drawRoundRect(ctx, 76, 348, 330, 46, 23);
    ctx.fillStyle = "rgba(201,187,255,.12)";
    ctx.fill();
    ctx.strokeStyle = "rgba(201,187,255,.26)";
    ctx.stroke();
    ctx.fillStyle = "rgba(238,232,255,.9)";
    ctx.font = "700 20px Inter, Segoe UI, Arial, sans-serif";
    ctx.fillText(`${t("ui.shareCard.socionics")}: ${payload.socioCode}`, 98, 378);
  }

  ctx.fillStyle = "rgba(232,236,255,.84)";
  ctx.font = "500 30px Inter, Segoe UI, Arial, sans-serif";
  drawWrappedText(ctx, payload.shareText, 76, 452, 710, 42, 3);

  ctx.fillStyle = "rgba(201,187,255,.78)";
  ctx.font = "800 20px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(t("ui.shareCard.traits"), 806, 178);

  let chipY = 212;
  (payload.traits || []).forEach((trait) => {
    drawRoundRect(ctx, 806, chipY, 306, 58, 18);
    ctx.fillStyle = "rgba(255,255,255,.055)";
    ctx.fill();
    ctx.strokeStyle = "rgba(255,255,255,.1)";
    ctx.stroke();
    ctx.fillStyle = "rgba(245,247,255,.86)";
    ctx.font = "700 21px Inter, Segoe UI, Arial, sans-serif";
    drawWrappedText(ctx, trait, 828, chipY + 35, 264, 24, 1);
    chipY += 72;
  });

  drawRoundRect(ctx, 806, 468, 306, 68, 20);
  ctx.fillStyle = "rgba(244,238,223,.12)";
  ctx.fill();
  ctx.strokeStyle = "rgba(244,238,223,.2)";
  ctx.stroke();
  ctx.fillStyle = "#f4eedf";
  ctx.font = "800 22px Inter, Segoe UI, Arial, sans-serif";
  drawWrappedText(ctx, t("ui.shareCard.cta"), 828, 500, 264, 26, 2);

  ctx.fillStyle = "rgba(232,236,255,.58)";
  ctx.font = "700 20px Inter, Segoe UI, Arial, sans-serif";
  ctx.fillText(payload.displayUrl, 76, 552);

  return canvas;
}

function canvasToBlob(canvas) {
  return new Promise((resolve) => canvas.toBlob(resolve, "image/png", 0.95));
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function openSharePreview(payload, rendered) {
  if (activeModal) closeActiveModal(null);
  const previousFocus = document.activeElement;
  const backdrop = document.createElement("div");
  const canNativeShare = Boolean(navigator.share);
  backdrop.className = "modal-backdrop share-backdrop";
  backdrop.innerHTML = `
    <div class="modal-panel share-modal" role="dialog" aria-modal="true" aria-labelledby="shareModalTitle">
      <div class="modal-head"><h3 id="shareModalTitle">${esc(t("ui.shareCard.preview"))}</h3></div>
      <p class="modal-copy share-modal__copy">${esc(t("ui.shareCard.hint", "The PNG card is ready. Share it, copy the link, or download the image."))}</p>
      <img class="share-card-preview" src="${rendered.previewUrl}" alt="${esc(t("ui.shareCard.preview"))}" data-share-preview data-share-source="${esc(rendered.source || "canvas")}" />
      <div class="modal-actions share-modal__actions">
        ${canNativeShare ? `<button type="button" class="modal-btn modal-btn--primary" data-share-web>${esc(t("ui.shareCard.share"))}</button>` : ""}
        <button type="button" class="modal-btn" data-share-link>${esc(t("ui.shareCard.copyLink", "Copy link"))}</button>
        <button type="button" class="modal-btn" data-share-download>${esc(t("ui.shareCard.download"))}</button>
        <button type="button" class="modal-btn" data-share-close>${esc(t("ui.shareCard.close"))}</button>
      </div>
    </div>`;
  const keyHandler = (event) => {
    if (!activeModal || activeModal.backdrop !== backdrop) return;
    if (event.key === "Escape") {
      event.preventDefault();
      closeActiveModal(false);
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
  activeModal = { backdrop, resolve: () => {}, previousFocus, keyHandler, cleanup: rendered.cleanup };
  document.addEventListener("keydown", keyHandler, true);
  requestAnimationFrame(() => backdrop.classList.add("visible"));
  setTimeout(() => backdrop.querySelector("button")?.focus(), 0);

  backdrop.addEventListener("mousedown", (event) => {
    if (event.target === backdrop) closeActiveModal(false);
  });
  backdrop.addEventListener("click", async (event) => {
    const target = event.target instanceof Element ? event.target : null;
    if (target?.closest("[data-share-close]")) {
      closeActiveModal(false);
      return;
    }
    if (target?.closest("[data-share-download]")) {
      downloadBlob(rendered.blob, `personality-type-${payload.type.toLowerCase()}.png`);
      showToast(t("ui.shareCard.downloadReady"), { title: t("ui.notices.done"), duration: 2200 });
      return;
    }
    if (target?.closest("[data-share-link]")) {
      try {
        if (!navigator.clipboard?.writeText) throw new Error("clipboard unavailable");
        await navigator.clipboard.writeText(payload.url);
        showToast(t("ui.shareCard.linkCopied", t("ui.shareCard.copied")), { title: t("ui.shareCard.copyLink", "Copy link"), duration: 2200 });
      } catch (_) {
        showToast(payload.url, { title: t("ui.shareCard.copyLink", "Copy link"), duration: 4200 });
      }
      return;
    }
    if (target?.closest("[data-share-web]")) {
      try {
        const file = new File([rendered.blob], `personality-type-${payload.type.toLowerCase()}.png`, { type: "image/png" });
        if (navigator.canShare?.({ files: [file] })) {
          await navigator.share({ title: payload.title, text: shareTextFor(payload.type), files: [file] });
        } else {
          await navigator.share({ title: payload.title, text: shareTextFor(payload.type), url: payload.url });
        }
      } catch (error) {
        if (error?.name !== "AbortError") showToast(shareTextFor(payload.type), { title: t("ui.shareCard.copy"), duration: 4200 });
      }
    }
  });
}

async function openShareCard() {
  if (!state.lastResult) return;
  const payload = sharePayload(state.lastResult.type);
  if (!payload) return;
  const staticCard = await loadStaticShareCard(payload.type);
  if (staticCard) {
    openSharePreview(payload, staticCard);
    return;
  }
  const canvas = drawShareCard(payload);
  const blob = await canvasToBlob(canvas);
  if (!blob) {
    showToast(shareTextFor(payload.type), { title: t("ui.shareCard.copy"), duration: 4200 });
    return;
  }
  openSharePreview(payload, { blob, previewUrl: canvas.toDataURL("image/png"), source: "canvas" });
}

function copyResult() {
  if (!state.lastResult) return;
  const text = shareTextFor(state.lastResult.type);
  if (!navigator.clipboard?.writeText) {
    showToast(text, { title: t("ui.result.copy"), duration: 4200 });
    return;
  }
  navigator.clipboard.writeText(text)
    .then(() => showToast(t("ui.notices.copied"), { title: t("ui.notices.done"), duration: 2200 }))
    .catch(() => showToast(text, { title: t("ui.result.copy"), duration: 4200 }));
}

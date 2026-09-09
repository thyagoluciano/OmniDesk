// OmniDesk Webview Dashboard Application

let localNode = null;
let currentPendingPIN = null;
let selectedTargetForUpload = null;
let trustedDevicesCache = [];
let discoveredDevicesCache = [];
let filesSyncEnabled = true;
let openSendRowId = null;

document.addEventListener("DOMContentLoaded", () => {
  setupTabNav();
  initApp();
  setupEventListeners();
  setupModuleToggles();
  setupKvmEventListeners();
  // Poll every 3 seconds
  setInterval(refreshDevicesAndStatus, 3000);
  setInterval(refreshKvm, 3000);
  setInterval(refreshClipboardHistory, 4000);
});

async function initApp() {
  await refreshDevicesAndStatus();
  await refreshKvm();
  await refreshClipboardHistory();
}

// ---------------------------------------------------------------------
// Toasts and confirmation modal (replace native alert()/confirm(), which
// render as unstyled OS dialogs that clash with the app's dark theme).
// ---------------------------------------------------------------------

const TOAST_ICONS = {
  success: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 6 9 17l-5-5"/></svg>`,
  error: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`,
  info: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`
};

function showToast(message, type, duration) {
  type = type || "info";
  duration = duration === undefined ? 3800 : duration;

  const container = document.getElementById("toast-container");
  const toast = document.createElement("div");
  toast.className = `toast toast-${type}`;
  toast.innerHTML = `
    <span class="toast-icon">${TOAST_ICONS[type] || TOAST_ICONS.info}</span>
    <span class="toast-text">${escapeHtml(message)}</span>
    <button class="toast-close" aria-label="Fechar">&times;</button>
  `;

  const remove = () => {
    toast.classList.add("toast-out");
    setTimeout(() => toast.remove(), 150);
  };
  toast.querySelector(".toast-close").addEventListener("click", remove);

  container.appendChild(toast);
  if (duration > 0) setTimeout(remove, duration);
}

function showConfirm(title, message, confirmLabel) {
  return new Promise((resolve) => {
    const modal = document.getElementById("modal-confirm");
    document.getElementById("confirm-title").textContent = title;
    document.getElementById("confirm-message").textContent = message;

    const okBtn = document.getElementById("btn-confirm-ok");
    const cancelBtn = document.getElementById("btn-confirm-cancel");
    const closeBtn = document.getElementById("btn-close-confirm-modal");
    okBtn.textContent = confirmLabel || "Confirmar";

    const cleanup = (result) => {
      modal.classList.add("hidden");
      okBtn.removeEventListener("click", onOk);
      cancelBtn.removeEventListener("click", onCancel);
      closeBtn.removeEventListener("click", onCancel);
      resolve(result);
    };
    const onOk = () => cleanup(true);
    const onCancel = () => cleanup(false);

    okBtn.addEventListener("click", onOk);
    cancelBtn.addEventListener("click", onCancel);
    closeBtn.addEventListener("click", onCancel);

    modal.classList.remove("hidden");
  });
}

// ---------------------------------------------------------------------
// Sidebar navigation
// ---------------------------------------------------------------------

function setupTabNav() {
  document.querySelectorAll(".nav-item").forEach((item) => {
    item.addEventListener("click", () => {
      const tab = item.dataset.tab;
      document.querySelectorAll(".nav-item").forEach((el) => el.classList.toggle("active", el === item));
      document.querySelectorAll(".panel").forEach((panel) => {
        panel.classList.toggle("hidden", panel.id !== `panel-${tab}`);
      });
    });
  });
}

// ---------------------------------------------------------------------
// Module master toggles (clipboard / files / input sharing)
// ---------------------------------------------------------------------

function setupModuleToggles() {
  document.getElementById("clipboard-toggle").addEventListener("change", async (e) => {
    try {
      const resp = await fetch("/api/v1/clipboard/toggle", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: e.target.checked })
      });
      const data = await resp.json();
      applyModuleState("clipboard", data.clipboard_sync);
    } catch (err) {
      console.error("Falha ao alterar sincronização de clipboard:", err);
    }
  });

  document.getElementById("files-toggle").addEventListener("change", async (e) => {
    try {
      const resp = await fetch("/api/v1/files/toggle", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: e.target.checked })
      });
      const data = await resp.json();
      applyModuleState("files", data.files_sync);
      await refreshDevicesAndStatus(); // re-render device rows so the send-file button shows/hides
    } catch (err) {
      console.error("Falha ao alterar sincronização de arquivos:", err);
    }
  });

  document.getElementById("kvm-toggle").addEventListener("change", async (e) => {
    const enabling = e.target.checked;

    if (enabling) {
      const confirmed = await showConfirm(
        "Controle Remoto (Beta)",
        "Este recurso ainda está em fase beta e pode apresentar instabilidades — perda de conexão do mouse/teclado, travamentos ou comportamento inesperado. Deseja ativar mesmo assim?",
        "Ativar assim mesmo"
      );
      if (!confirmed) {
        e.target.checked = false;
        return;
      }
    }

    try {
      const resp = await fetch("/api/v1/input/toggle", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: enabling })
      });
      const data = await resp.json();
      applyModuleState("kvm", data.input_share_enabled);
    } catch (err) {
      console.error("Falha ao alterar controle remoto:", err);
      e.target.checked = !enabling;
    }
  });
}

function applyModuleState(module, enabled) {
  const toggle = document.getElementById(`${module}-toggle`);
  const label = document.getElementById(`${module}-status-label`);
  const notice = document.getElementById(`${module}-disabled-notice`);
  const content = document.getElementById(`${module}-content`);
  const dot = document.getElementById(`nav-${module}-dot`);

  toggle.checked = !!enabled;
  label.textContent = enabled ? "Ativado" : "Desativado";
  notice.classList.toggle("hidden", !!enabled);
  content.classList.toggle("disabled", !enabled);
  dot.classList.toggle("off", !enabled);

  if (module === "files") {
    filesSyncEnabled = !!enabled;
  }
}

// ---------------------------------------------------------------------
// General event wiring (pairing, scan, modals)
// ---------------------------------------------------------------------

function setupEventListeners() {
  // Manual Scan
  document.getElementById("btn-manual-scan").addEventListener("click", () => {
    const btn = document.getElementById("btn-manual-scan");
    btn.disabled = true;
    btn.textContent = "Buscando...";
    refreshDevicesAndStatus().finally(() => {
      setTimeout(() => {
        btn.disabled = false;
        btn.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg> Escanear`;
      }, 800);
    });
  });

  // PIN Approval Modal
  const modalPin = document.getElementById("modal-pin-approve");
  const inputPin = document.getElementById("input-pin");
  const pinError = document.getElementById("pin-error-msg");

  document.getElementById("btn-pair-pin").addEventListener("click", () => {
    inputPin.value = "";
    pinError.classList.add("hidden");
    modalPin.classList.remove("hidden");
    inputPin.focus();
  });

  document.getElementById("btn-close-pin-modal").addEventListener("click", () => modalPin.classList.add("hidden"));
  document.getElementById("btn-cancel-pin").addEventListener("click", () => modalPin.classList.add("hidden"));

  document.getElementById("btn-submit-pin").addEventListener("click", async () => {
    const pin = inputPin.value.trim();
    if (pin.length !== 6) {
      pinError.textContent = "O PIN deve conter exatamente 6 dígitos numéricos.";
      pinError.classList.remove("hidden");
      return;
    }
    await approvePin(pin, modalPin, pinError);
  });

  // Banner actions
  document.getElementById("btn-banner-approve").addEventListener("click", async () => {
    if (currentPendingPIN) {
      await approvePin(currentPendingPIN, null, null);
    }
  });

  document.getElementById("btn-banner-dismiss").addEventListener("click", () => {
    document.getElementById("pairing-banner").classList.add("hidden");
  });

  // Request Pair Modal
  const modalReq = document.getElementById("modal-request-pair");
  const reqAddr = document.getElementById("input-target-addr");
  const reqError = document.getElementById("pair-error-msg");
  const flowState = document.getElementById("pair-flow-state");

  document.getElementById("btn-pair-request").addEventListener("click", () => {
    reqAddr.value = "";
    flowState.classList.add("hidden");
    reqError.classList.add("hidden");
    document.getElementById("btn-submit-req").style.display = "inline-flex";
    modalReq.classList.remove("hidden");
    reqAddr.focus();
  });

  document.getElementById("btn-close-req-modal").addEventListener("click", () => modalReq.classList.add("hidden"));
  document.getElementById("btn-cancel-req").addEventListener("click", () => modalReq.classList.add("hidden"));

  document.getElementById("btn-submit-req").addEventListener("click", async () => {
    const target = reqAddr.value.trim();
    if (!target) return;
    reqError.classList.add("hidden");

    try {
      const resp = await fetch("/api/v1/devices/pair", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target })
      });
      const data = await resp.json();
      if (!resp.ok) {
        reqError.textContent = data.error || "Erro ao solicitar pareamento";
        reqError.classList.remove("hidden");
        return;
      }

      document.getElementById("display-generated-pin").textContent = data.pin;
      flowState.classList.remove("hidden");
      document.getElementById("btn-submit-req").style.display = "none";
    } catch (e) {
      reqError.textContent = "Falha de rede ao contatar nó remoto: " + e.message;
      reqError.classList.remove("hidden");
    }
  });

  // Hidden File input change (shared by device rows and the Files panel)
  const fileInput = document.getElementById("global-file-input");
  fileInput.addEventListener("change", () => {
    if (fileInput.files && fileInput.files.length > 0 && selectedTargetForUpload) {
      uploadFilesToDevice(selectedTargetForUpload, fileInput.files);
    }
  });

  // Files panel dropzone (target picked from the select)
  const filesDropzone = document.getElementById("files-dropzone");
  filesDropzone.addEventListener("click", () => {
    const target = document.getElementById("files-target-select").value;
    if (!target) return;
    selectedTargetForUpload = target;
    fileInput.click();
  });
  filesDropzone.addEventListener("dragover", (e) => { e.preventDefault(); filesDropzone.classList.add("drag-over"); });
  filesDropzone.addEventListener("dragleave", () => filesDropzone.classList.remove("drag-over"));
  filesDropzone.addEventListener("drop", (e) => {
    e.preventDefault();
    filesDropzone.classList.remove("drag-over");
    const target = document.getElementById("files-target-select").value;
    if (target && e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      uploadFilesToDevice(target, e.dataTransfer.files, "files-progress", "files-progress-fill");
    }
  });
}

async function refreshDevicesAndStatus() {
  try {
    // 1. Fetch Local Status
    const statusResp = await fetch("/api/v1/status");
    if (statusResp.ok) {
      localNode = await statusResp.json();
      document.getElementById("local-device-name").textContent = localNode.device_name;
      document.getElementById("local-device-info").textContent = `ID: ${localNode.device_id.substring(0, 8)}... :${localNode.port || 24850}`;
      applyModuleState("clipboard", localNode.clipboard_sync);
      applyModuleState("files", localNode.files_sync);
      applyModuleState("kvm", localNode.input_share_enabled);
      if (localNode.download_dir) {
        document.getElementById("footer-storage-path").textContent = localNode.download_dir;
      }
    }

    // 2. Fetch Devices
    const devResp = await fetch("/api/v1/devices");
    if (devResp.ok) {
      const devData = await devResp.json();
      renderDevices(devData.trusted || [], devData.discovered || []);
    }

    // 3. Fetch Pending Pairings
    const pendingResp = await fetch("/api/v1/pair/pending");
    if (pendingResp.ok) {
      const pending = await pendingResp.json();
      renderPendingBanner(pending);
    }
  } catch (err) {
    console.warn("Erro ao atualizar status:", err);
  }
}

function renderPendingBanner(pendingList) {
  const banner = document.getElementById("pairing-banner");
  if (!pendingList || pendingList.length === 0) {
    banner.classList.add("hidden");
    currentPendingPIN = null;
    return;
  }

  const s = pendingList[0];
  currentPendingPIN = s.pin;
  document.getElementById("banner-title").textContent = `Pareamento Solicitado por ${s.requester_name}`;
  document.getElementById("banner-desc").innerHTML = `Dispositivo <strong>${s.requester_name}</strong> quer parear com esta máquina. PIN: <strong style="font-size:16px;letter-spacing:2px;">${s.pin}</strong>`;
  banner.classList.remove("hidden");
}

// ---------------------------------------------------------------------
// Devices panel
// ---------------------------------------------------------------------

function renderDevices(trusted, discovered) {
  trustedDevicesCache = trusted;
  if (discovered !== undefined) discoveredDevicesCache = discovered;
  const trustedList = document.getElementById("trusted-devices-list");
  const discoveredList = document.getElementById("discovered-devices-list");

  document.getElementById("trusted-count-badge").textContent = `${trusted.length} dispositivo${trusted.length === 1 ? "" : "s"}`;
  document.getElementById("nav-trusted-badge").textContent = trusted.length;

  // Filter discovered to exclude already trusted
  const trustedIDs = new Set(trusted.map((d) => d.id));
  const unpairedDiscovered = discoveredDevicesCache.filter((d) => !trustedIDs.has(d.id) && d.id !== (localNode ? localNode.device_id : ""));

  document.getElementById("discovered-count-badge").textContent = `${unpairedDiscovered.length} na rede`;

  // Render Trusted
  trustedList.innerHTML = "";
  if (trusted.length === 0) {
    trustedList.innerHTML = `
      <div class="empty-placeholder">
        <p>Nenhum dispositivo pareado ainda.</p>
        <p class="hint">Pareie com seus outros computadores abaixo para sincronizar clipboard e arquivos.</p>
      </div>
    `;
  } else {
    trusted.forEach((dev) => trustedList.appendChild(createTrustedRow(dev)));
  }

  // Update the target select on the Files panel
  populateFilesTargetSelect(trusted);

  // Render Discovered
  discoveredList.innerHTML = "";
  if (unpairedDiscovered.length === 0) {
    discoveredList.innerHTML = `
      <div class="empty-placeholder">
        <p>Nenhum outro nó OmniDesk detectado na LAN no momento.</p>
        <p class="hint">Certifique-se de que as máquinas estão na mesma rede Wi-Fi/cabo.</p>
      </div>
    `;
  } else {
    unpairedDiscovered.forEach((dev) => discoveredList.appendChild(createDiscoveredRow(dev)));
  }
}

function createTrustedRow(dev) {
  const wrap = document.createElement("div");

  const isOnline = dev.is_online;
  const statusColor = isOnline ? "var(--success)" : "var(--text-muted)";
  const statusLabel = isOnline ? "Online" : "Offline";
  const sendOpen = openSendRowId === dev.id;

  const sendBtnHtml = filesSyncEnabled
    ? `<button class="btn-icon${sendOpen ? " active" : ""}" data-action="toggle-send" title="Enviar arquivo">
         <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2 11 13"/><path d="M22 2 15 22 11 13 2 9z"/></svg>
       </button>`
    : "";

  wrap.innerHTML = `
    <div class="row-card" data-device-id="${dev.id}">
      <div class="row-icon">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
      </div>
      <div style="flex:1;min-width:0;">
        <div class="row-title">${escapeHtml(dev.name)}</div>
        <div class="row-sub">${escapeHtml(dev.last_addr || "Endereço desc.")} &middot; ${escapeHtml(dev.last_seen || "Recentemente")}</div>
      </div>
      <span class="status-pill" style="color:${statusColor};">${statusLabel}</span>
      ${sendBtnHtml}
      <button class="btn-icon" data-action="remove" title="Remover dispositivo">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
      </button>
    </div>
    ${sendOpen ? `
    <div class="inline-dropzone" id="inline-dropzone-${dev.id}">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
      <span>Arraste um arquivo aqui ou clique para enviar para <strong>${escapeHtml(dev.name)}</strong></span>
    </div>` : ""}
  `;

  const row = wrap.querySelector(".row-card");

  const toggleBtn = row.querySelector('[data-action="toggle-send"]');
  if (toggleBtn) {
    toggleBtn.addEventListener("click", () => {
      openSendRowId = sendOpen ? null : dev.id;
      renderDevices(trustedDevicesCache);
    });
  }

  row.querySelector('[data-action="remove"]').addEventListener("click", () => removeDevice(dev.id, dev.name));

  const inlineZone = wrap.querySelector(`#inline-dropzone-${dev.id}`);
  if (inlineZone) {
    inlineZone.addEventListener("click", () => {
      selectedTargetForUpload = dev.id;
      document.getElementById("global-file-input").click();
    });
    inlineZone.addEventListener("dragover", (e) => { e.preventDefault(); inlineZone.classList.add("drag-over"); });
    inlineZone.addEventListener("dragleave", () => inlineZone.classList.remove("drag-over"));
    inlineZone.addEventListener("drop", (e) => {
      e.preventDefault();
      inlineZone.classList.remove("drag-over");
      if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        uploadFilesToDevice(dev.id, e.dataTransfer.files);
      }
    });
  }

  return wrap;
}

function createDiscoveredRow(dev) {
  const wrap = document.createElement("div");
  wrap.innerHTML = `
    <div class="row-card">
      <div class="row-icon">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>
      </div>
      <div style="flex:1;min-width:0;">
        <div class="row-title">${escapeHtml(dev.name)}</div>
        <div class="row-sub">${escapeHtml(dev.addr)}</div>
      </div>
      <button class="btn btn-primary btn-sm" data-action="pair">Parear</button>
    </div>
  `;
  wrap.querySelector('[data-action="pair"]').addEventListener("click", () => {
    const modalReq = document.getElementById("modal-request-pair");
    document.getElementById("input-target-addr").value = dev.addr;
    document.getElementById("pair-flow-state").classList.add("hidden");
    document.getElementById("pair-error-msg").classList.add("hidden");
    document.getElementById("btn-submit-req").style.display = "inline-flex";
    modalReq.classList.remove("hidden");
  });
  return wrap;
}

async function approvePin(pin, modal, errorElem) {
  try {
    const resp = await fetch("/api/v1/pair/approve-pin", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pin })
    });
    const data = await resp.json();

    if (!resp.ok) {
      if (errorElem) {
        errorElem.textContent = data.error || "PIN inválido ou expirado.";
        errorElem.classList.remove("hidden");
      } else {
        showToast("Erro: " + (data.error || "PIN inválido"), "error");
      }
      return;
    }

    if (modal) modal.classList.add("hidden");
    document.getElementById("pairing-banner").classList.add("hidden");
    showToast(`Pareamento aprovado com sucesso com '${data.device_name}'!`, "success");
    await refreshDevicesAndStatus();
  } catch (err) {
    if (errorElem) {
      errorElem.textContent = "Falha ao conectar: " + err.message;
      errorElem.classList.remove("hidden");
    }
  }
}

async function removeDevice(deviceId, deviceName) {
  const confirmed = await showConfirm(
    "Remover dispositivo",
    `Remover o pareamento com '${deviceName}'? Este dispositivo deixará de ser confiável e precisará ser pareado novamente para voltar a sincronizar.`,
    "Remover"
  );
  if (!confirmed) return;

  try {
    const resp = await fetch("/api/v1/devices/remove", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ device_id: deviceId })
    });

    if (!resp.ok) {
      const data = await resp.json().catch(() => ({}));
      showToast("Erro ao remover dispositivo: " + (data.error || resp.statusText), "error");
      return;
    }

    if (openSendRowId === deviceId) openSendRowId = null;
    await refreshDevicesAndStatus();
  } catch (err) {
    showToast("Falha de rede ao remover dispositivo: " + err.message, "error");
  }
}

// ---------------------------------------------------------------------
// Files panel
// ---------------------------------------------------------------------

function populateFilesTargetSelect(trusted) {
  const select = document.getElementById("files-target-select");
  const previous = select.value;
  select.innerHTML = "";

  if (trusted.length === 0) {
    select.innerHTML = `<option value="">Nenhum dispositivo pareado</option>`;
    return;
  }

  trusted.forEach((dev) => {
    const opt = document.createElement("option");
    opt.value = dev.id;
    opt.textContent = `${dev.name} · ${dev.is_online ? "Online" : "Offline"}`;
    if (!dev.is_online) opt.disabled = true;
    select.appendChild(opt);
  });

  if (previous && Array.from(select.options).some((o) => o.value === previous)) {
    select.value = previous;
  }
}

function uploadFilesToDevice(deviceId, files, progressBarId, progressFillId) {
  for (let i = 0; i < files.length; i++) {
    sendFile(deviceId, files[i], progressBarId, progressFillId);
  }
}

function sendFile(deviceId, file, progressBarId, progressFillId) {
  const progressBar = document.getElementById(progressBarId || `progress-${deviceId}`);
  const progressFill = document.getElementById(progressFillId || `fill-${deviceId}`);

  if (progressBar && progressFill) {
    progressBar.classList.remove("hidden");
    progressFill.style.width = "0%";
  }

  const xhr = new XMLHttpRequest();
  xhr.open("POST", `/api/v1/devices/send?target=${encodeURIComponent(deviceId)}&filename=${encodeURIComponent(file.name)}`);

  xhr.upload.onprogress = (e) => {
    if (e.lengthComputable && progressFill) {
      const pct = Math.round((e.loaded / e.total) * 100);
      progressFill.style.width = `${pct}%`;
    }
  };

  xhr.onload = () => {
    if (xhr.status === 200) {
      if (progressFill) progressFill.style.width = "100%";
      setTimeout(() => {
        if (progressBar) progressBar.classList.add("hidden");
        showToast(`Arquivo '${file.name}' enviado com sucesso!`, "success");
      }, 500);
    } else {
      showToast(`Falha ao enviar '${file.name}': ${xhr.responseText}`, "error");
      if (progressBar) progressBar.classList.add("hidden");
    }
  };

  xhr.onerror = () => {
    showToast(`Erro de conexão ao enviar '${file.name}'.`, "error");
    if (progressBar) progressBar.classList.add("hidden");
  };

  xhr.send(file);
}

// ---------------------------------------------------------------------
// Clipboard history
// ---------------------------------------------------------------------

async function refreshClipboardHistory() {
  try {
    const resp = await fetch("/api/v1/clipboard/history");
    if (!resp.ok) return;
    const history = await resp.json();
    renderClipboardHistory(history || []);
  } catch (err) {
    console.warn("Erro ao atualizar histórico de clipboard:", err);
  }
}

function renderClipboardHistory(history) {
  const list = document.getElementById("clipboard-history-list");
  document.getElementById("history-count-badge").textContent = history.length;

  if (history.length === 0) {
    list.innerHTML = `<div class="empty-placeholder"><p>Nada copiado ainda.</p></div>`;
    return;
  }

  list.innerHTML = "";
  history.forEach((item) => {
    const row = document.createElement("div");
    row.className = "row-card";
    row.innerHTML = `
      <div style="flex:1;min-width:0;">
        <div class="row-text">${escapeHtml(item.text)}</div>
        <div class="row-sub">${escapeHtml(item.origin)} &middot; ${formatRelativeTime(item.time)}</div>
      </div>
      <button class="btn btn-outline btn-sm" data-action="copy" style="flex:0 0 auto;white-space:nowrap;">Copiar novamente</button>
      <button class="btn-icon" data-action="remove" title="Remover do histórico">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    `;

    const copyBtn = row.querySelector('[data-action="copy"]');
    copyBtn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(item.text);
        copyBtn.textContent = "Copiado ✓";
        copyBtn.style.color = "var(--success)";
        copyBtn.style.borderColor = "var(--success)";
        setTimeout(() => {
          copyBtn.textContent = "Copiar novamente";
          copyBtn.style.color = "";
          copyBtn.style.borderColor = "";
        }, 1600);
      } catch (err) {
        showToast("Não foi possível copiar automaticamente. Selecione o texto manualmente.", "error");
      }
    });

    row.querySelector('[data-action="remove"]').addEventListener("click", async () => {
      await fetch("/api/v1/clipboard/history/remove", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: item.id })
      });
      refreshClipboardHistory();
    });

    list.appendChild(row);
  });
}

function formatRelativeTime(isoString) {
  if (!isoString) return "";
  const then = new Date(isoString).getTime();
  if (Number.isNaN(then)) return "";
  const diffMs = Date.now() - then;
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "agora";
  if (diffMin < 60) return `há ${diffMin} min`;
  const diffH = Math.floor(diffMin / 60);
  if (diffH < 24) return `há ${diffH} h`;
  const diffD = Math.floor(diffH / 24);
  return `há ${diffD} d`;
}

function escapeHtml(str) {
  if (!str) return "";
  return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

// ---------------------------------------------------------------------
// Input Sharing (KVM): permissions, screen arrangement, escape settings.
// ---------------------------------------------------------------------

const KVM_TOUCH_TOLERANCE_PX = 12;
const KVM_DEFAULT_NODE_SIZE = { w: 1920, h: 1080 };
let kvmLayoutNodes = {}; // id -> { leftPx, topPx, widthRes, heightRes }
let kvmLocalNodeID = null;

function setupKvmEventListeners() {
  document.getElementById("btn-save-layout").addEventListener("click", saveKvmLayout);
  document.getElementById("kvm-hotcorner-select").addEventListener("change", saveKvmLayout);
}

function kvmCanvasPositionKey() {
  return "omnidesk-kvm-canvas-positions";
}

function loadStoredCanvasPositions() {
  try {
    const raw = localStorage.getItem(kvmCanvasPositionKey());
    return raw ? JSON.parse(raw) : {};
  } catch (e) {
    return {};
  }
}

function storeCanvasPositions() {
  try {
    const positions = {};
    for (const id in kvmLayoutNodes) {
      positions[id] = { left: kvmLayoutNodes[id].leftPx, top: kvmLayoutNodes[id].topPx };
    }
    localStorage.setItem(kvmCanvasPositionKey(), JSON.stringify(positions));
  } catch (e) {
    // best-effort convenience only
  }
}

async function refreshKvm() {
  try {
    const [statusResp, pendingResp, layoutResp] = await Promise.all([
      fetch("/api/v1/input/status"),
      fetch("/api/v1/input/permission/pending"),
      fetch("/api/v1/input/layout")
    ]);

    if (statusResp.ok) {
      renderKvmStatus(await statusResp.json());
    }
    if (pendingResp.ok) {
      renderKvmPending(await pendingResp.json());
    }
    if (layoutResp.ok) {
      const layout = await layoutResp.json();
      kvmLocalNodeID = layout.local_node || kvmLocalNodeID;
      mergeKvmLayoutFromServer(layout);
      document.getElementById("kvm-hotcorner-select").value = layout.hot_corner || "";
    }

    renderKvmPermissionList();
    renderKvmCanvas();
  } catch (err) {
    console.warn("Erro ao atualizar KVM:", err);
  }
}

function renderKvmStatus(status) {
  const badge = document.getElementById("kvm-status-badge");
  if (status.unavailable) {
    badge.textContent = status.unavailable_reason || "Indisponível nesta plataforma";
    badge.className = "badge";
    badge.style.color = "var(--danger)";
    badge.style.borderColor = "var(--danger)";
    badge.title = status.unavailable_reason || "";
    return;
  }
  badge.title = "";
  if (!status.active) {
    badge.textContent = "Inativo";
    badge.className = "badge badge-secondary";
    badge.style.color = "";
    badge.style.borderColor = "";
    return;
  }
  const dev = trustedDevicesCache.find((d) => d.id === status.peer_id);
  const name = dev ? dev.name : status.peer_id.substring(0, 8);
  badge.textContent = status.sending ? `Controlando ${name}` : `Sendo controlado por ${name}`;
  badge.className = "badge";
  badge.style.color = "var(--success)";
  badge.style.borderColor = "var(--success)";
}

function renderKvmPending(pending) {
  const el = document.getElementById("kvm-pending-requests");
  el.innerHTML = "";
  (pending || []).forEach((req) => {
    const row = document.createElement("div");
    row.className = "kvm-pending-row";
    row.innerHTML = `
      <span><strong>${escapeHtml(req.peer_name || req.peer_id)}</strong> pediu permissão para controlar este computador.</span>
      <span class="permission-actions">
        <button class="btn btn-sm btn-success" data-action="approve">Aprovar</button>
        <button class="btn btn-sm btn-outline" data-action="deny">Recusar</button>
      </span>
    `;
    row.querySelector('[data-action="approve"]').addEventListener("click", async () => {
      await kvmPost("/api/v1/input/permission/approve", { device_id: req.peer_id });
      await refreshKvm();
    });
    row.querySelector('[data-action="deny"]').addEventListener("click", async () => {
      await kvmPost("/api/v1/input/permission/deny", { device_id: req.peer_id });
      await refreshKvm();
    });
    el.appendChild(row);
  });
}

function renderKvmPermissionList() {
  const el = document.getElementById("kvm-permission-list");
  el.innerHTML = "";

  if (trustedDevicesCache.length === 0) {
    el.innerHTML = `<div class="empty-placeholder"><p>Pareie um dispositivo para liberar controle de mouse/teclado.</p></div>`;
    return;
  }

  trustedDevicesCache.forEach((dev) => {
    const row = document.createElement("div");
    row.className = "permission-row";
    row.innerHTML = `
      <span>${escapeHtml(dev.name)}</span>
      <span class="permission-actions">
        <button class="btn btn-sm btn-outline" data-action="ask">Pedir controle dele</button>
        <button class="btn btn-sm btn-outline" data-action="pause">Pausar/Retomar</button>
      </span>
    `;
    row.querySelector('[data-action="ask"]').addEventListener("click", async () => {
      const r = await kvmPost("/api/v1/input/permission/ask", { device_id: dev.id });
      if (!r.ok) showToast("Não foi possível solicitar controle: " + (r.data && r.data.error ? r.data.error : "dispositivo offline?"), "error");
    });
    row.querySelector('[data-action="pause"]').addEventListener("click", async () => {
      await kvmPost("/api/v1/input/pause", { device_id: dev.id });
    });
    el.appendChild(row);
  });
}

// KVM_BOX_W/H match the fixed on-screen footprint of .kvm-node in
// style.css — the same numbers detectKvmEdge already assumes when reading
// positions back. computePositionsFromLinks uses them to lay boxes out
// actually touching along whatever server-configured links exist.
const KVM_BOX_W = 128;
const KVM_BOX_H = 84;
const KVM_BOX_GAP = 2;

// computePositionsFromLinks derives canvas pixel positions from the
// server's real adjacency graph (BFS from a seed node, placing each
// linked neighbor touching the edge its link names), instead of trusting
// whatever this browser happens to have cached. Without this, a browser
// that never dragged a box (a fresh profile, or a layout configured via
// the API/CLI instead of this dashboard — see tasks.md 9.1) renders boxes
// at their untouched default grid spot, and clicking "Salvar" there
// recomputes links from THOSE positions — silently wiping the real
// configuration with an empty one (tasks.md 7.10).
function computePositionsFromLinks(nodeIDs, links, seedPos) {
  const adjacency = {};
  nodeIDs.forEach((id) => { adjacency[id] = []; });
  links.forEach((l) => {
    if (adjacency[l.from_node]) {
      adjacency[l.from_node].push({ edge: l.from_edge, to: l.to_node });
    }
  });

  const seed = kvmLocalNodeID && nodeIDs.includes(kvmLocalNodeID) ? kvmLocalNodeID : nodeIDs[0];
  const positions = { [seed]: { left: seedPos.left, top: seedPos.top } };
  const queue = [seed];

  while (queue.length) {
    const id = queue.shift();
    const pos = positions[id];
    adjacency[id].forEach(({ edge, to }) => {
      if (positions[to]) return; // already placed via some other path
      let left = pos.left, top = pos.top;
      switch (edge) {
        case "right": left = pos.left + KVM_BOX_W + KVM_BOX_GAP; break;
        case "left": left = pos.left - KVM_BOX_W - KVM_BOX_GAP; break;
        case "bottom": top = pos.top + KVM_BOX_H + KVM_BOX_GAP; break;
        case "top": top = pos.top - KVM_BOX_H - KVM_BOX_GAP; break;
      }
      positions[to] = { left, top };
      queue.push(to);
    });
  }

  return positions;
}

function mergeKvmLayoutFromServer(layout) {
  const stored = loadStoredCanvasPositions();
  const serverNodes = layout.nodes || {};
  const serverLinks = layout.links || [];
  const knownIDs = new Set(Object.keys(kvmLayoutNodes));

  // Ensure the local node and every trusted device has a canvas entry.
  const allIDs = new Set([kvmLocalNodeID, ...trustedDevicesCache.map((d) => d.id)].filter(Boolean));
  allIDs.forEach((id) => knownIDs.add(id));

  const idsWithLinks = new Set();
  serverLinks.forEach((l) => { idsWithLinks.add(l.from_node); idsWithLinks.add(l.to_node); });
  const linkedPositions = idsWithLinks.size > 0
    ? computePositionsFromLinks(
        Array.from(idsWithLinks), serverLinks,
        stored[kvmLocalNodeID] || defaultKvmGridPosition(0)
      )
    : {};

  let i = 0;
  knownIDs.forEach((id) => {
    let res = serverNodes[id] || KVM_DEFAULT_NODE_SIZE_FOR(id, layout);
    if (id === kvmLocalNodeID && localNode && localNode.screen_width && localNode.screen_height) {
      res = { width_px: localNode.screen_width, height_px: localNode.screen_height };
    } else {
      const dev = trustedDevicesCache.find((d) => d.id === id);
      if (dev && dev.screen_width && dev.screen_height) {
        res = { width_px: dev.screen_width, height_px: dev.screen_height };
      }
    }
    // A node the server's links actually place always wins over a stale
    // (or absent) localStorage snapshot — see comment on
    // computePositionsFromLinks above.
    const pos = linkedPositions[id] || stored[id] || defaultKvmGridPosition(i);
    kvmLayoutNodes[id] = {
      leftPx: pos.left,
      topPx: pos.top,
      widthRes: res.width_px || KVM_DEFAULT_NODE_SIZE.w,
      heightRes: res.height_px || KVM_DEFAULT_NODE_SIZE.h
    };
    i++;
  });

  // Persist the derived positions: reloading stays stable, and a save
  // from right here recomputes the SAME links instead of different ones.
  storeCanvasPositions();
}

function KVM_DEFAULT_NODE_SIZE_FOR(id, layout) {
  return (layout.nodes && layout.nodes[id]) || {};
}

function defaultKvmGridPosition(index) {
  const col = index % 4;
  const row = Math.floor(index / 4);
  return { left: 20 + col * 150, top: 20 + row * 100 };
}

function kvmDeviceName(id) {
  if (id === kvmLocalNodeID) return (localNode && localNode.device_name) || "Este computador";
  const dev = trustedDevicesCache.find((d) => d.id === id);
  return dev ? dev.name : id.substring(0, 8);
}

function renderKvmCanvas() {
  const canvas = document.getElementById("kvm-layout-canvas");
  canvas.innerHTML = "";

  Object.keys(kvmLayoutNodes).forEach((id) => {
    const n = kvmLayoutNodes[id];
    const box = document.createElement("div");
    box.className = "kvm-node" + (id === kvmLocalNodeID ? " local" : "");
    box.style.left = n.leftPx + "px";
    box.style.top = n.topPx + "px";
    box.dataset.nodeId = id;
    box.innerHTML = `
      <div class="kvm-node-name">${escapeHtml(kvmDeviceName(id))}</div>
      <div class="kvm-node-res">${n.widthRes}x${n.heightRes}</div>
    `;
    makeKvmNodeDraggable(box, canvas);
    canvas.appendChild(box);
  });
}

function makeKvmNodeDraggable(box, canvas) {
  let dragging = false;
  let offsetX = 0, offsetY = 0;

  box.addEventListener("mousedown", (e) => {
    dragging = true;
    box.classList.add("dragging");
    const rect = box.getBoundingClientRect();
    offsetX = e.clientX - rect.left;
    offsetY = e.clientY - rect.top;
    e.preventDefault();
  });

  document.addEventListener("mousemove", (e) => {
    if (!dragging) return;
    const canvasRect = canvas.getBoundingClientRect();
    let left = e.clientX - canvasRect.left - offsetX;
    let top = e.clientY - canvasRect.top - offsetY;
    left = Math.max(0, Math.min(left, canvasRect.width - box.offsetWidth));
    top = Math.max(0, Math.min(top, canvasRect.height - box.offsetHeight));
    box.style.left = left + "px";
    box.style.top = top + "px";
    const id = box.dataset.nodeId;
    if (kvmLayoutNodes[id]) {
      kvmLayoutNodes[id].leftPx = left;
      kvmLayoutNodes[id].topPx = top;
    }
  });

  document.addEventListener("mouseup", () => {
    if (!dragging) return;
    dragging = false;
    box.classList.remove("dragging");
    storeCanvasPositions();
  });
}

// Detects which edges are "touching" between two node rectangles (within
// KVM_TOUCH_TOLERANCE_PX) and returns the link direction, mirroring the
// backend's Edge model (specs/screen-layout: adjacency by touching edges).
function detectKvmEdge(a, b) {
  const aBox = { l: a.leftPx, t: a.topPx, r: a.leftPx + KVM_BOX_W, bo: a.topPx + KVM_BOX_H };
  const bBox = { l: b.leftPx, t: b.topPx, r: b.leftPx + KVM_BOX_W, bo: b.topPx + KVM_BOX_H };

  const verticalOverlap = Math.min(aBox.bo, bBox.bo) - Math.max(aBox.t, bBox.t) > 0;
  const horizontalOverlap = Math.min(aBox.r, bBox.r) - Math.max(aBox.l, bBox.l) > 0;

  if (verticalOverlap && Math.abs(aBox.r - bBox.l) <= KVM_TOUCH_TOLERANCE_PX) {
    return "right"; // a's right edge touches b's left edge
  }
  if (verticalOverlap && Math.abs(aBox.l - bBox.r) <= KVM_TOUCH_TOLERANCE_PX) {
    return "left";
  }
  if (horizontalOverlap && Math.abs(aBox.bo - bBox.t) <= KVM_TOUCH_TOLERANCE_PX) {
    return "bottom";
  }
  if (horizontalOverlap && Math.abs(aBox.t - bBox.bo) <= KVM_TOUCH_TOLERANCE_PX) {
    return "top";
  }
  return null;
}

const KVM_OPPOSITE_EDGE = { top: "bottom", bottom: "top", left: "right", right: "left" };

function computeKvmLinksFromCanvas() {
  const ids = Object.keys(kvmLayoutNodes);
  const links = [];
  for (let i = 0; i < ids.length; i++) {
    for (let j = 0; j < ids.length; j++) {
      if (i === j) continue;
      const edge = detectKvmEdge(kvmLayoutNodes[ids[i]], kvmLayoutNodes[ids[j]]);
      if (edge) {
        links.push({
          from_node: ids[i], from_edge: edge,
          to_node: ids[j], to_edge: KVM_OPPOSITE_EDGE[edge],
          offset: 0
        });
      }
    }
  }
  return links;
}

async function saveKvmLayout() {
  const nodes = {};
  Object.keys(kvmLayoutNodes).forEach((id) => {
    nodes[id] = { width_px: kvmLayoutNodes[id].widthRes, height_px: kvmLayoutNodes[id].heightRes };
  });

  const payload = {
    nodes,
    links: computeKvmLinksFromCanvas(),
    hotkey_hid: [],
    hot_corner: document.getElementById("kvm-hotcorner-select").value
  };

  const r = await kvmPost("/api/v1/input/layout", payload);
  if (!r.ok) {
    showToast("Falha ao salvar arranjo: " + (r.data && r.data.error ? r.data.error : r.status), "error");
  } else {
    showToast("Arranjo de telas salvo.", "success");
  }
}

async function kvmPost(url, body) {
  try {
    const resp = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    let data = {};
    try { data = await resp.json(); } catch (e) { /* no body */ }
    return { ok: resp.ok, status: resp.status, data };
  } catch (err) {
    return { ok: false, status: 0, data: { error: err.message } };
  }
}

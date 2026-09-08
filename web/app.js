// OmniDesk Webview Dashboard Application

let localNode = null;
let currentPendingPIN = null;
let selectedTargetForUpload = null;

document.addEventListener("DOMContentLoaded", () => {
  initApp();
  setupEventListeners();
  // Poll every 3 seconds
  setInterval(refreshDevicesAndStatus, 3000);
});

async function initApp() {
  await refreshDevicesAndStatus();
}

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

  // Clipboard toggle
  const clipToggle = document.getElementById("clipboard-toggle");
  clipToggle.addEventListener("change", async () => {
    try {
      const resp = await fetch("/api/v1/clipboard/toggle", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: clipToggle.checked })
      });
      const data = await resp.json();
      updateClipboardUI(data.clipboard_sync);
    } catch (e) {
      console.error("Falha ao alterar sincronização de clipboard:", e);
    }
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

  // Hidden File input change
  const fileInput = document.getElementById("global-file-input");
  fileInput.addEventListener("change", () => {
    if (fileInput.files && fileInput.files.length > 0 && selectedTargetForUpload) {
      uploadFilesToDevice(selectedTargetForUpload, fileInput.files);
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
      updateClipboardUI(localNode.clipboard_sync);
      if (localNode.download_dir) {
        document.getElementById("footer-storage-path").textContent = `Pasta de Downloads: ${localNode.download_dir}`;
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

function updateClipboardUI(enabled) {
  const toggle = document.getElementById("clipboard-toggle");
  const label = document.getElementById("clipboard-status-label");
  toggle.checked = !!enabled;
  if (enabled) {
    label.textContent = "Ativo";
    label.style.color = "var(--success)";
  } else {
    label.textContent = "Pausado";
    label.style.color = "var(--text-muted)";
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

function renderDevices(trusted, discovered) {
  const trustedList = document.getElementById("trusted-devices-list");
  const discoveredList = document.getElementById("discovered-devices-list");
  const emptyState = document.getElementById("empty-trusted-state");

  document.getElementById("trusted-count-badge").textContent = `${trusted.length} dispositivo${trusted.length === 1 ? '' : 's'}`;

  // Filter discovered to exclude already trusted
  const trustedIDs = new Set(trusted.map(d => d.id));
  const unpairedDiscovered = discovered.filter(d => !trustedIDs.has(d.id) && d.id !== (localNode ? localNode.device_id : ""));

  document.getElementById("discovered-count-badge").textContent = `${unpairedDiscovered.length} na rede`;

  // Render Trusted
  trustedList.innerHTML = "";
  if (trusted.length === 0) {
    trustedList.appendChild(emptyState);
  } else {
    trusted.forEach(dev => {
      const card = createTrustedCard(dev);
      trustedList.appendChild(card);
    });
  }

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
    unpairedDiscovered.forEach(dev => {
      const card = createDiscoveredCard(dev);
      discoveredList.appendChild(card);
    });
  }
}

function createTrustedCard(dev) {
  const card = document.createElement("div");
  card.className = "device-card";
  card.dataset.deviceId = dev.id;

  const isOnline = dev.is_online;
  const statusColor = isOnline ? "var(--success)" : "var(--text-muted)";
  const statusLabel = isOnline ? "Online" : "Offline";
  const osType = (dev.name && dev.name.toLowerCase().includes("mac")) ? "macOS" : "Linux / PC";

  card.innerHTML = `
    <div class="device-header">
      <div class="device-meta">
        <div class="device-os-icon">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
            <line x1="8" y1="21" x2="16" y2="21"/>
            <line x1="12" y1="17" x2="12" y2="21"/>
          </svg>
        </div>
        <div>
          <div class="device-title">${escapeHtml(dev.name)}</div>
          <div class="device-sub">${osType} &bull; ${escapeHtml(dev.last_addr || "Endereço desc.")}</div>
        </div>
      </div>
      <span class="badge" style="color: ${statusColor}; border-color: ${statusColor}; background: transparent;">
        ${statusLabel}
      </span>
    </div>

    <div class="device-details">
      <div><strong>ID:</strong> <code>${dev.id.substring(0, 16)}...</code></div>
      <div><strong>Último Contato:</strong> ${dev.last_seen || "Recentemente"}</div>
    </div>

    <div class="card-dropzone" id="dropzone-${dev.id}">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="17 8 12 3 7 8"/>
        <line x1="12" y1="3" x2="12" y2="15"/>
      </svg>
      <div class="dropzone-label">Arraste arquivos aqui</div>
      <div class="dropzone-hint">ou clique para selecionar e enviar para este nó</div>
      <div class="transfer-progress-bar hidden" id="progress-${dev.id}">
        <div class="transfer-progress-fill" id="fill-${dev.id}"></div>
      </div>
    </div>
  `;

  // Drag and Drop listeners
  const dropzone = card.querySelector(`#dropzone-${dev.id}`);

  card.addEventListener("dragover", (e) => {
    e.preventDefault();
    card.classList.add("drag-over");
  });

  card.addEventListener("dragleave", (e) => {
    if (!card.contains(e.relatedTarget)) {
      card.classList.remove("drag-over");
    }
  });

  card.addEventListener("drop", (e) => {
    e.preventDefault();
    card.classList.remove("drag-over");
    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      uploadFilesToDevice(dev.id, e.dataTransfer.files);
    }
  });

  dropzone.addEventListener("click", () => {
    selectedTargetForUpload = dev.id;
    const fileInput = document.getElementById("global-file-input");
    fileInput.click();
  });

  return card;
}

function createDiscoveredCard(dev) {
  const card = document.createElement("div");
  card.className = "device-card";

  card.innerHTML = `
    <div class="device-header">
      <div class="device-meta">
        <div class="device-os-icon">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 16v-4M12 8h.01"/>
          </svg>
        </div>
        <div>
          <div class="device-title">${escapeHtml(dev.name)}</div>
          <div class="device-sub">${escapeHtml(dev.addr)}</div>
        </div>
      </div>
      <span class="badge badge-secondary">Não pareado</span>
    </div>

    <div class="device-details">
      <div><strong>ID:</strong> <code>${dev.id.substring(0, 16)}...</code></div>
    </div>

    <button class="btn btn-primary btn-sm btn-pair-peer" style="margin-top:auto; width:100%; justify-content:center;">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
        <circle cx="8.5" cy="7" r="4"/>
        <line x1="20" y1="8" x2="20" y2="14"/>
        <line x1="23" y1="11" x2="17" y2="11"/>
      </svg>
      Parear com este nó
    </button>
  `;

  card.querySelector(".btn-pair-peer").addEventListener("click", () => {
    const modalReq = document.getElementById("modal-request-pair");
    document.getElementById("input-target-addr").value = dev.addr;
    document.getElementById("pair-flow-state").classList.add("hidden");
    document.getElementById("pair-error-msg").classList.add("hidden");
    document.getElementById("btn-submit-req").style.display = "inline-flex";
    modalReq.classList.remove("hidden");
  });

  return card;
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
        alert("Erro: " + (data.error || "PIN inválido"));
      }
      return;
    }

    if (modal) modal.classList.add("hidden");
    document.getElementById("pairing-banner").classList.add("hidden");
    alert(`Pareamento aprovado com sucesso com '${data.device_name}'!`);
    await refreshDevicesAndStatus();
  } catch (err) {
    if (errorElem) {
      errorElem.textContent = "Falha ao conectar: " + err.message;
      errorElem.classList.remove("hidden");
    }
  }
}

function uploadFilesToDevice(deviceId, files) {
  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    sendFile(deviceId, file);
  }
}

function sendFile(deviceId, file) {
  const progressBar = document.getElementById(`progress-${deviceId}`);
  const progressFill = document.getElementById(`fill-${deviceId}`);

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
        alert(`Arquivo '${file.name}' enviado com sucesso!`);
      }, 500);
    } else {
      alert(`Falha ao enviar '${file.name}': ${xhr.responseText}`);
      if (progressBar) progressBar.classList.add("hidden");
    }
  };

  xhr.onerror = () => {
    alert(`Erro de conexão ao enviar '${file.name}'.`);
    if (progressBar) progressBar.classList.add("hidden");
  };

  xhr.send(file);
}

function escapeHtml(str) {
  if (!str) return "";
  return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

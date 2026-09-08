// OmniDesk Webview Dashboard Application

let localNode = null;
let currentPendingPIN = null;
let selectedTargetForUpload = null;
let trustedDevicesCache = [];

document.addEventListener("DOMContentLoaded", () => {
  initApp();
  setupEventListeners();
  setupKvmEventListeners();
  // Poll every 3 seconds
  setInterval(refreshDevicesAndStatus, 3000);
  setInterval(refreshKvm, 3000);
});

async function initApp() {
  await refreshDevicesAndStatus();
  await refreshKvm();
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
  trustedDevicesCache = trusted;
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

    <button class="btn btn-outline btn-sm btn-remove-device">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="3 6 5 6 21 6"/>
        <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
      </svg>
      Remover dispositivo
    </button>

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

  card.querySelector(".btn-remove-device").addEventListener("click", () => {
    removeDevice(dev.id, dev.name);
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

async function removeDevice(deviceId, deviceName) {
  const confirmed = confirm(`Remover o pareamento com '${deviceName}'? Este dispositivo deixará de ser confiável e precisará ser pareado novamente para voltar a sincronizar.`);
  if (!confirmed) return;

  try {
    const resp = await fetch("/api/v1/devices/remove", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ device_id: deviceId })
    });

    if (!resp.ok) {
      const data = await resp.json().catch(() => ({}));
      alert("Erro ao remover dispositivo: " + (data.error || resp.statusText));
      return;
    }

    await refreshDevicesAndStatus();
  } catch (err) {
    alert("Falha de rede ao remover dispositivo: " + err.message);
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
  const dev = trustedDevicesCache.find(d => d.id === status.peer_id);
  const name = dev ? dev.name : status.peer_id.substring(0, 8);
  badge.textContent = status.sending ? `Controlando ${name}` : `Sendo controlado por ${name}`;
  badge.className = "badge";
  badge.style.color = "var(--success)";
  badge.style.borderColor = "var(--success)";
}

function renderKvmPending(pending) {
  const el = document.getElementById("kvm-pending-requests");
  el.innerHTML = "";
  (pending || []).forEach(req => {
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

  trustedDevicesCache.forEach(dev => {
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
      if (!r.ok) alert("Não foi possível solicitar controle: " + (r.data && r.data.error ? r.data.error : "dispositivo offline?"));
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
const KVM_BOX_W = 120;
const KVM_BOX_H = 80;
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
  nodeIDs.forEach(id => { adjacency[id] = []; });
  links.forEach(l => {
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
  const allIDs = new Set([kvmLocalNodeID, ...trustedDevicesCache.map(d => d.id)].filter(Boolean));
  allIDs.forEach(id => knownIDs.add(id));

  const idsWithLinks = new Set();
  serverLinks.forEach(l => { idsWithLinks.add(l.from_node); idsWithLinks.add(l.to_node); });
  const linkedPositions = idsWithLinks.size > 0
    ? computePositionsFromLinks(
        Array.from(idsWithLinks), serverLinks,
        stored[kvmLocalNodeID] || defaultKvmGridPosition(0)
      )
    : {};

  let i = 0;
  knownIDs.forEach(id => {
    const res = serverNodes[id] || KVM_DEFAULT_NODE_SIZE_FOR(id, layout);
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
  const dev = trustedDevicesCache.find(d => d.id === id);
  return dev ? dev.name : id.substring(0, 8);
}

function renderKvmCanvas() {
  const canvas = document.getElementById("kvm-layout-canvas");
  canvas.innerHTML = "";

  Object.keys(kvmLayoutNodes).forEach(id => {
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
  const aBox = { l: a.leftPx, t: a.topPx, r: a.leftPx + 120, bo: a.topPx + 80 };
  const bBox = { l: b.leftPx, t: b.topPx, r: b.leftPx + 120, bo: b.topPx + 80 };

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
  Object.keys(kvmLayoutNodes).forEach(id => {
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
    alert("Falha ao salvar arranjo: " + (r.data && r.data.error ? r.data.error : r.status));
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

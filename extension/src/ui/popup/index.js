import {
  getDevConfig,
  resolveRuntimeConfig,
  saveDevConfig,
} from "../../shared/config.js";
import { getRuntimeState } from "../../shared/runtime-state.js";

const ADMIN_UI_PASSWORD = "forlittle-admin";

const modeLabel = document.getElementById("modeLabel");
const form = document.getElementById("devConfigForm");
const adminUnlockForm = document.getElementById("adminUnlockForm");
const adminGatePanel = document.getElementById("adminGatePanel");
const configPanel = document.getElementById("configPanel");
const adminPasswordInput = document.getElementById("adminPassword");
const machineIdInput = document.getElementById("machineId");
const displayNameInput = document.getElementById("displayName");
const serverBaseUrlInput = document.getElementById("serverBaseUrl");
const savedMessage = document.getElementById("savedMessage");
const displayNameValue = document.getElementById("displayNameValue");
const registrationStatus = document.getElementById("registrationStatus");
const policyVersion = document.getElementById("policyVersion");
const pendingLogs = document.getElementById("pendingLogs");
const logFlushStatus = document.getElementById("logFlushStatus");
const lastLogFlushAt = document.getElementById("lastLogFlushAt");
const lastError = document.getElementById("lastError");
const refreshRuntimeButton = document.getElementById("refreshRuntimeButton");
const syncPolicyButton = document.getElementById("syncPolicyButton");
const flushLogsButton = document.getElementById("flushLogsButton");
let isAdminUnlocked = false;

async function sendRuntimeMessage(message) {
  const response = await chrome.runtime.sendMessage(message).catch(() => null);
  if (!response?.ok) {
    throw new Error(response?.error || "Runtime request failed");
  }

  return response;
}

function renderAdminPanels() {
  adminGatePanel.hidden = isAdminUnlocked;
  configPanel.hidden = !isAdminUnlocked;
}

function formatDateTime(value) {
  if (!value) {
    return "-";
  }

  return new Date(value).toLocaleString();
}

function queueStatusLabel(status, pendingCount) {
  if (status === "syncing") {
    return "Syncing logs...";
  }

  if (status === "error") {
    return "Retrying automatically";
  }

  if (pendingCount > 0) {
    return "Waiting for next flush";
  }

  return "Idle";
}

async function render() {
  const [config, runtimeState, devConfig] = await Promise.all([
    resolveRuntimeConfig(),
    getRuntimeState(),
    getDevConfig(),
  ]);

  modeLabel.textContent = `Mode: ${config.mode}`;
  displayNameValue.textContent = config.displayName || devConfig.displayName || "-";
  registrationStatus.textContent = runtimeState.registrationStatus;
  policyVersion.textContent = String(runtimeState.policyVersion || 0);
  const pendingLogCount = runtimeState.pendingLogs || 0;
  pendingLogs.textContent = String(pendingLogCount);
  logFlushStatus.textContent = queueStatusLabel(runtimeState.logFlushStatus, pendingLogCount);
  lastLogFlushAt.textContent = formatDateTime(runtimeState.lastLogFlushAt);
  lastError.textContent = runtimeState.lastError || "-";
  renderAdminPanels();

  if (config.mode === "managed" || config.mode === "dev_local") {
    machineIdInput.value = config.machineId || "";
    displayNameInput.value = config.displayName || "";
    serverBaseUrlInput.value = config.serverBaseUrl || "";
    return;
  }

  machineIdInput.value = devConfig.machineId || "";
  displayNameInput.value = devConfig.displayName || "";
  serverBaseUrlInput.value = devConfig.serverBaseUrl || "";
}

adminUnlockForm.addEventListener("submit", (event) => {
  event.preventDefault();

  if (adminPasswordInput.value !== ADMIN_UI_PASSWORD) {
    savedMessage.textContent = "Invalid admin password.";
    isAdminUnlocked = false;
    renderAdminPanels();
    adminPasswordInput.select();
    return;
  }

  isAdminUnlocked = true;
  savedMessage.textContent = "Admin config unlocked for this popup session.";
  renderAdminPanels();
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  try {
    await saveDevConfig({
      machineId: machineIdInput.value.trim(),
      displayName: displayNameInput.value.trim(),
      serverBaseUrl: serverBaseUrlInput.value.trim(),
    });

    await chrome.runtime
      .sendMessage({ type: "reinitialize-runtime" })
      .catch(() => null);
    savedMessage.textContent = "Saved. Extension has reconnected with the new settings.";
    await render();
  } catch (error) {
    savedMessage.textContent = error.message || "Could not save config.";
  }
});

refreshRuntimeButton.addEventListener("click", async () => {
  try {
    await sendRuntimeMessage({ type: "reinitialize-runtime" });
    savedMessage.textContent = "Extension reconnected.";
    await render();
  } catch (error) {
    savedMessage.textContent = error.message || "Reconnect failed.";
  }
});

syncPolicyButton.addEventListener("click", async () => {
  try {
    await sendRuntimeMessage({ type: "sync-policy-now" });
    savedMessage.textContent = "Policy synced.";
    await render();
  } catch (error) {
    savedMessage.textContent = error.message || "Policy sync failed.";
  }
});

flushLogsButton.addEventListener("click", async () => {
  try {
    const response = await sendRuntimeMessage({ type: "flush-logs-now" });
    savedMessage.textContent = `Flush complete. Sent ${response.sent || 0} log(s).`;
    await render();
  } catch (error) {
    savedMessage.textContent = error.message || "Log flush failed.";
  }
});

chrome.storage.onChanged.addListener((changes, areaName) => {
  if (areaName === "local" && (changes.runtimeState || changes.devConfig)) {
    render().catch(console.error);
  }
});

render().catch(console.error);

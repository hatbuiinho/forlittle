import { resolveRuntimeConfig } from "../shared/config.js";
import { ALARM_HEARTBEAT, ALARM_LOG_FLUSH, ALARM_POLICY_SYNC, LOG_FLUSH_BATCH_SIZE } from "../shared/constants.js";
import { fetchPolicy, HttpError, registerAgent, sendHeartbeat, sendLogBatch } from "../shared/network.js";
import { evaluatePolicy, domainFromUrl } from "../shared/rules.js";
import { getRuntimeState, patchRuntimeState } from "../shared/runtime-state.js";
import { clearDeviceToken, enqueueLog, getDeviceToken, getOrCreateProfileInstanceId, getPolicy, peekLogs, removeLogs, setPolicy } from "../shared/storage.js";

let runtimeConfig = { mode: "unconfigured" };
let profileInstanceId = "";
let initializationPromise = null;
let lastConfigSignature = "";
const pendingVisitTimers = new Map();
const lastLoggedVisitByTab = new Map();
const VISIT_CAPTURE_DELAY_MS = 500;

function configSignature(config) {
  return [
    config.mode,
    config.machineId || "",
    config.displayName || "",
    config.serverBaseUrl || ""
  ].join("|");
}

async function setBadge(text) {
  await chrome.action.setBadgeText({ text });
}

async function configureAlarms() {
  await chrome.alarms.clearAll();
  await chrome.alarms.create(ALARM_HEARTBEAT, { periodInMinutes: 5 });
  await chrome.alarms.create(ALARM_POLICY_SYNC, { periodInMinutes: 5 });
  await chrome.alarms.create(ALARM_LOG_FLUSH, { periodInMinutes: 1 });
}

async function bootstrap(forceRefresh = false) {
  runtimeConfig = await resolveRuntimeConfig();
  profileInstanceId = await getOrCreateProfileInstanceId();
  const nextSignature = configSignature(runtimeConfig);
  const runtimeState = await getRuntimeState();
  const previousSignature = configSignature(runtimeState);
  const configChanged = previousSignature && nextSignature !== previousSignature;

  if (configChanged) {
    lastConfigSignature = nextSignature;
    await clearDeviceToken();
  } else {
    lastConfigSignature = nextSignature;
  }

  if (runtimeConfig.mode === "unconfigured") {
    await chrome.alarms.clearAll();
    await patchRuntimeState({
      mode: "unconfigured",
      machineId: "",
      serverBaseUrl: "",
      registrationStatus: "missing_config",
      lastError: "Missing machine ID or server URL",
      policyVersion: 0
    });
    await setBadge("!");
    return;
  }

  await patchRuntimeState({
    mode: runtimeConfig.mode,
    machineId: runtimeConfig.machineId,
    serverBaseUrl: runtimeConfig.serverBaseUrl,
    registrationStatus: "initializing",
    lastError: ""
  });

  await ensureRegistered();
  await syncPolicy();
  await flushLogs();
  await configureAlarms();
  await setBadge("");
}

async function ensureInitialized(forceRefresh = false) {
  if (!initializationPromise || forceRefresh) {
    initializationPromise = bootstrap(forceRefresh).catch(async (error) => {
      await patchRuntimeState({
        registrationStatus: "error",
        lastError: error.message || "Unknown runtime error"
      });
      await setBadge("!");
      throw error;
    });
  }

  return initializationPromise;
}

async function ensureRegistered(forceRefresh = false) {
  const token = forceRefresh ? "" : await getDeviceToken();
  if (token) {
    await patchRuntimeState({
      registrationStatus: "registered"
    });
    return;
  }

  const response = await registerAgent(runtimeConfig, profileInstanceId);
  await patchRuntimeState({
    registrationStatus: response.machine_status === "active" ? "active" : "registered",
    lastRegistrationAt: new Date().toISOString(),
    lastError: ""
  });
}

async function syncPolicy() {
  let policy;

  try {
    policy = await fetchPolicy(runtimeConfig);
  } catch (error) {
    if (error instanceof HttpError && error.status === 401) {
      await clearDeviceToken();
      await ensureRegistered(true);
      policy = await fetchPolicy(runtimeConfig);
    } else {
      throw error;
    }
  }

  await setPolicy(policy);
  await patchRuntimeState({
    policyVersion: policy.policy_version || 0,
    lastSyncAt: new Date().toISOString(),
    lastError: ""
  });
}

async function flushLogs() {
  const events = await peekLogs();
  if (!events.length) {
    await patchRuntimeState({ pendingLogs: 0 });
    return 0;
  }

  const batch = events.slice(0, LOG_FLUSH_BATCH_SIZE);

  try {
    await sendLogBatch(runtimeConfig, batch);
  } catch (error) {
    if (error instanceof HttpError && error.status === 401) {
      await clearDeviceToken();
      await ensureRegistered(true);
      await sendLogBatch(runtimeConfig, batch);
    } else {
      await patchRuntimeState({
        pendingLogs: events.length,
        lastError: error.message || "Could not flush logs"
      });
      throw error;
    }
  }

  await removeLogs(batch.length);
  const remainingLogs = await peekLogs();
  await patchRuntimeState({
    pendingLogs: remainingLogs.length,
    lastError: ""
  });

  return batch.length;
}

async function heartbeat() {
  try {
    await sendHeartbeat(runtimeConfig, profileInstanceId);
    await patchRuntimeState({ lastError: "" });
  } catch (error) {
    if (error instanceof HttpError && error.status === 401) {
      await clearDeviceToken();
      await ensureRegistered(true);
      await sendHeartbeat(runtimeConfig, profileInstanceId);
      await patchRuntimeState({ lastError: "" });
      return;
    }

    await patchRuntimeState({
      lastError: error.message || "Could not send heartbeat"
    });
    throw error;
  }
}

async function handleNavigation(details) {
  await ensureInitialized();

  if (details.frameId !== 0 || !details.url?.startsWith("http")) {
    return;
  }

  await enforcePolicy(details.tabId, details.url);
  scheduleVisitCapture(details.tabId);
}

async function enforcePolicy(tabId, url) {
  if (!url?.startsWith("http")) {
    return;
  }

  const policy = await getPolicy();
  const domain = domainFromUrl(url);
  const result = evaluatePolicy(policy, domain);

  if (result.blocked) {
    const blockUrl = chrome.runtime.getURL(`src/ui/block/index.html?domain=${encodeURIComponent(domain)}`);
    await chrome.tabs.update(tabId, { url: blockUrl });
  }
}

function scheduleVisitCapture(tabId) {
  const previousTimer = pendingVisitTimers.get(tabId);
  if (previousTimer) {
    clearTimeout(previousTimer);
  }

  const timerId = setTimeout(() => {
    pendingVisitTimers.delete(tabId);
    captureVisitFromTab(tabId).catch(console.error);
  }, VISIT_CAPTURE_DELAY_MS);

  pendingVisitTimers.set(tabId, timerId);
}

function shouldSkipVisit(tabId, url, title) {
  const lastVisit = lastLoggedVisitByTab.get(tabId);
  if (!lastVisit) {
    return false;
  }

  const sameUrl = lastVisit.url === url;
  const sameTitle = lastVisit.title === title;
  const visitedRecently = Date.now() - lastVisit.loggedAt < 5000;

  return sameUrl && sameTitle && visitedRecently;
}

async function captureVisitFromTab(tabId) {
  const tab = await chrome.tabs.get(tabId).catch(() => null);
  if (!tab?.url?.startsWith("http")) {
    return;
  }

  const url = tab.url;
  const title = tab.title || url;

  if (shouldSkipVisit(tabId, url, title)) {
    return;
  }

  const policy = await getPolicy();
  const domain = domainFromUrl(url);
  const result = evaluatePolicy(policy, domain);

  await enqueueLog({
    profile_instance_id: profileInstanceId,
    tab_id: tabId,
    url,
    domain,
    title: title || url,
    visited_at: new Date().toISOString(),
    action: result.action
  });

  lastLoggedVisitByTab.set(tabId, {
    url,
    title,
    loggedAt: Date.now()
  });

  const pendingLogs = await peekLogs();
  await patchRuntimeState({ pendingLogs: pendingLogs.length });
}

chrome.runtime.onInstalled.addListener(() => {
  ensureInitialized(true).catch(console.error);
});

chrome.runtime.onStartup.addListener(() => {
  ensureInitialized().catch(console.error);
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === "reinitialize-runtime") {
    ensureInitialized(true)
      .then(() => sendResponse({ ok: true }))
      .catch((error) => sendResponse({ ok: false, error: error.message }));

    return true;
  }

  if (message?.type === "flush-logs-now") {
    ensureInitialized()
      .then(() => flushLogs())
      .then((sent) => sendResponse({ ok: true, sent }))
      .catch((error) => sendResponse({ ok: false, error: error.message }));

    return true;
  }

  return false;
});

chrome.storage.onChanged.addListener((changes, areaName) => {
  const devConfigChanged = areaName === "local" && "devConfig" in changes;
  const managedChanged = areaName === "managed";

  if (devConfigChanged || managedChanged) {
    initializationPromise = null;
    ensureInitialized(true).catch(console.error);
  }
});

chrome.alarms.onAlarm.addListener((alarm) => {
  if (runtimeConfig.mode === "unconfigured") {
    return;
  }

  if (alarm.name === ALARM_HEARTBEAT) {
    ensureInitialized()
      .then(() => heartbeat())
      .catch(console.error);
  }

  if (alarm.name === ALARM_POLICY_SYNC) {
    ensureInitialized()
      .then(() => syncPolicy())
      .catch(console.error);
  }

  if (alarm.name === ALARM_LOG_FLUSH) {
    ensureInitialized()
      .then(() => flushLogs())
      .catch(console.error);
  }
});

chrome.webNavigation.onCommitted.addListener((details) => {
  handleNavigation(details).catch(console.error);
});

chrome.webNavigation.onHistoryStateUpdated.addListener((details) => {
  handleNavigation(details).catch(console.error);
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (!tab.url?.startsWith("http")) {
    return;
  }

  if (changeInfo.status === "complete" || typeof changeInfo.title === "string") {
    ensureInitialized()
      .then(() => scheduleVisitCapture(tabId))
      .catch(console.error);
  }
});

chrome.tabs.onRemoved.addListener((tabId) => {
  const timerId = pendingVisitTimers.get(tabId);
  if (timerId) {
    clearTimeout(timerId);
    pendingVisitTimers.delete(tabId);
  }

  lastLoggedVisitByTab.delete(tabId);
});

ensureInitialized().catch(console.error);

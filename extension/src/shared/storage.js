import { MAX_LOG_QUEUE_SIZE } from "./constants.js";

const PROFILE_KEY = "profileInstanceId";
const TOKEN_KEY = "deviceToken";
const POLICY_KEY = "runtimePolicy";
const LOG_QUEUE_KEY = "logQueue";

export async function getOrCreateProfileInstanceId() {
  const stored = await chrome.storage.local.get([PROFILE_KEY]);
  if (stored[PROFILE_KEY]) {
    return stored[PROFILE_KEY];
  }

  const profileInstanceId = crypto.randomUUID();
  await chrome.storage.local.set({ [PROFILE_KEY]: profileInstanceId });
  return profileInstanceId;
}

export async function getDeviceToken() {
  const stored = await chrome.storage.local.get([TOKEN_KEY]);
  return stored[TOKEN_KEY] || "";
}

export async function setDeviceToken(deviceToken) {
  await chrome.storage.local.set({ [TOKEN_KEY]: deviceToken });
}

export async function clearDeviceToken() {
  await chrome.storage.local.remove([TOKEN_KEY]);
}

export async function getPolicy() {
  const stored = await chrome.storage.local.get([POLICY_KEY]);
  return stored[POLICY_KEY] || { policy_version: 0, default_action: "allow", rules: [] };
}

export async function setPolicy(policy) {
  await chrome.storage.local.set({ [POLICY_KEY]: policy });
}

export async function enqueueLog(event) {
  const stored = await chrome.storage.local.get([LOG_QUEUE_KEY]);
  const queue = stored[LOG_QUEUE_KEY] || [];
  queue.push(event);
  await chrome.storage.local.set({ [LOG_QUEUE_KEY]: queue.slice(-MAX_LOG_QUEUE_SIZE) });
}

export async function peekLogs() {
  const stored = await chrome.storage.local.get([LOG_QUEUE_KEY]);
  return stored[LOG_QUEUE_KEY] || [];
}

export async function removeLogs(count) {
  const stored = await chrome.storage.local.get([LOG_QUEUE_KEY]);
  const queue = stored[LOG_QUEUE_KEY] || [];
  await chrome.storage.local.set({ [LOG_QUEUE_KEY]: queue.slice(count) });
}

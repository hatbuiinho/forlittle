const MANAGED_KEYS = ["machineId", "displayName", "serverBaseUrl", "enrollmentKey"];
const DEV_CONFIG_KEY = "devConfig";

export async function resolveRuntimeConfig() {
  const managed = await chrome.storage.managed.get(MANAGED_KEYS).catch(() => ({}));

  if (managed.machineId && managed.serverBaseUrl) {
    return {
      mode: "managed",
      machineId: managed.machineId,
      displayName: managed.displayName || "",
      serverBaseUrl: managed.serverBaseUrl,
      enrollmentKey: managed.enrollmentKey || ""
    };
  }

  const local = await chrome.storage.local.get([DEV_CONFIG_KEY]);
  const devConfig = local[DEV_CONFIG_KEY] || {};

  if (devConfig.machineId && devConfig.serverBaseUrl) {
    return {
      mode: "dev_local",
      machineId: devConfig.machineId,
      displayName: devConfig.displayName || "",
      serverBaseUrl: devConfig.serverBaseUrl,
      enrollmentKey: devConfig.enrollmentKey || ""
    };
  }

  return { mode: "unconfigured" };
}

export async function getDevConfig() {
  const local = await chrome.storage.local.get([DEV_CONFIG_KEY]);
  return local[DEV_CONFIG_KEY] || {};
}

export async function saveDevConfig(config) {
  await chrome.storage.local.set({
    [DEV_CONFIG_KEY]: config
  });
}

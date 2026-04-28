import { getDeviceToken, setDeviceToken } from "./storage.js";

export class HttpError extends Error {
  constructor(status, message) {
    super(message);
    this.name = "HttpError";
    this.status = status;
  }
}

async function request(baseUrl, path, options = {}) {
  const { skipAuth = false, ...fetchOptions } = options;
  const token = await getDeviceToken();
  const headers = new Headers(options.headers || {});

  headers.set("Content-Type", "application/json");
  if (!skipAuth && token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${baseUrl}${path}`, {
    ...fetchOptions,
    headers
  });

  if (!response.ok) {
    throw new HttpError(response.status, `Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
}

export async function registerAgent(config, profileInstanceId) {
  const payload = {
    machine_id: config.machineId,
    display_name: config.displayName || "",
    profile_instance_id: profileInstanceId,
    extension_version: chrome.runtime.getManifest().version,
    platform: navigator.platform,
    browser: "Chrome",
    browser_version: navigator.userAgent
  };

  const data = await request(config.serverBaseUrl, "/api/v1/agents/register", {
    method: "POST",
    skipAuth: true,
    body: JSON.stringify(payload)
  });

  if (data.device_token) {
    await setDeviceToken(data.device_token);
  }

  return data;
}

export async function fetchPolicy(config) {
  return request(config.serverBaseUrl, "/api/v1/agents/policy");
}

export async function sendHeartbeat(config, profileInstanceId) {
  return request(config.serverBaseUrl, "/api/v1/agents/heartbeat", {
    method: "POST",
    body: JSON.stringify({
      profile_instance_id: profileInstanceId,
      sent_at: new Date().toISOString()
    })
  });
}

export async function sendLogBatch(config, events) {
  return request(config.serverBaseUrl, "/api/v1/agents/logs/batch", {
    method: "POST",
    body: JSON.stringify({ events })
  });
}

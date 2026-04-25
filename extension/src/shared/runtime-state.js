const RUNTIME_STATE_KEY = "runtimeState";

const defaultState = {
  mode: "unconfigured",
  machineId: "",
  serverBaseUrl: "",
  registrationStatus: "idle",
  policyVersion: 0,
  pendingLogs: 0,
  lastError: "",
  lastSyncAt: "",
  lastRegistrationAt: ""
};

export async function getRuntimeState() {
  const stored = await chrome.storage.local.get([RUNTIME_STATE_KEY]);
  return {
    ...defaultState,
    ...(stored[RUNTIME_STATE_KEY] || {})
  };
}

export async function patchRuntimeState(partialState) {
  const current = await getRuntimeState();
  await chrome.storage.local.set({
    [RUNTIME_STATE_KEY]: {
      ...current,
      ...partialState
    }
  });
}

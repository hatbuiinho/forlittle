# For Little Windows Agent

Local enforcement runner for Windows machines. It launches Chrome with a fixed profile and the For Little extension, then keeps Chrome from being bypassed by closing unmanaged Chrome processes.

## MVP Scope

- Launch Chrome with `--user-data-dir` and `--load-extension`.
- Relaunch Chrome when the managed process exits.
- Detect and kill Chrome processes that do not use the managed profile and extension.
- Run as a foreground process for development, or via Windows Scheduled Task for startup/logon.

Native Windows Service mode is intentionally not included in this first cut. A service runs in Session 0 and should not directly launch desktop GUI apps. The correct production shape is service + per-user runner, or a scheduled task running in the active user session.

## Build

```powershell
go build -o forlittle-agent.exe ./cmd/forlittle-agent
```

## Run

```powershell
.\forlittle-agent.exe -config .\config.json
```

Create `config.json` from `config.example.json` and adjust paths.

`extension_path` must point to the unpacked extension folder that directly contains `manifest.json`. Do not point it to Chrome's installed extension store folder.

Install the unpacked extension into `ProgramData` before testing with a standard user:

```powershell
.\scripts\install-extension.ps1 `
  -SourceExtensionPath "D:\For Little\extension"
```

This copies the extension to `C:\ProgramData\ForLittle\Extension` and grants standard users read/execute access. Avoid loading the extension directly from an admin user's project folder because the Chrome process may not be able to read it.

Recommended first test config:

```json
{
  "chrome_path": "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
  "extension_path": "C:\\ProgramData\\ForLittle\\Extension",
  "profile_path": "C:\\ProgramData\\ForLittle\\ChromeUserData",
  "relaunch_delay_seconds": 2,
  "scan_interval_seconds": 3,
  "kill_unmanaged_chrome": true,
  "strict_extension_only": false,
  "force_restart_on_start": true,
  "chrome_log_path": "C:\\ProgramData\\ForLittle\\chrome-debug.log",
  "startup_urls": [
    "chrome://extensions/"
  ],
  "chrome_args": [
    "--profile-directory=Default",
    "--no-first-run",
    "--disable-sync",
    "--disable-background-mode",
    "--disable-features=Translate"
  ]
}
```

Before running the agent, make sure this exists:

```text
C:\ProgramData\ForLittle\Extension\manifest.json
```

Keep `strict_extension_only` as `false` while testing extension loading. After the extension loads reliably, it can be changed to `true` to add `--disable-extensions-except`.

Keep `force_restart_on_start` as `true` while testing. It closes existing Chrome root processes before launching the managed instance, which prevents Chrome from reusing an old process and ignoring `--load-extension`.

When debugging side-load issues, inspect:

```text
C:\ProgramData\ForLittle\chrome-debug.log
```

## Time Control Service

The Time Control Windows Service registers its own machine and Little Monk on its first successful enrollment. Create `C:\ProgramData\ForLittle\TimeControl\config.json` from `config.time-control.example.json`, then set:

- `machine_id`: stable unique identifier for this Windows computer.
- `little_monk_code`: stable identifier for the Little Monk. Reuse the same code on multiple computers for the same Little Monk.
- `little_monk_display_name`: Vietnamese display name shown on the dashboard.
- `enrollment_key`: the server's `DEVICE_ENROLLMENT_KEY`.

The service creates the Little Monk if its code is new, or reuses the existing record when the code already exists. It assigns an unassigned machine automatically. A machine already assigned to a different Little Monk is rejected rather than reassigned; change it deliberately through an administrator workflow if needed.

Credentials created by older Time Control releases are automatically enrolled once after this upgrade, so an existing `pending` machine is also assigned without deleting local state.

Keep `config.json` readable only by Administrators and `SYSTEM`, because the enrollment key can register a device. After initial rollout, rotate the enrollment key if it was shared outside the protected deployment process.

### Build And Deploy Release

On the release machine, run:

```bash
./scripts/build-time-control-release.sh
```

The script detects the host CPU architecture and creates either an `arm64` or `amd64` release. The target Windows architecture must match the release. Override it only when cross-building, for example `FORLITTLE_WINDOWS_ARCH=amd64 ./scripts/build-time-control-release.sh`.

The command creates a timestamped folder under `dist/` containing both executables, the installer, deployment script, and config template. Copy that entire folder to the Windows computer, then double-click `install-time-control.cmd`. On its first run, the script creates `config.json` from the template and opens it in Notepad; fill in the values, save, and double-click it again. It requests Administrator elevation, installs or updates the service, and starts it immediately at boot. Deployment failures stay visible and are written to `deploy-error.log` in the release directory.

Use `deploy-time-control.ps1 -ForceReenroll` only when you intentionally need to discard the local device credential and enroll the computer again.

The installer adds an all-users Start Menu shortcut at `For Little > Lich dung may cua Chu Tieu`. A Standard User can open it to view the read-only policy cache currently applied by the Windows Service. It communicates through the local named pipe and does not grant access to credentials or configuration files.

To remove the service, interactive agent, installed binaries, local credentials, policy cache, and logs before a clean installation, double-click the release's `uninstall-time-control.cmd`. It requests Administrator elevation and retains the machine record on the dashboard.

## Auto Start With Scheduled Task

Run PowerShell as Administrator:

```powershell
.\scripts\install-scheduled-task.ps1 `
  -AgentPath "C:\ProgramData\ForLittle\Agent\forlittle-agent.exe" `
  -ConfigPath "C:\ProgramData\ForLittle\Agent\config.json"
```

This starts the agent at user logon with highest privileges. For stronger enforcement, combine this with Windows AppLocker or Software Restriction Policies to block unmanaged browsers.

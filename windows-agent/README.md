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
go build -o forlittle-agent.exe .\cmd\forlittle-agent
```

## Run

```powershell
.\forlittle-agent.exe -config .\config.json
```

Create `config.json` from `config.example.json` and adjust paths.

## Auto Start With Scheduled Task

Run PowerShell as Administrator:

```powershell
.\scripts\install-scheduled-task.ps1 `
  -AgentPath "C:\ProgramData\ForLittle\Agent\forlittle-agent.exe" `
  -ConfigPath "C:\ProgramData\ForLittle\Agent\config.json"
```

This starts the agent at user logon with highest privileges. For stronger enforcement, combine this with Windows AppLocker or Software Restriction Policies to block unmanaged browsers.

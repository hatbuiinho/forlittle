# For Little Time Control MVP

## Checklist

- [x] Define protocol, state precedence, data ownership, and rollout boundaries.
- [x] Implement server persistence and device/admin APIs.
- [x] Implement dashboard schedule and immediate-control UI.
- [x] Implement the Windows Service policy engine, durable state, and IPC.
- [x] Implement the per-session overlay agent and watchdog.
- [x] Add deployment assets, verification, and an operational rollout checklist.

## Scope

For Little already manages web access through the Chrome extension. Time Control adds
machine access schedules, immediate administrative control, and coarse application
usage reporting. It intentionally does not capture keystrokes, screenshots, webcam,
audio, document contents, or browser page content.

## Authority and State

The Windows Service is the sole authority on a managed computer. The desktop agent
only renders the state supplied over the local named pipe and reports session signals.

```
Dashboard -> Server command store -> Service -> policy engine -> named pipe -> overlay
```

Effective state values are `ALLOWED`, `BLOCKED`, and `EXTENDED`. The calculation uses
the trusted current time, the assigned weekly schedule, and persisted overrides.

Precedence, from highest to lowest:

1. `FORCE_BLOCK` until explicitly cleared by an administrative unblock command.
2. A time-bounded `MANUAL_UNBLOCK`.
3. A time-bounded `EXTRA_TIME` extension.
4. The normal weekly schedule.

Every unblock and extra-time command has an expiry. The server must never issue an
unbounded temporary unlock by default. When offline, the service enforces the last
valid policy and persisted override state; it never unlocks simply because the server
cannot be reached.

## Identity and Credentials

`machines` remains the common physical-computer record used by the browser extension
and Time Control. Credentials are separated by client type. A Windows Service must
not use the extension's `/api/v1/agents/register` flow because that flow rotates the
single existing extension token.

`device_clients` stores one hashed token per `(machine_id, client_type)` where
`client_type` is `extension` or `windows_service`. Enrolment uses a one-time admin
generated code in production; the initial MVP has a server-side enrollment key for
controlled rollout.

## API

Service endpoints use `/api/v1/devices` and a `Bearer` token belonging to
`windows_service`:

- `POST /enroll` - exchange machine identity and the enrollment key for a service token.
- `GET /time-policy` - fetch the assigned policy and server time.
- `GET /commands` - recover unacknowledged commands after reconnect.
- `POST /heartbeat` - publish effective state and software/session health.
- `POST /usage` - send aggregate foreground-app usage buckets.
- `POST /commands/:commandId/ack` - acknowledge `RECEIVED`, `APPLIED`, `FAILED`, or `IGNORED_DUPLICATE`.
- `GET /ws` - receive command notifications. The database command queue remains authoritative.

Admin endpoints are under `/api/v1/admin/time-control`:

- policy read/write by Little Monk;
- machine state and daily usage read;
- `BLOCK`, `UNBLOCK`, `EXTRA_TIME`, `REFRESH_POLICY`, `FORCE_LOCK`, and `FORCE_LOGOUT` creation;
- command and delivery status read.

## Local Service Storage

The service owns `C:\ProgramData\ForLittle\TimeControl\`. It stores the last valid
policy, effective state, command idempotency records, server time metadata, logs, and
SQLite usage aggregation. Standard users receive no write permission to this path.

Windows time is synchronized with the AD domain controller and Standard Users are
denied the Change the system time right through GPO. Server time offset is a secondary
sanity check, not the only clock protection.

## Enforcement Boundaries

The WPF overlay covers all monitors and is recreated if closed. Windows secure desktop
actions such as Ctrl+Alt+Del cannot be intercepted reliably or safely. For a hard
deadline the Service requests a session lock; optional forced logoff is an explicit
admin command because it may discard unsaved work.

GPO deploys the signed MSI, starts the service automatically, launches the UI agent at
user logon, restricts Standard User rights, applies Chrome policy, and can apply
AppLocker or Software Restriction Policies. GPO does not provide dynamic per-app time
quotas; the service provides those later from locally measured foreground-app duration.

## Rollout Checklist

1. Generate a long random `DEVICE_ENROLLMENT_KEY`, set it in the server environment,
   and restart the API. Do not commit the value.
2. Back up PostgreSQL, deploy the API, and verify `GET /healthz` plus authenticated
   dashboard access before enrolling a computer.
3. In the dashboard, create or select the Little Monk, assign one pilot machine, and
   save a restrictive test schedule with a short allowed window.
4. Build the Go service for `windows/amd64`, publish the WPF agent with .NET 8 on a
   Windows build host, and install both on one non-production joined computer.
5. Verify the service: `Get-Service ForLittleTimeControl`, Windows Event Viewer, and
   that `C:\ProgramData\ForLittle\TimeControl\credentials.json` appears only after
   successful enrollment.
6. Verify the named-pipe overlay: issue `BLOCK` in the dashboard, expect the overlay
   within the command polling interval or immediately when WebSocket is connected;
   issue five minutes of `EXTRA_TIME`, then verify it blocks again after expiry.
7. Disconnect the network, change into and out of the schedule, and verify cached
   policy continues to enforce. Reconnect and verify command recovery and heartbeat.
8. Terminate the UI agent as an administrator in the pilot environment and verify the
   LocalSystem Service starts it again. Do not test forced logoff with unsaved work.
9. Apply GPO only after the pilot passes: no local administrator membership for users,
   deny system-time changes, deploy Chrome policy, and restrict unapproved browsers
   through AppLocker or Software Restriction Policies as appropriate.
10. Rotate the enrollment key after rollout. Use a new per-device enrollment mechanism
    before expanding beyond a controlled group of computers.

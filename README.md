# For Little

Phase 1 monorepo scaffold for:

- `server/`: Go + Gin + GORM backend API
- `extension/`: Chrome MV3 extension with managed-ready config and dev fallback
- `dashboard/`: Next.js admin dashboard

## Phase 1 Scope

- register machine by `machine_id`
- track Chrome profile by `profile_instance_id`
- fetch policy rules from server
- log visited sites and send batch logs
- block domains by allowlist/blacklist rules
- review data from dashboard

## Local Development

1. Start PostgreSQL:
   - `docker compose up -d`
2. Prepare backend env:
   - copy `server/.env.example` to `server/.env`
3. Prepare dashboard env:
   - copy `dashboard/.env.example` to `dashboard/.env.local`
4. Install dependencies when network is available:
   - `cd server && go mod tidy`
   - `cd dashboard && npm install`
5. Run backend:
   - one-shot: `cd server && go run ./cmd/api`
   - hot reload with Air: `cd server && air -c .air.toml`
6. Run dashboard:
   - `cd dashboard && npm run dev`
7. Load `extension/` as an unpacked Chrome extension.
   - open popup
   - set dev config like `DEV-LM-001` and `http://localhost:8080`

## Notes

- `chrome.storage.managed` is optional during development.
- The extension falls back to dev-local config until you test on a managed machine.
- Admin auth is intentionally simple in phase 1 and should be tightened in the next pass.

## Managed Extension Releases

The backend can serve a self-hosted Chrome extension update channel for managed installs:

- `GET /extensions/:slug/update.xml`
- `GET /extensions/:slug/:filename`

Recommended flow:

1. Keep a stable private key file outside the repo.
2. Package the extension with that key.
3. Generate `update.xml`.
4. Store the release files under `server/extension-releases/<slug>/`.
5. Force-install through Chrome policy with a stable `update_url`.

Example release command:

```bash
UPDATE_BASE_URL="https://little-be.hatbuinho.me/extensions/forlittle" \
PEM_PATH="/secure/path/forlittle-extension.pem" \
EXTENSION_ID="your_extension_id_here" \
./scripts/package-extension.sh
```

Chrome policy should keep the `update_url` stable:

```json
{
  "your_extension_id_here": {
    "installation_mode": "force_installed",
    "update_url": "https://little-be.hatbuinho.me/extensions/forlittle/update.xml",
    "override_update_url": true
  }
}
```

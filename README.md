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

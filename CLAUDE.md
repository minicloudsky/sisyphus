# Sisyphus — AI / contributor context

This file orients **Claude Code**, **Cursor**, and humans on how this repo is organized and how to work safely in it. Authoritative user docs stay in `README.md`, `docs/GUIDE.md`, and `QUICKSTART.md` (Chinese mirrors: `*.zh-CN.md`).

## What it is

**Sisyphus** is an enterprise-style **database ops platform**: SQL orders (goInception, gh-ost), multi-engine query (MySQL / ClickHouse / Redis / HBase), RBAC, archiving, binlog rollback, Redis analysis, notifications, etc.

## Stack

| Layer | Tech |
|--------|------|
| Backend | **Go 1.21+**, **Kratos v2** (HTTP + gRPC), **GORM**, **Wire** DI |
| Frontend | **Umi 4**, **React**, **Ant Design 5**, **Monaco** editor |
| App DB | MySQL (`migrations/init.sql`) |
| Jobs / queue | Redis + **Asynq** |
| E2E | **Playwright** (`e2e/`) |

## Repo layout

```
server/           # Go: cmd/sisyphus, internal/{biz,data,service,server,...}, api/proto/v1, configs/
web/              # Umi SPA; API baseURL /api → proxied to backend :8000
e2e/              # Playwright tests and doc screenshots
migrations/       # Schema + optional test_data.sql
scripts/          # setup, init_db, local helpers
deploy/k8s/       # Kubernetes manifests
docs/             # GUIDE, api.md, screenshots (en/ + zh-CN/)
```

## Local dev

- **Both**: `./dev.sh` from repo root (backend + frontend).
- **Backend only**: `cd server && go run ./cmd/sisyphus -conf configs/config.local.yaml` (copy from `config.yaml` first). HTTP **:8000**.
- **Frontend**: `cd web && npm install && npm start` — **:8001**, `#/` hash routing.

Default test users (change in production): see `README.md` / `QUICKSTART.md`.

## Backend conventions

- **Layers**: `service` (HTTP/gRPC handlers) → `biz` (use cases) → `data` (GORM, external clients). Keep SQL and side effects in `data`, orchestration in `biz`.
- **Auth**: JWT middleware; some routes are registered manually in `internal/server/http.go` (`registerCustomRoutes`) with `extractBearerToken` + `ValidateToken` — mirror that pattern for new downloads or admin-only GETs.
- **API**: Protobuf in `server/api/proto/v1/*.proto`. After editing proto: `cd server && make api` (requires `protoc` and Kratos codegen tools — see `server/Makefile` `init` target).
- **DI**: `cd server/cmd/sisyphus && wire` after changing constructor sets in `wire.go`.

Verify: `cd server && go build ./...`

## Frontend conventions

- **HTTP**: `web/src/services/api.ts` — Axios instance `baseURL: '/api'`, Bearer from `getToken()`. Kratos list APIs often need flattened query keys (see `kratosPageRequestQuery` and friends in `api.ts`).
- **Success checks**: `web/src/utils/kratosResponse.ts` — `isCommonResponseOk` (`code` 200/0 or `message: success`).
- **i18n**: `web/src/locales/` — `en-US` / `zh-CN` split across `*.ts` and `*.part2.ts`; add keys in both languages when changing UI copy.
- **Routing / layout**: `web/config/routes.ts`, `web/src/layouts/BasicLayout.tsx`.

## Database & config

- Schema: `migrations/init.sql`. Optional seed: `migrations/test_data.sql`.
- Do **not** commit real passwords, JWT secrets, or production DSNs. Use `config.local.yaml` locally (gitignored if your team sets that up).

## Testing & CI

- **Go**: `cd server && go test ./...`
- **E2E**: `cd e2e && npm install && npx playwright test` (services must be up; see `e2e/README` or CI `.github/workflows/e2e.yml`).
- **Screenshots**: `e2e` can regenerate `docs/screenshots/en` and `zh-CN` — see root README.

## Documentation policy

- Default **English** for new README-style / guide content unless the user explicitly wants Chinese-only files.
- Do not churn unrelated markdown or screenshots in the same change as a small code fix unless asked.

## Security mindset

This project touches **production database credentials**, **SQL execution**, and **RBAC**. Favor least privilege, validate inputs, prevent path traversal on any new file download/upload routes, and never log secrets.

## Cursor IDE

Persistent agent rules for this repo live under **`.cursor/rules/`** (`.mdc` files with YAML frontmatter: `description`, `globs` or `alwaysApply`). They complement this file; keep them short and one-topic per file when editing them.

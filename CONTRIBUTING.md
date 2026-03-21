# Contributing to Sisyphus

Default docs are English (`README.md`, `docs/GUIDE.md`, `QUICKSTART.md`). Chinese versions: `README.zh-CN.md`, `docs/GUIDE.zh-CN.md`, `QUICKSTART.zh-CN.md`. See the root README for the **Claude Code** / **Cursor** development attribution.

## Development Setup

1. Clone the repository
2. Follow the [Quick Start guide](QUICKSTART.md) to set up your local environment
3. Run `./dev.sh` to start backend + frontend together

## Project Structure

```
sisyphus/
├── server/         # Go backend (Kratos v2 + gRPC/HTTP)
├── web/            # React frontend (Umi 4 + Ant Design 5)
├── e2e/            # End-to-end tests (Playwright)
├── migrations/     # Database schema & seed SQL
├── scripts/        # Setup and utility scripts
├── deploy/         # Kubernetes manifests
├── docs/           # Documentation
│   ├── GUIDE.md         # User guide (English, default)
│   ├── GUIDE.zh-CN.md   # 使用手册（中文）
│   ├── screenshots/     # en/ + zh-CN/ PNGs for README / guides (Playwright)
│   └── api.md           # REST API notes
├── dev.sh          # One-command local dev startup
├── README.md       # Project overview (English, default)
├── README.zh-CN.md # 项目说明（中文）
├── QUICKSTART.md   # Quick start (English)
└── QUICKSTART.zh-CN.md
```

## Backend Development

```bash
cd server
go build ./...        # Compile
go test ./...         # Run tests
make run              # Start with air (hot reload)
```

## Frontend Development

```bash
cd web
nvm use 18
npm install
npm run dev           # Start dev server on :8001
npm run build         # Production build
```

## E2E Tests

Requires **Node.js ≥ 18** (if your default Node is older, run `nvm use 18` or newer first).

```bash
cd e2e
npm install
npx playwright install chromium
npx playwright test                      # full suite (backend :8000, frontend :8001)
npx playwright test --reporter=html    # HTML report under e2e/playwright-report/ (gitignored)
npm run test:screenshots                 # writes ../docs/screenshots/en/ and zh-CN/ (services must be up)
# or from repo root: ./scripts/capture-doc-screenshots.sh
```

## Code Style

- **Backend**: Follow standard Go conventions, `gofmt`-formatted
- **Frontend**: ESLint + Prettier (run `npm run lint`)

## Submitting Changes

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Ensure E2E tests pass locally
4. Open a pull request with a description of your changes

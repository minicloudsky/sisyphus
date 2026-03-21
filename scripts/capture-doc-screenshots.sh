#!/usr/bin/env bash
# Regenerate README / guide screenshots (English + Chinese UI).
# Requires: backend :8000, frontend :8001, DB seeded (same as E2E).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/e2e"
export UPDATE_DOC_SCREENSHOTS=1
npm run test:screenshots -- --project=chromium

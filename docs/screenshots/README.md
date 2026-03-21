# Documentation screenshots

| Directory   | Used by                          |
|------------|-----------------------------------|
| `en/`      | `README.md`, `docs/GUIDE.md`      |
| `zh-CN/`   | `README.zh-CN.md`, `docs/GUIDE.zh-CN.md` |

Regenerate from the repo root (with backend `:8000`, frontend `:8001`, seeded DB):

```bash
cd e2e
npm install
npx playwright install chromium
npm run test:screenshots
```

This sets `UPDATE_DOC_SCREENSHOTS=1` and runs `tests/08-docs-screenshots.spec.ts`, which drives the UI in each locale via `localStorage` (`umi_locale`) before capture.

Legacy flat `*.png` files in this folder (if any) are obsolete; prefer the locale subdirectories above.

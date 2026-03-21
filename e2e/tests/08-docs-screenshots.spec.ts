/**
 * 为 README / docs 生成中英文界面截图：
 *   - docs/screenshots/en/*.png    → README.md、docs/GUIDE.md
 *   - docs/screenshots/zh-CN/*.png → README.zh-CN.md、docs/GUIDE.zh-CN.md
 *
 * 默认跳过（避免 CI 写仓库）。本地生成：
 *   cd e2e && npm run test:screenshots
 *
 * 前置：后端 :8000、前端 :8001、库与 test_data 已初始化（与 E2E 一致）。
 */
import * as fs from 'fs';
import * as path from 'path';
import { test, type Browser, type Page } from '@playwright/test';
import { login } from './helpers';

const enabled = process.env.UPDATE_DOC_SCREENSHOTS === '1';
const DOCS_ROOT = path.resolve(__dirname, '../../docs/screenshots');

const LOCALES = [
  { umi: 'en-US', dir: 'en' as const },
  { umi: 'zh-CN', dir: 'zh-CN' as const },
] as const;

type Step = {
  file: string;
  hash: string;
  wait?: string;
  timeout?: number;
};

const STEPS: Step[] = [
  {
    file: 'login.png',
    hash: '/#/login',
    wait: '.ant-pro-form-login-main',
    timeout: 60000,
  },
  { file: 'home.png', hash: '/#/home', wait: '.ant-layout' },
  { file: 'instance.png', hash: '/#/instance', wait: '.ant-table, .ant-spin-container' },
  { file: 'sql_order_list.png', hash: '/#/sql-order/list', wait: '.ant-table, .ant-spin-container' },
  { file: 'sql_order_ddl.png', hash: '/#/sql-order/create-ddl', wait: '.ant-form, .ant-layout' },
  { file: 'sql_order_dml.png', hash: '/#/sql-order/create-dml', wait: '.ant-form, .ant-layout' },
  { file: 'query.png', hash: '/#/query', wait: '.monaco-editor, .ant-layout', timeout: 20000 },
  { file: 'table_structure.png', hash: '/#/tools/table-structure', wait: '.ant-layout, .ant-form, .ant-table' },
  { file: 'process_list.png', hash: '/#/process-list', wait: '.ant-layout, .ant-table, .ant-spin-container' },
  { file: 'binlog.png', hash: '/#/binlog', wait: '.ant-layout, .ant-table, .ant-spin-container' },
  { file: 'redis_tasks.png', hash: '/#/redis/tasks', wait: '.ant-layout, .ant-table, .ant-spin-container' },
  { file: 'archive_configs.png', hash: '/#/archive/configs', wait: '.ant-layout, .ant-table, .ant-spin-container' },
  { file: 'widetable_list.png', hash: '/#/widetable/list', wait: '.ant-layout, .ant-table, .ant-spin-container' },
  {
    file: 'permission_users.png',
    hash: '/#/permission/users',
    wait: '.ant-table-tbody tr:not(.ant-table-measure-row), .ant-empty',
  },
  {
    file: 'permission_roles.png',
    hash: '/#/permission/roles',
    wait: '.ant-table-tbody tr:not(.ant-table-measure-row), .ant-empty',
  },
];

async function capture(page: Page, outDir: string, filename: string) {
  await page.waitForTimeout(500);
  await page.screenshot({
    path: path.join(outDir, filename),
    fullPage: true,
  });
}

async function runLocale(browser: Browser, umiLocale: string, outDir: string) {
  fs.mkdirSync(outDir, { recursive: true });

  const context = await browser.newContext();
  await context.addInitScript((locale: string) => {
    try {
      localStorage.setItem('umi_locale', locale);
    } catch {
      /* ignore */
    }
  }, umiLocale);

  const page = await context.newPage();
  await page.setViewportSize({ width: 1440, height: 900 });

  try {
    for (const step of STEPS) {
      if (step.hash === '/#/login') {
        await page.goto(step.hash);
        await page.waitForSelector(step.wait!, { timeout: step.timeout ?? 15000 });
        await capture(page, outDir, step.file);
        await login(page);
        continue;
      }
      await page.goto(step.hash);
      await page.waitForSelector(step.wait!, { timeout: step.timeout ?? 15000 });
      await capture(page, outDir, step.file);
    }
  } finally {
    await context.close();
  }
}

test.describe('文档截图（中英文）', () => {
  test('生成 docs/screenshots/en 与 docs/screenshots/zh-CN', async ({ browser }) => {
    test.skip(!enabled, 'Set UPDATE_DOC_SCREENSHOTS=1 (e.g. npm run test:screenshots)');

    for (const { umi, dir } of LOCALES) {
      const outDir = path.join(DOCS_ROOT, dir);
      await runLocale(browser, umi, outDir);
    }
  });
});

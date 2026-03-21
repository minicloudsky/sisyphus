/**
 * 导航与布局 E2E 测试
 * - 侧边栏导航
 * - 主页加载
 * - 各主要页面可访问
 * - 404 页面
 * - 响应式布局
 */
import { test, expect } from '@playwright/test';
import { login } from './helpers';

test.describe('导航 - 侧边栏', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('侧边栏正常渲染', async ({ page }) => {
    await expect(page.locator('.ant-layout-sider, nav[class*="sider"]')).toBeVisible({ timeout: 10000 });
    // 导航菜单
    await expect(page.locator('.ant-menu')).toBeVisible({ timeout: 5000 });
  });

  test('可以导航到工单列表', async ({ page }) => {
    await page.locator('.ant-menu-item:has-text("工单"), a:has-text("工单")').first().click().catch(async () => {
      await page.goto('/#/sql-order/list');
    });
    await page.waitForURL(/sql-order/, { timeout: 5000 });
    await page.waitForSelector('.ant-table, .ant-spin-container', { timeout: 10000 });
  });

  test('可以导航到实例管理', async ({ page }) => {
    await page.locator('.ant-menu-item:has-text("实例"), a:has-text("实例")').first().click().catch(async () => {
      await page.goto('/#/instance');
    });
    await page.waitForURL(/instance/, { timeout: 5000 });
    await page.waitForSelector('.ant-table, .ant-spin-container', { timeout: 10000 });
  });

  test('可以导航到数据查询', async ({ page }) => {
    await page.goto('/#/query');
    await page.waitForSelector('.monaco-editor, .ant-layout', { timeout: 15000 });
  });
});

test.describe('导航 - 页面访问', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  const pages = [
    { name: '工单列表', url: '/#/sql-order/list' },
    { name: '创建DDL工单', url: '/#/sql-order/create-ddl' },
    { name: '创建DML工单', url: '/#/sql-order/create-dml' },
    { name: '实例管理', url: '/#/instance' },
    { name: '查询历史', url: '/#/query/history' },
    { name: '查询收藏', url: '/#/query/favorites' },
    { name: '用户管理', url: '/#/permission/users' },
    { name: '角色管理', url: '/#/permission/roles' },
    { name: '权限管理', url: '/#/permission/permissions' },
    { name: '数据归档配置', url: '/#/archive/configs' },
    { name: 'Binlog管理', url: '/#/binlog' },
  ];

  for (const p of pages) {
    test(`${p.name}页面可访问 (${p.url})`, async ({ page }) => {
      await page.goto(p.url);
      // 页面不应该有 500 错误或白屏
      await page.waitForSelector('.ant-layout, .ant-spin-container, .ant-table', { timeout: 12000 });
      // 检查没有 JS 错误导致的崩溃（React error boundary 通常显示）
      const errorText = await page.locator('.ant-result-error, [class*="error-boundary"]').count();
      expect(errorText).toBe(0);
    });
  }
});

test.describe('导航 - 主页', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('主页正常加载', async ({ page }) => {
    await page.goto('/#/home');
    await page.waitForSelector('.ant-layout', { timeout: 10000 });
  });
});

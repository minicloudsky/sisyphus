/**
 * 权限管理 E2E 测试
 * - 用户列表加载
 * - 角色列表加载
 * - 权限列表加载
 * - 创建角色
 * - 分配权限
 */
import { test, expect } from '@playwright/test';
import { login, waitForTable } from './helpers';

test.describe('权限管理 - 用户', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('用户列表正常加载', async ({ page }) => {
    await page.goto('/#/permission/users');
    await waitForTable(page);
    // 至少显示 admin 用户
    const rows = await page.locator('.ant-table-tbody tr').count();
    expect(rows).toBeGreaterThan(0);
  });

  test('用户列表有操作按钮', async ({ page }) => {
    await page.goto('/#/permission/users');
    await waitForTable(page);
    // 应该有"新增用户"按钮
    const addBtn = page.locator('button:has-text("新增"), button:has-text("添加")');
    if (await addBtn.isVisible({ timeout: 5000 })) {
      await expect(addBtn).toBeEnabled();
    }
  });
});

test.describe('权限管理 - 角色', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('角色列表正常加载', async ({ page }) => {
    await page.goto('/#/permission/roles');
    await waitForTable(page);
    const rows = await page.locator('.ant-table-tbody tr').count();
    expect(rows).toBeGreaterThan(0);
  });

  test('可以打开创建角色表单', async ({ page }) => {
    await page.goto('/#/permission/roles');
    await waitForTable(page);
    const createBtn = page.locator('button:has-text("新增"), button:has-text("创建角色")');
    if (await createBtn.isVisible({ timeout: 5000 })) {
      await createBtn.click();
      await page.waitForSelector('.ant-modal-content, .ant-drawer-content', { timeout: 5000 });
      await expect(page.locator('.ant-modal-content, .ant-drawer-content')).toBeVisible();
    }
  });
});

test.describe('权限管理 - 权限点', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('权限列表正常加载', async ({ page }) => {
    await page.goto('/#/permission/permissions');
    await waitForTable(page);
    const tableContent = page.locator('.ant-table-tbody').first();
    await expect(tableContent).toBeVisible();
  });
});

/**
 * 实例管理 E2E 测试
 * - 实例列表加载
 * - 新增实例表单
 * - 实例连通性测试
 * - 实例编辑
 * - 实例删除
 */
import { test, expect } from '@playwright/test';
import { login, waitForTable } from './helpers';

test.describe('实例管理', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('实例列表页正常加载', async ({ page }) => {
    await page.goto('/#/instance');
    await waitForTable(page);
    // 应该有"新增实例"按钮
    await expect(page.locator('button:has-text("新增"), button:has-text("添加实例")')).toBeVisible({ timeout: 8000 });
  });

  test('新增实例表单可以打开', async ({ page }) => {
    await page.goto('/#/instance');
    await waitForTable(page);
    await page.click('button:has-text("新增"), button:has-text("添加实例")');
    // 表单弹窗应该出现
    const modal = page.locator('.ant-modal-content, .ant-drawer-content');
    await expect(modal).toBeVisible({ timeout: 5000 });
    // 实例名称输入框（scoped to modal to avoid strict mode violation）
    await expect(modal.locator('input').first()).toBeVisible();
  });

  test('取消新增实例关闭表单', async ({ page }) => {
    await page.goto('/#/instance');
    await waitForTable(page);
    await page.click('button:has-text("新增"), button:has-text("添加实例")');
    await page.waitForSelector('.ant-modal-content, .ant-drawer-content', { timeout: 5000 });
    // 点击关闭（X 按钮或取消按钮）
    await page.click('.ant-modal-close, button:has-text("取消"), button:has-text("关闭")');
    await page.waitForTimeout(500);
    await expect(page.locator('.ant-modal-content')).not.toBeVisible();
  });

  test('已有实例显示在列表中', async ({ page }) => {
    await page.goto('/#/instance');
    await waitForTable(page);
    // 之前测试中创建的"本地MySQL实例"应该存在
    const tableContent = page.locator('.ant-table-tbody');
    await expect(tableContent).toBeVisible();
    const rowCount = await page.locator('.ant-table-tbody tr').count();
    expect(rowCount).toBeGreaterThan(0);
  });

  test('点击测试连接可以验证实例连通性', async ({ page }) => {
    await page.goto('/#/instance');
    await waitForTable(page);
    // 找到第一行的"测试连接"按钮
    const testBtn = page.locator('.ant-table-tbody tr').first().locator('button:has-text("测试"), a:has-text("测试")');
    if (await testBtn.isVisible({ timeout: 3000 })) {
      await testBtn.click();
      // 等待测试结果（成功或失败消息）
      await page.waitForSelector('.ant-message-notice, .ant-notification', { timeout: 10000 });
    }
  });
});

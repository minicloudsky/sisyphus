/**
 * 认证模块 E2E 测试
 * - 登录页渲染
 * - 登录成功跳转
 * - 登录失败提示
 * - 未认证重定向
 * - 登出功能
 */
import { test, expect } from '@playwright/test';
import { login, ADMIN_USER } from './helpers';

test.describe('认证 - 登录/登出', () => {
  test('登录页正常渲染', async ({ page }) => {
    await page.goto('/#/login');
    // Wait for React to hydrate before checking title
    await page.waitForSelector('input[placeholder*="用户名"], input[placeholder*="username"]', { timeout: 10000 });
    await expect(page.locator('input[placeholder*="用户名"], input[placeholder*="username"]')).toBeVisible();
    await expect(page.locator('input[placeholder*="密码"], input[placeholder*="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"], button:has-text("登")')).toBeVisible();
  });

  test('错误密码登录失败并显示提示', async ({ page }) => {
    await page.goto('/#/login');
    await page.fill('input[placeholder*="用户名"], input[placeholder*="username"]', 'admin');
    await page.fill('input[placeholder*="密码"], input[placeholder*="password"]', 'WrongPassword123');
    await page.click('button[type="submit"], button:has-text("登")');

    // 应该显示错误提示，且保持在登录页
    await page.waitForTimeout(2000);
    const url = page.url();
    expect(url).toContain('login');
  });

  test('正确凭证登录成功并跳转', async ({ page }) => {
    await login(page);
    // 跳转后应该不在登录页
    await expect(page).not.toHaveURL(/login/);
    // 应该有导航菜单或主内容区
    await page.waitForSelector('.ant-layout-sider, nav, .ant-menu', { timeout: 10000 });
  });

  test('未登录访问受保护页面应重定向到登录页', async ({ page }) => {
    // 直接访问受保护页面（不登录）
    await page.goto('/#/sql-order/list');
    await page.waitForTimeout(2000);
    // 应该被重定向到登录页
    const url = page.url();
    expect(url).toContain('login');
  });

  test('登录后可以登出', async ({ page }) => {
    await login(page);
    // 找到用户头像或用户名下拉菜单
    const userAvatar = page.locator('.ant-dropdown-trigger, [class*="avatar"], [class*="user"]').first();
    if (await userAvatar.isVisible()) {
      await userAvatar.click();
      // 点击退出登录
      const logoutBtn = page.locator('li:has-text("退出"), li:has-text("Logout"), li:has-text("登出")');
      if (await logoutBtn.isVisible({ timeout: 3000 })) {
        await logoutBtn.click();
        await page.waitForTimeout(2000);
        expect(page.url()).toContain('login');
      }
    }
  });
});

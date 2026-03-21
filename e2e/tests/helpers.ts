import { Page, expect } from '@playwright/test';

export const BASE_URL = 'http://localhost:8001';
export const API_URL = 'http://localhost:8000';
export const ADMIN_USER = { username: 'admin', password: 'Admin@123' };

/** 登录并等待首页加载 */
export async function login(page: Page, username = ADMIN_USER.username, password = ADMIN_USER.password) {
  await page.goto('/#/login', { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('.ant-pro-form-login-main', { timeout: 60000 });
  // Prefer placeholder (visible fields); ProForm may render hidden measure inputs first
  await page.getByPlaceholder(/用户名|Username/i).fill(username);
  await page.getByPlaceholder(/密码|Password/i).fill(password);
  // Primary submit works for zh-CN / en-US Pro LoginForm
  await page.locator('.ant-pro-form-login-main button.ant-btn-primary').first().click();
  // 等待跳转到非登录页
  await page.waitForURL((url) => !url.hash.includes('/login'), { timeout: 10000 });
}

/** 通过 API 获取 token */
export async function getToken(): Promise<string> {
  const resp = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ADMIN_USER),
  });
  const data = await resp.json();
  return data.data?.token ?? '';
}

/** 等待 antd Table 加载完成 */
export async function waitForTable(page: Page) {
  // Exclude the ant-table-measure-row which is aria-hidden and never becomes visible
  await page.waitForSelector('.ant-table-tbody tr:not(.ant-table-measure-row), .ant-empty', { timeout: 10000 });
}

/** 等待 antd message 弹出 */
export async function waitForMessage(page: Page, text?: string) {
  const selector = text
    ? `.ant-message-notice-content:has-text("${text}")`
    : '.ant-message-notice-content';
  await page.waitForSelector(selector, { timeout: 8000 });
}

/** 关闭所有 antd 弹窗 */
export async function closeModals(page: Page) {
  const closeButtons = page.locator('.ant-modal-close');
  const count = await closeButtons.count();
  for (let i = 0; i < count; i++) {
    await closeButtons.first().click().catch(() => {});
  }
}

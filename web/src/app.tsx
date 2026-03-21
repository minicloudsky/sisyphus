import type { ReactNode } from 'react';

/**
 * rootContainer 在 Umi locale 的 IntlProvider 之外执行，此处不能使用 useIntl。
 * <html lang> 由 DocumentLang 挂在各路由组件内（BasicLayout / Login / Register / 404）。
 */
export function rootContainer(container: ReactNode) {
  return container;
}

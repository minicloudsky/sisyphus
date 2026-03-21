/**
 * Umi `@umijs/plugins/dist/locale` 官方 API 统一出口（需在 `.umirc.ts` 中注册插件）。
 * 业务代码优先从此文件引用，便于跳转定义与后续替换实现。
 */
export {
  useIntl,
  setLocale,
  getLocale,
  getAllLocales,
  getIntl,
  formatMessage,
  FormattedMessage,
  FormattedDate,
  FormattedNumber,
  FormattedTime,
  SelectLang,
  injectIntl,
} from 'umi';

import { defineConfig } from '@umijs/max';
import routes from './config/routes';

const publicPath = process.env.NODE_ENV === 'production' ? '/sisyphus/' : '/';
const faviconIcoHref =
  publicPath === '/' ? '/favicon.ico' : `${publicPath.replace(/\/$/, '')}/favicon.ico`;

export default defineConfig({
  plugins: [require.resolve('@umijs/plugins/dist/locale')],
  locale: {
    default: 'zh-CN',
    antd: true,
    baseNavigator: false,
    useLocalStorage: true,
    title: false,
  },
  routes,
  proxy: {
    '/api': {
      target: 'http://localhost:8000',
      changeOrigin: true,
      pathRewrite: { '^/api': '/api' },
    },
  },
  hash: true,
  history: {
    type: 'hash',
  },
  publicPath,
  /** 与 src/favicon.svg 并存；开发环境下 /favicon.ico 会命中 public，避免 Umi 默认 umi.png */
  links: [{ rel: 'icon', type: 'image/x-icon', href: faviconIcoHref }],
  outputPath: 'dist',
  mfsu: false,
});

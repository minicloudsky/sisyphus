import { defineConfig } from '@umijs/max';
import routes from './routes';

export default defineConfig({
  title: 'Sisyphus',
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
  publicPath: process.env.NODE_ENV === 'production' ? '/sisyphus/' : '/',
  outputPath: 'dist',
  mfsu: false,
});

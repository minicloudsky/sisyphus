# Sisyphus 前端项目

基于 Umi + Ant Design + ahooks 的数据库智能平台前端。

## 技术栈

- **框架**: Umi 4.x
- **UI组件**: Ant Design 5.x
- **Hooks库**: ahooks
- **状态管理**: Zustand
- **代码编辑器**: Monaco Editor
- **构建工具**: Umi内置

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 代码格式化
npm run format

# 代码检查
npm run lint
```

## 项目结构

```
src/
├── pages/          # 页面组件
│   ├── Login/     # 登录页
│   ├── Register/  # 注册页
│   ├── Home/      # 首页
│   ├── SQLOrder/  # SQL工单
│   ├── Query/     # 数据查询
│   ├── Binlog/    # Binlog回滚
│   ├── Redis/     # Redis分析
│   ├── Archive/   # 数据归档
│   ├── Instance/  # 实例管理
│   ├── Permission/# 权限管理
│   └── Tools/     # 工具
├── services/      # API服务
├── components/    # 公共组件
├── utils/         # 工具函数
├── locales/       # 国际化：zh-CN.ts、en-US.ts、I18nContext.tsx
├── typings.d.ts   # TypeScript类型定义
└── app.tsx        # 根容器：I18nProvider + Ant Design 语言包
```

## 国际化（中英文）

- 默认 **中文**；语言保存在 `localStorage` 键 `sisyphus-locale`（`zh-CN` / `en-US`）。
- 顶栏与登录/注册页提供语言切换；Ant Design 组件随语言切换内置文案（表单校验等）。
- 在页面中使用：`import { useI18n } from '@/locales/I18nContext'`，然后 `const { t, locale, setLocale } = useI18n()`，`t('menu.home')` 等。
- 新增文案：在 `src/locales/zh-CN.ts` 与 `src/locales/en-US.ts` 中为同一 key 各写一条翻译。

## 环境变量

创建 `.env` 文件：

```env
UMI_ENV=dev
API_BASE_URL=http://localhost:8000
```

## 部署

### Docker构建

```bash
docker build -t sisyphus-frontend:latest .
```

### K8s部署

参考 `deploy/k8s/web.yaml`

import React, { useState, useEffect, useMemo } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Avatar, Dropdown, message, Select } from 'antd';
import {
  HomeOutlined,
  FileTextOutlined,
  SearchOutlined,
  HistoryOutlined,
  ApiOutlined,
  DatabaseOutlined,
  SafetyOutlined,
  LogoutOutlined,
  SettingOutlined,
  InboxOutlined,
  CloudServerOutlined,
  TableOutlined,
  ToolOutlined,
  UserOutlined,
  TeamOutlined,
  BellOutlined,
  GlobalOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useIntl, setLocale } from '@/locales';
import DocumentLang from '@/locales/DocumentLang';
import { removeToken, removeUser, getUser } from '@/utils/auth';
import './BasicLayout.less';

const { Header, Sider, Content } = Layout;

const PAGE_TITLE_IDS: Record<string, string> = {
  '/home': 'pageTitle.home',
  '/sql-order/list': 'pageTitle.sqlOrderList',
  '/sql-order/create-ddl': 'pageTitle.sqlOrderCreateDdl',
  '/sql-order/create-dml': 'pageTitle.sqlOrderCreateDml',
  '/sql-order/detail': 'pageTitle.sqlOrderDetail',
  '/query': 'pageTitle.query',
  '/binlog': 'pageTitle.binlog',
  '/redis/hotkeys': 'pageTitle.redisHotkeys',
  '/redis/bigkeys': 'pageTitle.redisBigkeys',
  '/redis/tasks': 'pageTitle.redisTasks',
  '/archive/configs': 'pageTitle.archiveConfigs',
  '/archive/records': 'pageTitle.archiveRecords',
  '/widetable/list': 'pageTitle.widetableList',
  '/widetable/create': 'pageTitle.widetableCreate',
  '/widetable/detail': 'pageTitle.widetableDetail',
  '/instance': 'pageTitle.instance',
  '/permission/users': 'pageTitle.permissionUsers',
  '/permission/roles': 'pageTitle.permissionRoles',
  '/permission/permissions': 'pageTitle.permissionPermissions',
  '/process-list': 'pageTitle.processList',
  '/profile': 'pageTitle.profile',
  '/settings/notification': 'pageTitle.notification',
  '/query/history': 'pageTitle.queryHistory',
  '/query/favorites': 'pageTitle.queryFavorites',
  '/query/hbase': 'pageTitle.queryHBase',
  '/tools/mybatis': 'pageTitle.toolsMybatis',
  '/tools/table-structure': 'pageTitle.tableStructure',
};

function getPageTitleKey(pathname: string): string | null {
  for (const [prefix, id] of Object.entries(PAGE_TITLE_IDS)) {
    if (pathname === prefix || pathname.startsWith(prefix + '/')) {
      return id;
    }
  }
  return null;
}

const SISYPHUS_SVG = encodeURIComponent(
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">` +
    `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">` +
    `<stop offset="0%" stop-color="#4096ff"/><stop offset="100%" stop-color="#1677ff"/>` +
    `</linearGradient></defs>` +
    `<circle cx="50" cy="50" r="50" fill="url(#g)"/>` +
    `<circle cx="30" cy="27" r="9" fill="white"/>` +
    `<line x1="30" y1="36" x2="44" y2="56" stroke="white" stroke-width="4" stroke-linecap="round"/>` +
    `<line x1="39" y1="47" x2="62" y2="42" stroke="white" stroke-width="3" stroke-linecap="round"/>` +
    `<line x1="44" y1="56" x2="28" y2="74" stroke="white" stroke-width="4" stroke-linecap="round"/>` +
    `<line x1="44" y1="56" x2="54" y2="73" stroke="white" stroke-width="4" stroke-linecap="round"/>` +
    `<circle cx="72" cy="38" r="17" fill="rgba(255,255,255,0.18)" stroke="white" stroke-width="2.5"/>` +
    `<circle cx="70" cy="36" r="5" fill="rgba(255,255,255,0.4)"/>` +
    `</svg>`
);
const SISYPHUS_AVATAR_SRC = `data:image/svg+xml;charset=utf-8,${SISYPHUS_SVG}`;

const BasicLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const currentUser = getUser();
  const intl = useIntl();

  useEffect(() => {
    const id = getPageTitleKey(location.pathname);
    const appName = intl.formatMessage({ id: 'app.name' });
    if (!id) {
      document.title = appName;
      return;
    }
    const title = intl.formatMessage({ id });
    document.title = `${title} — ${appName}`;
  }, [location.pathname, intl.locale, intl]);

  const menuItems: MenuProps['items'] = useMemo(
    () => [
      {
        key: '/home',
        icon: <HomeOutlined />,
        label: intl.formatMessage({ id: 'menu.home' }),
      },
      { type: 'divider' },
      {
        key: '/query',
        icon: <SearchOutlined />,
        label: intl.formatMessage({ id: 'menu.query' }),
      },
      {
        key: '/sql-order',
        icon: <FileTextOutlined />,
        label: intl.formatMessage({ id: 'menu.sqlOrder' }),
        children: [
          {
            key: '/sql-order/list',
            icon: <FileTextOutlined />,
            label: intl.formatMessage({ id: 'menu.sqlOrder.list' }),
          },
          {
            key: '/sql-order/create-ddl',
            icon: <ToolOutlined />,
            label: intl.formatMessage({ id: 'menu.sqlOrder.ddl' }),
          },
          {
            key: '/sql-order/create-dml',
            icon: <TableOutlined />,
            label: intl.formatMessage({ id: 'menu.sqlOrder.dml' }),
          },
        ],
      },
      { type: 'divider' },
      {
        key: '/binlog',
        icon: <HistoryOutlined />,
        label: intl.formatMessage({ id: 'menu.binlog' }),
      },
      {
        key: '/redis',
        icon: <ApiOutlined />,
        label: intl.formatMessage({ id: 'menu.redis' }),
        children: [
          {
            key: '/redis/hotkeys',
            icon: <ApiOutlined />,
            label: intl.formatMessage({ id: 'menu.redis.hotkeys' }),
          },
          {
            key: '/redis/bigkeys',
            icon: <DatabaseOutlined />,
            label: intl.formatMessage({ id: 'menu.redis.bigkeys' }),
          },
          {
            key: '/redis/tasks',
            icon: <HistoryOutlined />,
            label: intl.formatMessage({ id: 'menu.redis.tasks' }),
          },
        ],
      },
      {
        key: '/archive',
        icon: <InboxOutlined />,
        label: intl.formatMessage({ id: 'menu.archive' }),
        children: [
          {
            key: '/archive/configs',
            icon: <SettingOutlined />,
            label: intl.formatMessage({ id: 'menu.archive.configs' }),
          },
          {
            key: '/archive/records',
            icon: <HistoryOutlined />,
            label: intl.formatMessage({ id: 'menu.archive.records' }),
          },
        ],
      },
      {
        key: '/widetable/list',
        icon: <TableOutlined />,
        label: intl.formatMessage({ id: 'menu.widetable' }),
      },
      {
        key: '/process-list',
        icon: <DatabaseOutlined />,
        label: intl.formatMessage({ id: 'menu.processList' }),
      },
      { type: 'divider' },
      {
        key: '/instance',
        icon: <CloudServerOutlined />,
        label: intl.formatMessage({ id: 'menu.instance' }),
      },
      {
        key: '/permission',
        icon: <SafetyOutlined />,
        label: intl.formatMessage({ id: 'menu.permission' }),
        children: [
          {
            key: '/permission/users',
            icon: <TeamOutlined />,
            label: intl.formatMessage({ id: 'menu.permission.users' }),
          },
          {
            key: '/permission/roles',
            icon: <UserOutlined />,
            label: intl.formatMessage({ id: 'menu.permission.roles' }),
          },
          {
            key: '/permission/permissions',
            icon: <ToolOutlined />,
            label: intl.formatMessage({ id: 'menu.permission.permissions' }),
          },
        ],
      },
      { type: 'divider' },
      {
        key: '/settings',
        icon: <SettingOutlined />,
        label: intl.formatMessage({ id: 'menu.settings' }),
        children: [
          {
            key: '/settings/notification',
            icon: <BellOutlined />,
            label: intl.formatMessage({ id: 'menu.settings.notification' }),
          },
        ],
      },
    ],
    [intl.locale, intl]
  );

  const userMenuItems: MenuProps['items'] = useMemo(
    () => [
      {
        key: 'profile',
        icon: <UserOutlined />,
        label: intl.formatMessage({ id: 'user.profile' }),
      },
      { type: 'divider' },
      {
        key: 'logout',
        icon: <LogoutOutlined />,
        label: intl.formatMessage({ id: 'user.logout' }),
        danger: true,
      },
    ],
    [intl.locale, intl]
  );

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    navigate(key);
  };

  const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') {
      removeToken();
      removeUser();
      message.success(intl.formatMessage({ id: 'user.logoutSuccess' }));
      navigate('/login');
    } else if (key === 'profile') {
      navigate('/profile');
    }
  };

  const getSelectedKeys = () => {
    const path = location.pathname;
    if (path.startsWith('/sql-order/detail/')) return ['/sql-order/list'];
    if (path.startsWith('/widetable')) return ['/widetable/list'];
    return [path];
  };

  const getOpenKeys = () => {
    const path = location.pathname;
    const keys: string[] = [];
    if (path.startsWith('/sql-order')) keys.push('/sql-order');
    if (path.startsWith('/redis')) keys.push('/redis');
    if (path.startsWith('/archive')) keys.push('/archive');
    if (path.startsWith('/permission')) keys.push('/permission');
    if (path.startsWith('/settings')) keys.push('/settings');
    return keys;
  };

  return (
    <>
      <DocumentLang />
      <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        theme="dark"
        width={220}
      >
        <div className="logo">
          <h1>{intl.formatMessage({ id: 'app.name' })}</h1>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={getSelectedKeys()}
          defaultOpenKeys={getOpenKeys()}
          items={menuItems}
          onClick={handleMenuClick}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: '#fff',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            boxShadow: '0 1px 4px rgba(0,21,41,.08)',
          }}
        >
          <div className="header-title" style={{ color: '#262626', fontWeight: 500 }}>
            {intl.formatMessage({ id: 'app.name' })} &nbsp;·&nbsp;
            <span style={{ color: '#8c8c8c', fontWeight: 400, fontSize: '13px' }}>
              {intl.formatMessage({ id: 'app.subtitle' })}
            </span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <span style={{ color: '#8c8c8c', fontSize: 13, display: 'flex', alignItems: 'center', gap: 6 }}>
              <GlobalOutlined />
              {intl.formatMessage({ id: 'lang.switch' })}
            </span>
            <Select
              size="small"
              value={intl.locale}
              style={{ width: 120 }}
              options={[
                { value: 'zh-CN', label: intl.formatMessage({ id: 'lang.zh' }) },
                { value: 'en-US', label: intl.formatMessage({ id: 'lang.en' }) },
              ]}
              onChange={(v) => setLocale(v as string, false)}
            />
            <Dropdown
              menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
              placement="bottomRight"
            >
              <div
                style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '8px' }}
              >
                <Avatar src={SISYPHUS_AVATAR_SRC} size={32} />
                <span style={{ fontSize: '14px', color: '#262626' }}>
                  {currentUser?.username || intl.formatMessage({ id: 'app.user.fallback' })}
                </span>
              </div>
            </Dropdown>
          </div>
        </Header>
        <Content
          style={{
            margin: '20px 12px',
            padding: '20px 18px',
            background: '#fff',
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
    </>
  );
};

export default BasicLayout;

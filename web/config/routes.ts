export default [
  {
    path: '/login',
    component: './Login',
  },
  {
    path: '/register',
    component: './Register',
  },
  {
    path: '/',
    component: '@/layouts/BasicLayout',
    routes: [
      {
        path: '/',
        redirect: '/home',
      },
      {
        path: '/home',
        component: './Home',
      },
      {
        path: '/sql-order/list',
        component: './SQLOrder/List',
      },
      {
        path: '/sql-order/create-ddl',
        component: './SQLOrder/CreateDDL',
      },
      {
        path: '/sql-order/create-dml',
        component: './SQLOrder/CreateDML',
      },
      {
        path: '/sql-order/detail/:id',
        component: './SQLOrder/Detail',
      },
      {
        path: '/query',
        component: './Query',
      },
      {
        path: '/query/history',
        component: './Query/History',
      },
      {
        path: '/query/favorites',
        component: './Query/Favorites',
      },
      {
        path: '/query/hbase',
        component: './Query/HBase',
      },
      {
        path: '/binlog',
        component: './Binlog/List',
      },
      {
        path: '/redis/hotkeys',
        component: './Redis/HotKeys',
      },
      {
        path: '/redis/bigkeys',
        component: './Redis/BigKeys',
      },
      {
        path: '/redis/tasks',
        component: './Redis/Tasks',
      },
      {
        path: '/archive/configs',
        component: './Archive/Configs',
      },
      {
        path: '/archive/records',
        component: './Archive/Records',
      },
      {
        path: '/widetable/list',
        component: './Widetable/List',
      },
      {
        path: '/widetable/create',
        component: './Widetable/Create',
      },
      {
        path: '/widetable/detail/:id',
        component: './Widetable/Detail',
      },
      {
        path: '/instance',
        component: './Instance/List',
      },
      {
        path: '/permission/users',
        component: './Permission/Users',
      },
      {
        path: '/permission/roles',
        component: './Permission/Roles',
      },
      {
        path: '/permission/permissions',
        component: './Permission/Permissions',
      },
      {
        path: '/tools/mybatis',
        component: './Tools/MyBatis',
      },
      {
        path: '/tools/table-structure',
        component: './TableStructure',
      },
      {
        path: '/process-list',
        component: './ProcessList',
      },
      {
        path: '/profile',
        component: './Profile',
      },
      {
        path: '/settings/notification',
        component: './Settings/Notification',
      },
    ],
  },
  {
    path: '/404',
    component: './404',
  },
];

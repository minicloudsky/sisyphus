/** 权限目录数据：module / 文案 key 与后端 code 对应，界面用 intl 渲染。 */

export type PermModuleKey =
  | 'dataQuery'
  | 'sqlOrder'
  | 'binlog'
  | 'redisAnalysis'
  | 'archive'
  | 'widetable'
  | 'process'
  | 'instance'
  | 'userAdmin'
  | 'rbac';

export interface PermDef {
  module: PermModuleKey;
  nameKey: string;
  descKey: string;
  code: string;
  resource: string;
  action: string;
}

export const PERM_MODULE_ORDER: PermModuleKey[] = [
  'dataQuery',
  'sqlOrder',
  'binlog',
  'redisAnalysis',
  'archive',
  'widetable',
  'process',
  'instance',
  'userAdmin',
  'rbac',
];

export const MODULE_COLOR: Record<PermModuleKey, string> = {
  dataQuery: 'blue',
  sqlOrder: 'geekblue',
  binlog: 'purple',
  redisAnalysis: 'red',
  archive: 'orange',
  widetable: 'cyan',
  process: 'lime',
  instance: 'green',
  userAdmin: 'gold',
  rbac: 'magenta',
};

export const SYSTEM_PERMISSIONS: PermDef[] = [
  {
    module: 'dataQuery',
    nameKey: 'perm.query.execute',
    descKey: 'perm.query.execute.desc',
    code: 'query:execute',
    resource: '/api/v1/query',
    action: 'execute',
  },
  {
    module: 'dataQuery',
    nameKey: 'perm.query.export',
    descKey: 'perm.query.export.desc',
    code: 'query:export',
    resource: '/api/v1/query/export',
    action: 'export',
  },
  {
    module: 'dataQuery',
    nameKey: 'perm.query.history',
    descKey: 'perm.query.history.desc',
    code: 'query:history',
    resource: '/api/v1/query/history',
    action: 'view',
  },
  {
    module: 'sqlOrder',
    nameKey: 'perm.sql_order.create',
    descKey: 'perm.sql_order.create.desc',
    code: 'sql_order:create',
    resource: '/api/v1/sql-orders',
    action: 'create',
  },
  {
    module: 'sqlOrder',
    nameKey: 'perm.sql_order.view',
    descKey: 'perm.sql_order.view.desc',
    code: 'sql_order:view',
    resource: '/api/v1/sql-orders',
    action: 'view',
  },
  {
    module: 'sqlOrder',
    nameKey: 'perm.sql_order.approve',
    descKey: 'perm.sql_order.approve.desc',
    code: 'sql_order:approve',
    resource: '/api/v1/sql-orders/:id/approve',
    action: 'approve',
  },
  {
    module: 'sqlOrder',
    nameKey: 'perm.sql_order.execute',
    descKey: 'perm.sql_order.execute.desc',
    code: 'sql_order:execute',
    resource: '/api/v1/sql-orders/:id/execute',
    action: 'execute',
  },
  {
    module: 'binlog',
    nameKey: 'perm.binlog.view',
    descKey: 'perm.binlog.view.desc',
    code: 'binlog:view',
    resource: '/api/v1/binlog',
    action: 'view',
  },
  {
    module: 'binlog',
    nameKey: 'perm.binlog.execute',
    descKey: 'perm.binlog.execute.desc',
    code: 'binlog:execute',
    resource: '/api/v1/binlog/:id/execute',
    action: 'execute',
  },
  {
    module: 'redisAnalysis',
    nameKey: 'perm.redis.hotkey',
    descKey: 'perm.redis.hotkey.desc',
    code: 'redis:hotkey',
    resource: '/api/v1/redis/hotkeys',
    action: 'execute',
  },
  {
    module: 'redisAnalysis',
    nameKey: 'perm.redis.bigkey',
    descKey: 'perm.redis.bigkey.desc',
    code: 'redis:bigkey',
    resource: '/api/v1/redis/bigkeys',
    action: 'execute',
  },
  {
    module: 'archive',
    nameKey: 'perm.archive.config',
    descKey: 'perm.archive.config.desc',
    code: 'archive:config',
    resource: '/api/v1/archive/configs',
    action: 'manage',
  },
  {
    module: 'archive',
    nameKey: 'perm.archive.view',
    descKey: 'perm.archive.view.desc',
    code: 'archive:view',
    resource: '/api/v1/archive/records',
    action: 'view',
  },
  {
    module: 'archive',
    nameKey: 'perm.archive.execute',
    descKey: 'perm.archive.execute.desc',
    code: 'archive:execute',
    resource: '/api/v1/archive/:id/execute',
    action: 'execute',
  },
  {
    module: 'widetable',
    nameKey: 'perm.widetable.manage',
    descKey: 'perm.widetable.manage.desc',
    code: 'widetable:manage',
    resource: '/api/v1/widetable',
    action: 'manage',
  },
  {
    module: 'process',
    nameKey: 'perm.processlist.view',
    descKey: 'perm.processlist.view.desc',
    code: 'processlist:view',
    resource: '/api/v1/process-list',
    action: 'view',
  },
  {
    module: 'process',
    nameKey: 'perm.processlist.kill',
    descKey: 'perm.processlist.kill.desc',
    code: 'processlist:kill',
    resource: '/api/v1/process-list/:id/kill',
    action: 'execute',
  },
  {
    module: 'instance',
    nameKey: 'perm.instance.view',
    descKey: 'perm.instance.view.desc',
    code: 'instance:view',
    resource: '/api/v1/instances',
    action: 'view',
  },
  {
    module: 'instance',
    nameKey: 'perm.instance.create',
    descKey: 'perm.instance.create.desc',
    code: 'instance:create',
    resource: '/api/v1/instances',
    action: 'create',
  },
  {
    module: 'instance',
    nameKey: 'perm.instance.update',
    descKey: 'perm.instance.update.desc',
    code: 'instance:update',
    resource: '/api/v1/instances/:id',
    action: 'update',
  },
  {
    module: 'instance',
    nameKey: 'perm.instance.delete',
    descKey: 'perm.instance.delete.desc',
    code: 'instance:delete',
    resource: '/api/v1/instances/:id',
    action: 'delete',
  },
  {
    module: 'userAdmin',
    nameKey: 'perm.user.view',
    descKey: 'perm.user.view.desc',
    code: 'user:view',
    resource: '/api/v1/permission/users',
    action: 'view',
  },
  {
    module: 'userAdmin',
    nameKey: 'perm.user.create',
    descKey: 'perm.user.create.desc',
    code: 'user:create',
    resource: '/api/v1/permission/users',
    action: 'create',
  },
  {
    module: 'userAdmin',
    nameKey: 'perm.user.update',
    descKey: 'perm.user.update.desc',
    code: 'user:update',
    resource: '/api/v1/permission/users/:id',
    action: 'update',
  },
  {
    module: 'userAdmin',
    nameKey: 'perm.user.delete',
    descKey: 'perm.user.delete.desc',
    code: 'user:delete',
    resource: '/api/v1/permission/users/:id',
    action: 'delete',
  },
  {
    module: 'rbac',
    nameKey: 'perm.role.manage',
    descKey: 'perm.role.manage.desc',
    code: 'role:manage',
    resource: '/api/v1/roles',
    action: 'manage',
  },
  {
    module: 'rbac',
    nameKey: 'perm.permission.manage',
    descKey: 'perm.permission.manage.desc',
    code: 'permission:manage',
    resource: '/api/v1/permissions',
    action: 'manage',
  },
];

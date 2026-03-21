import axios from 'axios';
import { message } from 'antd';
import { getToken, removeToken } from '@/utils/auth';

// 创建axios实例
const apiClient = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    const token = getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    message.error('请求发送失败');
    return Promise.reject(error);
  }
);

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    const { response } = error;
    if (response) {
      const { status, data } = response;
      if (status === 401) {
        message.error('未登录或登录已过期，请重新登录');
        removeToken();
        window.location.href = '/#/login';
      } else if (status === 403) {
        message.error('权限不足');
      } else if (status >= 500) {
        message.error('服务器错误，请稍后重试');
      } else {
        message.error(data?.message || '请求失败');
      }
    } else {
      message.error('网络错误，请检查网络连接');
    }
    return Promise.reject(error);
  }
);

// 封装request函数
const request = async <T = any>(url: string, options?: any): Promise<T> => {
  const { method = 'GET', data, params, ...rest } = options || {};

  // 对于没有data的POST请求，不设置Content-Type
  const config: any = {
    url,
    method,
    params,
    ...rest,
  };

  if (data !== undefined) {
    config.data = data;
  } else if (method === 'POST' || method === 'PUT' || method === 'PATCH') {
    // 对于POST/PUT/PATCH请求，如果没有data，发送空对象
    config.data = {};
  }

  const response = await apiClient.request(config);
  return response as T;
};

// ── Kratos HTTP GET：proto 里分页在 PageRequest page_request 下，需发 page_request.page / page_request.page_size
function kratosPageRequestQuery(params: Record<string, any>) {
  const page = Number(params.page ?? params.page_request?.page ?? 1) || 1;
  const pageSize =
    Number(
      params.pageSize ??
        params.page_request?.page_size ??
        params.page_request?.pageSize ??
        20,
    ) || 20;
  return {
    'page_request.page': page,
    'page_request.page_size': pageSize,
  };
}

function omitPaginationFields(params: Record<string, any>) {
  const { page, pageSize, page_request, ...rest } = params;
  return rest;
}

/** 前端常用驼峰 → proto query 蛇形（与 optional 字段对齐） */
function mapListQuerySnakeCase(rest: Record<string, any>): Record<string, any> {
  const q: Record<string, any> = { ...rest };
  if (q.instanceId !== undefined && q.instance_id === undefined) {
    q.instance_id = q.instanceId;
    delete q.instanceId;
  }
  if (q.dbType !== undefined && q.db_type === undefined) {
    q.db_type = q.dbType;
    delete q.dbType;
  }
  if (q.taskType !== undefined && q.task_type === undefined) {
    q.task_type = q.taskType;
    delete q.taskType;
  }
  if (q.instancePermissionScope !== undefined && q.instance_permission_scope === undefined) {
    q.instance_permission_scope = q.instancePermissionScope;
    delete q.instancePermissionScope;
  }
  return q;
}

function buildKratosPagedListQuery(params: Record<string, any>) {
  return {
    ...mapListQuerySnakeCase(omitPaginationFields(params)),
    ...kratosPageRequestQuery(params),
  };
}

// 认证相关API
export async function login(data: { username: string; password: string }) {
  return request<API.LoginResponse>('/v1/auth/login', {
    method: 'POST',
    data,
  });
}

export async function register(data: API.RegisterRequest) {
  return request<API.RegisterResponse>('/v1/auth/register', {
    method: 'POST',
    data,
  });
}

// SQL工单相关API
export async function createSQLOrder(data: API.CreateOrderRequest) {
  return request<API.CreateOrderResponse>('/v1/sql-orders', {
    method: 'POST',
    data,
  });
}

export async function listSQLOrders(params: API.ListOrdersRequest) {
  return request<API.ListOrdersResponse>('/v1/sql-orders', {
    method: 'GET',
    params,
  });
}

export async function getSQLOrder(id: string) {
  return request<API.GetOrderResponse>(`/v1/sql-orders/${id}`, {
    method: 'GET',
  });
}

export async function approveSQLOrder(id: string, data: API.ApproveOrderRequest) {
  return request<API.CommonResponse>(`/v1/sql-orders/${id}/approve`, {
    method: 'PUT',
    data,
  });
}

export async function rejectSQLOrder(id: string, data: API.RejectOrderRequest) {
  return request<API.CommonResponse>(`/v1/sql-orders/${id}/reject`, {
    method: 'POST',
    data,
  });
}

export async function executeSQLOrder(id: string) {
  return request<API.CommonResponse>(`/v1/sql-orders/${id}/execute`, {
    method: 'POST',
  });
}

export async function getOrderItems(id: string) {
  return request<API.GetOrderItemsResponse>(`/v1/sql-orders/${id}/items`, {
    method: 'GET',
  });
}

export async function getOrderRecords(id: string) {
  return request<API.GetOrderRecordsResponse>(`/v1/sql-orders/${id}/records`, {
    method: 'GET',
  });
}

export async function auditSQL(data: API.AuditSQLRequest) {
  return request<API.AuditSQLResponse>('/v1/sql-orders/audit', {
    method: 'POST',
    data,
  });
}

// 直接审核SQL（不需要工单ID）
export async function reviewSQL(data: API.ReviewSQLRequest) {
  return request<API.ReviewSQLResponse>('/v1/sql-review', {
    method: 'POST',
    data: {
      instanceId: data.instanceId,
      databaseName: data.databaseName,
      sqlContent: data.sqlContent,
    },
  });
}

export async function checkSQLType(data: API.CheckSQLTypeRequest) {
  return request<API.CheckSQLTypeResponse>('/v1/sql-orders/check-type', {
    method: 'POST',
    data,
  });
}

// 查询相关API
export async function mysqlQuery(data: API.MySQLQueryRequest) {
  return request<API.QueryResponse>('/v1/query/mysql', {
    method: 'POST',
    data,
  });
}

export async function redisQuery(data: API.RedisQueryRequest) {
  return request<API.QueryResponse>('/v1/query/redis', {
    method: 'POST',
    data,
  });
}

export async function clickHouseQuery(data: API.ClickHouseQueryRequest) {
  return request<API.QueryResponse>('/v1/query/clickhouse', {
    method: 'POST',
    data,
  });
}

export async function hbaseQuery(data: API.HBaseQueryRequest) {
  return request<API.QueryResponse>('/v1/query/hbase', {
    method: 'POST',
    data,
  });
}

/** POST /v1/query/export：返回 JSON（含 downloadUrl），需再 GET 下载真实文件 */
export type QueryExportMetaResponse = {
  code?: number;
  message?: string;
  data?: {
    fileName?: string;
    file_name?: string;
    downloadUrl?: string;
    download_url?: string;
  };
};

export async function exportQuery(data: API.ExportQueryRequest) {
  return request<QueryExportMetaResponse>('/v1/query/export', {
    method: 'POST',
    data,
  });
}

/** downloadUrl 形如 /api/v1/query/download/xxx.xlsx */
export async function downloadQueryExport(downloadUrl: string) {
  const u = downloadUrl.trim();
  const path = u.startsWith('/api/') ? u.slice(5) : u.startsWith('/') ? u.slice(1) : u;
  return request<Blob>(path, {
    method: 'GET',
    responseType: 'blob',
  });
}

export async function listQueryHistory(params: API.ListHistoryRequest) {
  return request<API.ListHistoryResponse>('/v1/query/history', {
    method: 'GET',
    params,
  });
}

export async function deleteQueryHistory(id: string) {
  return request<API.CommonResponse>(`/v1/query/history/${id}`, {
    method: 'DELETE',
  });
}

// Binlog相关API
export async function parseBinlog(data: API.ParseBinlogRequest) {
  return request<API.ParseBinlogResponse>('/v1/binlog/parse', {
    method: 'POST',
    data,
  });
}

export async function listBinlogOrders(params: API.ListBinlogOrdersRequest) {
  return request<API.ListBinlogOrdersResponse>('/v1/binlog/orders', {
    method: 'GET',
    params: buildKratosPagedListQuery(params as Record<string, any>),
  });
}

export async function getBinlogOrder(id: string) {
  return request<API.GetBinlogOrderResponse>(`/v1/binlog/orders/${id}`, {
    method: 'GET',
  });
}

export async function getBinlogOrderSqls(id: string) {
  return request<API.GetBinlogOrderSqlsResponse>(`/v1/binlog/orders/${id}/sqls`, {
    method: 'GET',
  });
}

// Redis分析相关API
export async function analyzeHotKeys(data: API.AnalyzeHotKeysRequest) {
  return request<API.AnalyzeHotKeysResponse>('/v1/redis/hotkeys', {
    method: 'POST',
    data,
  });
}

export async function analyzeBigKeys(data: API.AnalyzeBigKeysRequest) {
  return request<API.AnalyzeBigKeysResponse>('/v1/redis/bigkeys', {
    method: 'POST',
    data,
  });
}

export async function listAnalysisTasks(params: API.ListAnalysisTasksRequest) {
  return request<API.ListAnalysisTasksResponse>('/v1/redis/tasks', {
    method: 'GET',
    params: buildKratosPagedListQuery(params as Record<string, any>),
  });
}

export async function getAnalysisTask(id: string) {
  return request<API.GetAnalysisTaskResponse>(`/v1/redis/tasks/${id}`, {
    method: 'GET',
  });
}

// 数据归档相关API
export async function listArchiveConfigs(params: API.ListArchiveConfigsRequest) {
  return request<API.ListArchiveConfigsResponse>('/v1/archive/configs', {
    method: 'GET',
    params: buildKratosPagedListQuery(params as Record<string, any>),
  });
}

export async function createArchiveConfig(data: API.CreateArchiveConfigRequest) {
  return request<API.CreateArchiveConfigResponse>('/v1/archive/configs', {
    method: 'POST',
    data,
  });
}

export async function updateArchiveConfig(id: string, data: API.UpdateArchiveConfigRequest) {
  return request<API.CommonResponse>(`/v1/archive/configs/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteArchiveConfig(id: string) {
  return request<API.CommonResponse>(`/v1/archive/configs/${id}`, {
    method: 'DELETE',
  });
}

export async function listArchiveRecords(params: API.ListArchiveRecordsRequest & { id: string }) {
  const { id, ...rest } = params;
  return request<API.ListArchiveRecordsResponse>(`/v1/archive/configs/${id}/records`, {
    method: 'GET',
    params: buildKratosPagedListQuery(rest as Record<string, any>),
  });
}

// 实例管理相关API
export async function listInstances(params: API.ListInstancesRequest) {
  return request<API.ListInstancesResponse>('/v1/instances', {
    method: 'GET',
    params: buildKratosPagedListQuery(params as Record<string, any>),
  });
}

export async function createInstance(data: API.CreateInstanceRequest) {
  return request<API.CreateInstanceResponse>('/v1/instances', {
    method: 'POST',
    data,
  });
}

export async function getInstance(id: string) {
  return request<API.GetInstanceResponse>(`/v1/instances/${id}`, {
    method: 'GET',
  });
}

export async function updateInstance(id: string, data: API.UpdateInstanceRequest) {
  return request<API.CommonResponse>(`/v1/instances/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteInstance(id: string) {
  return request<API.CommonResponse>(`/v1/instances/${id}`, {
    method: 'DELETE',
  });
}

export async function testConnection(id: string) {
  return request<API.TestConnectionResponse>(`/v1/instances/${id}/test`, {
    method: 'POST',
  });
}

// 获取数据库列表
export async function listDatabases(instanceId: string) {
  return request<{ code: number; message: string; databases: string[] }>(
    `/v1/instances/${instanceId}/databases`,
    {
      method: 'GET',
    }
  );
}

// 获取表列表
export async function listTables(instanceId: string, database: string) {
  return request<{ code: number; message: string; tables: string[] }>(
    `/v1/instances/${instanceId}/databases/${database}/tables`,
    {
      method: 'GET',
    }
  );
}

// 获取表DDL
export async function getTableDDL(instanceId: string, database: string, table: string) {
  return request<{ code: number; message: string; ddl: string }>(
    `/v1/instances/${instanceId}/databases/${database}/tables/${table}/ddl`,
    {
      method: 'GET',
    }
  );
}

// 获取执行计划
export async function getExplain(instanceId: string, database: string, sql: string) {
  return request<{ code: number; message: string; explain: string }>(
    `/v1/instances/${instanceId}/explain`,
    {
      method: 'POST',
      data: { database, sql },
    }
  );
}

// 权限管理相关API
export async function listUsers(params: API.ListUsersRequest) {
  // 手写路由读 query.Get("page") / page_size，与 pageSize 不一致会导致一直用默认 20
  const p = params as Record<string, any>;
  const page = Number(p.page ?? 1) || 1;
  const pageSize = Number(p.pageSize ?? p.page_size ?? 20) || 20;
  return request<API.ListUsersResponse>('/v1/permission/users', {
    method: 'GET',
    params: { page, page_size: pageSize },
  });
}

export async function createUser(data: API.CreateUserRequest) {
  return request<API.CreateUserResponse>('/v1/permission/users', {
    method: 'POST',
    data,
  });
}

export async function updateUser(idOrData: string | API.UpdateUserRequest, data?: API.UpdateUserRequest) {
  if (typeof idOrData === 'string') {
    return request<API.UpdateUserResponse>(`/v1/permission/users/${idOrData}`, {
      method: 'PUT',
      data,
    });
  }
  return request<API.UpdateUserResponse>(`/v1/auth/user`, {
    method: 'PUT',
    data: idOrData,
  });
}

export async function deleteUser(id: string) {
  return request<API.CommonResponse>(`/v1/permission/users/${id}`, {
    method: 'DELETE',
  });
}

export async function listRoles(params: API.ListRolesRequest) {
  return request<API.ListRolesResponse>('/v1/roles', {
    method: 'GET',
    params: buildKratosPagedListQuery(params as Record<string, any>),
  });
}

export async function createRole(data: API.CreateRoleRequest) {
  return request<API.CreateRoleResponse>('/v1/roles', {
    method: 'POST',
    data,
  });
}

export async function updateRole(id: string, data: API.UpdateRoleRequest) {
  return request<API.CommonResponse>(`/v1/roles/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteRole(id: string) {
  return request<API.CommonResponse>(`/v1/roles/${id}`, {
    method: 'DELETE',
  });
}

export async function getRolePermissions(roleId: string) {
  return request<API.ListPermissionsResponse>(`/v1/roles/${roleId}/permissions`, {
    method: 'GET',
  });
}

export async function assignPermissionsToRole(roleId: string, permissionIds: string[]) {
  return request<API.CommonResponse>(`/v1/roles/${roleId}/permissions`, {
    method: 'POST',
    data: { permissionIds: permissionIds.map(Number) },
  });
}

/** 用户直授实例权限（读：本人或 user:manage；写：user:manage） */
export async function getUserInstancePermissions(userId: string) {
  return request<{
    code: number;
    message: string;
    list?: { instance_id?: number; instanceId?: number; permission_type?: string; permissionType?: string }[];
  }>(`/v1/users/${userId}/instance-permissions`, { method: 'GET' });
}

export async function replaceUserInstancePermissions(
  userId: string,
  items: { instance_id: number; permission_type: string }[],
) {
  return request<API.CommonResponse>(`/v1/users/${userId}/instance-permissions`, {
    method: 'PUT',
    data: { items },
  });
}

/** 角色实例权限（需 role:manage） */
export async function getRoleInstancePermissions(roleId: string) {
  return request<{
    code: number;
    message: string;
    list?: { instance_id?: number; instanceId?: number; permission_type?: string; permissionType?: string }[];
  }>(`/v1/roles/${roleId}/instance-permissions`, { method: 'GET' });
}

export async function replaceRoleInstancePermissions(
  roleId: string,
  items: { instance_id: number; permission_type: string }[],
) {
  return request<API.CommonResponse>(`/v1/roles/${roleId}/instance-permissions`, {
    method: 'PUT',
    data: { items },
  });
}

export async function listPermissions(params: API.ListPermissionsRequest) {
  return request<API.ListPermissionsResponse>('/v1/permissions', {
    method: 'GET',
    params: buildKratosPagedListQuery(params as Record<string, any>),
  });
}

export async function createPermission(data: API.CreatePermissionRequest) {
  return request<API.CreatePermissionResponse>('/v1/permissions', {
    method: 'POST',
    data,
  });
}

export async function updatePermission(id: string, data: API.UpdatePermissionRequest) {
  return request<API.CommonResponse>(`/v1/permissions/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deletePermission(id: string) {
  return request<API.CommonResponse>(`/v1/permissions/${id}`, {
    method: 'DELETE',
  });
}

// SQL收藏相关API
export async function listFavorites(params: API.ListFavoritesRequest) {
  return request<API.ListFavoritesResponse>('/v1/favorites', {
    method: 'GET',
    params,
  });
}

export async function createFavorite(data: API.CreateFavoriteRequest) {
  return request<API.CreateFavoriteResponse>('/v1/favorites', {
    method: 'POST',
    data,
  });
}

export async function updateFavorite(id: string, data: API.UpdateFavoriteRequest) {
  return request<API.CommonResponse>(`/v1/favorites/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteFavorite(id: string) {
  return request<API.CommonResponse>(`/v1/favorites/${id}`, {
    method: 'DELETE',
  });
}

export async function pinFavorite(id: string) {
  return request<API.CommonResponse>(`/v1/favorites/${id}/pin`, {
    method: 'POST',
  });
}

// MyBatis生成相关API
export async function generateMybatisXML(data: API.GenerateMybatisXMLRequest) {
  return request<API.GenerateMybatisXMLResponse>('/v1/mybatis/generate', {
    method: 'POST',
    data,
  });
}

// 宽表服务相关API
export async function listWidetableConfigs(params: { page?: number; page_size?: number; keyword?: string }) {
  return request<any>('/v1/widetable/configs', {
    method: 'GET',
    params,
  });
}

export async function createWidetableConfig(data: any) {
  return request<any>('/v1/widetable/configs', {
    method: 'POST',
    data,
  });
}

export async function updateWidetableConfig(id: number, data: any) {
  return request<any>(`/v1/widetable/configs/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteWidetableConfig(id: number) {
  return request<any>(`/v1/widetable/configs/${id}`, {
    method: 'DELETE',
  });
}

export async function getWidetableConfig(id: number) {
  return request<any>(`/v1/widetable/configs/${id}`, {
    method: 'GET',
  });
}

export async function previewWidetableSQL(id: number, data: { limit?: number }) {
  return request<any>(`/v1/widetable/configs/${id}/preview-sql`, {
    method: 'POST',
    data,
  });
}

export async function queryWidetable(id: number, data: { page?: number; page_size?: number; extra_condition?: string; order_by?: string }) {
  return request<any>(`/v1/widetable/configs/${id}/query`, {
    method: 'POST',
    data,
  });
}

export async function syncWidetable(id: number, data: { batch_size?: number }) {
  return request<any>(`/v1/widetable/configs/${id}/sync`, {
    method: 'POST',
    data,
  });
}

export async function listWidetableSyncRecords(configId: number, params?: { page?: number; page_size?: number }) {
  return request<any>(`/v1/widetable/configs/${configId}/sync-records`, {
    method: 'GET',
    params,
  });
}

// 进程列表相关API
export async function getProcessList(instanceId: string | number) {
  return request<any>(`/v1/instances/${instanceId}/processes`, {
    method: 'GET',
  });
}

export async function killProcess(instanceId: string | number, processId: string) {
  return request<any>(`/v1/instances/${instanceId}/processes/${encodeURIComponent(processId)}`, {
    method: 'DELETE',
  });
}

// 表结构相关API
export async function getTableColumns(instanceId: string | number, database: string, table: string) {
  return request<any>(`/v1/instances/${instanceId}/databases/${database}/tables/${table}/columns`, {
    method: 'GET',
  });
}

// SQL优化建议
export async function getAdvisorSuggestions(instanceId: string | number, data: { database: string; table?: string; sql: string }) {
  return request<any>(`/v1/instances/${instanceId}/advisor`, {
    method: 'POST',
    data,
  });
}

// 通知渠道配置相关API
export async function getNotificationConfigs() {
  return request<any>('/v1/notification/configs', { method: 'GET' });
}

export async function updateNotificationConfig(
  channel: string,
  data: { enabled: boolean; webhook_url: string; extra_config?: string },
) {
  return request<any>(`/v1/notification/configs/${channel}`, { method: 'PUT', data });
}

export async function testNotificationChannel(channel: string) {
  return request<any>(`/v1/notification/test/${channel}`, { method: 'POST' });
}

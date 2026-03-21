declare namespace API {
  // 通用响应
  interface CommonResponse {
    code: number;
    message: string;
  }

  // 分页响应
  interface PageResponse {
    total: number;
    page: number;
    page_size: number;
  }

  // 用户相关
  interface User {
    id: string;
    username: string;
    email: string;
    realName: string;
    phone?: string;
    department?: string;
    status: number;
    role?: string;
    createdAt?: string;
    lastLoginAt?: string;
  }

  interface LoginRequest {
    username: string;
    password: string;
  }

  interface LoginResponse extends CommonResponse {
    data?: {
      token: string;
      user: User;
    };
  }

  interface RegisterRequest {
    username: string;
    password: string;
    email: string;
    realName: string;
    phone?: string;
    department?: string;
  }

  interface RegisterResponse extends CommonResponse {
    data?: User;
  }

  // SQL工单相关
  interface SQLOrder {
    id: string;
    workId: string;
    instanceId: string;
    databaseName: string;
    sqlContent: string;
    orderType: number; // 0-DDL, 1-DML, 2-导出
    status: number;
    riskLevel: number;
    submitterId: string;
    auditorId?: string;
    executorId?: string;
    rejectReason?: string;
    remark?: string;
    needBackup: boolean;
    executeMethod: number;
    createdAt: string;
    updatedAt: string;
  }

  interface CreateOrderRequest {
    instanceId: string;
    databaseName: string;
    sqlContent: string;
    orderType: number;
    remark?: string;
    needBackup?: boolean;
    executeMethod?: number;
  }

  interface CreateOrderResponse extends CommonResponse {
    data?: SQLOrder;
  }

  interface ListOrdersRequest {
    page?: number;
    pageSize?: number;
    status?: number;
    orderType?: number;
    instanceId?: string;
  }

  interface ListOrdersResponse extends CommonResponse {
    data?: {
      list?: SQLOrder[];
      page?: PageResponse;
    };
  }

  interface GetOrderResponse extends CommonResponse {
    data?: SQLOrder;
  }

  interface ApproveOrderRequest {
    remark?: string;
  }

  interface RejectOrderRequest {
    reason: string;
  }

  interface AuditSQLRequest {
    instanceId: string;
    databaseName: string;
    sqlContent: string;
  }

  interface AuditSQLResponse extends CommonResponse {
    data?: {
      records: SQLReviewRecord[];
    };
  }

  interface ReviewSQLRequest {
    instanceId: string;
    databaseName: string;
    sqlContent: string;
  }

  interface ReviewSQLResponse extends CommonResponse {
    data?: SQLReviewRecord[];
  }

  interface SQLReviewRecord {
    id: string;
    orderId: string;
    sequence: number;
    sqlText: string;
    errorLevel: number;
    errorMessage?: string;
    stageStatus?: string;
    affectedRows?: number;
    executeTime?: string;
  }

  interface CheckSQLTypeRequest {
    sqlContent: string;
  }

  interface CheckSQLTypeResponse extends CommonResponse {
    data?: {
      sqlType: string;
      canSelfExecute: boolean;
    };
  }

  interface SQLOrderItem {
    id: string;
    orderId: string;
    sequence: number;
    sqlText: string;
    status: number;
    errorMessage?: string;
    affectedRows?: number;
    executeTime?: string;
    createdAt: string;
  }

  interface GetOrderItemsResponse extends CommonResponse {
    data?: SQLOrderItem[];
  }

  interface GetOrderRecordsResponse extends CommonResponse {
    data?: SQLReviewRecord[];
  }

  // 查询相关
  interface MySQLQueryRequest {
    instanceId: string;
    database: string;
    sql: string;
  }

  interface RedisQueryRequest {
    instanceId: string;
    command: string;
  }

  interface ClickHouseQueryRequest {
    instanceId: string;
    database: string;
    sql: string;
  }

  interface HBaseQueryRequest {
    instanceId: string;
    sql: string;
    limit?: number;
  }

  interface QueryResponse extends CommonResponse {
    data?: {
      columns: string[];
      rows: any[][];
      count?: number;
      time?: string;
    };
  }

  interface ExportQueryRequest {
    instanceId: string;
    database: string;
    sql: string;
    format: 'excel' | 'csv' | 'json';
  }

  interface ListHistoryRequest {
    page?: number;
    pageSize?: number;
    instanceId?: string;
  }

  interface ListHistoryResponse extends CommonResponse {
    data?: {
      list?: QueryHistory[];
      page?: PageResponse;
    };
  }

  interface QueryHistory {
    id: string;
    userId: string;
    instanceId: string;
    databaseName: string;
    sqlText: string;
    executeTime: number;
    affectedRows: number;
    resultCount: number;
    createdAt: string;
  }

  // Binlog相关
  interface ParseBinlogRequest {
    instanceId: string;
    databaseName: string;
    tableName: string;
    binlogFile: string;
    startTime: string;
    endTime: string;
    remark?: string;
  }

  interface ParseBinlogResponse extends CommonResponse {
    data?: {
      orderId: string;
      workId: string;
    };
  }

  interface ListBinlogOrdersRequest {
    page?: number;
    pageSize?: number;
    instanceId?: string;
  }

  interface ListBinlogOrdersResponse extends CommonResponse {
    list?: BinlogOrder[];
    pageResponse?: PageResponse;
  }

  interface BinlogOrder {
    id: string;
    workId: string;
    instanceId: string;
    databaseName: string;
    tableName: string;
    binlogFile: string;
    startTime: string;
    endTime: string;
    status: number;
    submitterId: string;
    remark?: string;
    createdAt: string;
  }

  interface GetBinlogOrderResponse extends CommonResponse {
    data?: BinlogOrder;
  }

  interface BinlogRollbackSql {
    id: string;
    orderId: string;
    sequence: number;
    tableName: string;
    operationType: string;
    originalSql?: string;
    rollbackSql: string;
    createdAt: string;
  }

  interface GetBinlogOrderSqlsResponse extends CommonResponse {
    list?: BinlogRollbackSql[];
    data?: BinlogRollbackSql[];
  }

  // Redis分析相关
  interface AnalyzeHotKeysRequest {
    instanceId: string;
    timeWindowSeconds: number;
  }

  interface AnalyzeHotKeysResponse extends CommonResponse {
    data?: {
      taskId: string;
      workId: string;
      message: string;
    };
  }

  interface AnalyzeBigKeysRequest {
    instanceId: string;
    sizeThreshold: number;
  }

  interface AnalyzeBigKeysResponse extends CommonResponse {
    data?: {
      taskId: string;
      workId: string;
      message: string;
    };
  }

  interface ListAnalysisTasksRequest {
    page?: number;
    pageSize?: number;
    instanceId?: string;
  }

  interface ListAnalysisTasksResponse extends CommonResponse {
    list?: RedisAnalysisTask[];
    pageResponse?: PageResponse;
  }

  interface GetAnalysisTaskResponse extends CommonResponse {
    data?: RedisAnalysisTask;
  }

  interface RedisAnalysisTask {
    id: string;
    instanceId: string;
    taskType: string;
    status: number;
    submitterId: string;
    result?: string;
    errorMessage?: string;
    createdAt: string;
    updatedAt: string;
  }

  // 数据归档相关
  interface ArchiveConfig {
    id: string;
    instanceId: string;
    sourceDatabase: string;
    sourceTable: string;
    destInstanceId?: string;
    destDatabase: string;
    destTable: string;
    archiveField: string;
    retentionDays: number;
    archiveCondition?: string;
    isEnabled: boolean;
    cronExpression: string;
    owner: string;
    createdAt: string;
    updatedAt: string;
  }

  interface ListArchiveConfigsRequest {
    page?: number;
    pageSize?: number;
  }

  interface ListArchiveConfigsResponse extends CommonResponse {
    list?: ArchiveConfig[];
    pageResponse?: PageResponse;
  }

  interface CreateArchiveConfigRequest {
    instanceId: string;
    sourceDatabase: string;
    sourceTable: string;
    destInstanceId?: string;
    destDatabase: string;
    destTable: string;
    archiveField: string;
    retentionDays: number;
    archiveCondition?: string;
    isEnabled?: boolean;
    cronExpression: string;
  }

  interface CreateArchiveConfigResponse extends CommonResponse {
    data?: ArchiveConfig;
  }

  interface UpdateArchiveConfigRequest {
    archiveCondition?: string;
    isEnabled?: boolean;
    cronExpression?: string;
    retentionDays?: number;
  }

  interface ArchiveRecord {
    id: string;
    configId: string;
    workId: string;
    status: number;
    archivedCount?: number;
    errorMessage?: string;
    startTime?: string;
    endTime?: string;
    createdAt: string;
  }

  interface ListArchiveRecordsRequest {
    id?: string;
    page?: number;
    pageSize?: number;
  }

  interface ListArchiveRecordsResponse extends CommonResponse {
    list?: ArchiveRecord[];
    pageResponse?: PageResponse;
  }

  // 实例管理相关
  interface DatabaseInstance {
    id: string;
    name: string;
    connectionName: string;
    dbType: string;
    host: string;
    port: number;
    username: string;
    env: string;
    status: number;
    owner: string;
    description?: string;
    createdAt: string;
    updatedAt: string;
  }

  interface ListInstancesRequest {
    page?: number;
    pageSize?: number;
    dbType?: string;
    env?: string;
  }

  interface ListInstancesResponse extends CommonResponse {
    list?: DatabaseInstance[];
    pageResponse?: PageResponse;
  }

  interface CreateInstanceRequest {
    name: string;
    connectionName: string;
    dbType: string;
    host: string;
    port: number;
    username: string;
    password: string;
    env: string;
    description?: string;
  }

  interface CreateInstanceResponse extends CommonResponse {
    data?: DatabaseInstance;
  }

  interface GetInstanceResponse extends CommonResponse {
    data?: DatabaseInstance;
  }

  interface UpdateInstanceRequest {
    name?: string;
    host?: string;
    port?: number;
    username?: string;
    password?: string;
    env?: string;
    status?: number;
    description?: string;
  }

  interface TestConnectionResponse extends CommonResponse {
    data?: {
      success: boolean;
      message: string;
    };
  }

  // 权限管理相关
  interface ListUsersRequest {
    page?: number;
    pageSize?: number;
  }

  interface ListUsersResponse extends CommonResponse {
    list?: User[];
    pageResponse?: PageResponse;
  }

  interface CreateUserRequest {
    username: string;
    password: string;
    email: string;
    realName: string;
    phone?: string;
    department?: string;
  }

  interface CreateUserResponse extends CommonResponse {
    data?: User;
  }

  interface UpdateUserRequest {
    email?: string;
    realName?: string;
    phone?: string;
    department?: string;
    position?: string;
    description?: string;
  }

  interface UpdateUserResponse {
    code: number;
    message: string;
    data?: User;
  }

  interface Role {
    id: string;
    name: string;
    code: string;
    description?: string;
    createdAt: string;
    updatedAt: string;
  }

  interface ListRolesRequest {
    page?: number;
    pageSize?: number;
  }

  interface ListRolesResponse extends CommonResponse {
    list?: Role[];
    pageResponse?: PageResponse;
  }

  interface CreateRoleRequest {
    name: string;
    code: string;
    description?: string;
  }

  interface CreateRoleResponse extends CommonResponse {
    data?: Role;
  }

  interface UpdateRoleRequest {
    name?: string;
    description?: string;
  }

  interface Permission {
    id: string;
    name: string;
    code: string;
    resourceType: string;
    resourcePath: string;
    description?: string;
    createdAt: string;
    updatedAt: string;
  }

  interface ListPermissionsRequest {
    page?: number;
    pageSize?: number;
  }

  interface ListPermissionsResponse extends CommonResponse {
    list?: Permission[];
    pageResponse?: PageResponse;
  }

  interface CreatePermissionRequest {
    name: string;
    code: string;
    resourceType: string;
    resourcePath: string;
    description?: string;
  }

  interface CreatePermissionResponse extends CommonResponse {
    data?: Permission;
  }

  interface UpdatePermissionRequest {
    name?: string;
    resourceType?: string;
    resourcePath?: string;
    description?: string;
  }

  // SQL收藏相关
  interface Favorite {
    id: string;
    userId: string;
    instanceId: string;
    databaseName: string;
    sqlText: string;
    title: string;
    description?: string;
    isPinned: boolean;
    createdAt: string;
    updatedAt: string;
  }

  interface ListFavoritesRequest {
    page?: number;
    pageSize?: number;
  }

  interface ListFavoritesResponse extends CommonResponse {
    favorites?: Favorite[];
    pageResponse?: PageResponse;
  }

  interface CreateFavoriteRequest {
    instanceId: string;
    databaseName: string;
    sqlText: string;
    title: string;
    description?: string;
  }

  interface CreateFavoriteResponse extends CommonResponse {
    data?: Favorite;
  }

  interface UpdateFavoriteRequest {
    title?: string;
    description?: string;
    sqlText?: string;
  }

  // 进程列表相关
  interface ProcessInfo {
    id: string;
    user: string;
    host: string;
    db: string;
    command: string;
    time: number;
    state: string;
    info: string;
    query_id?: string;
    progress?: string;
  }

  interface GetProcessListResponse extends CommonResponse {
    processes?: ProcessInfo[];
  }

  // SQL优化建议相关
  interface AdvisorSuggestion {
    level: string;
    rule: string;
    message: string;
    detail?: string;
  }

  interface GetAdvisorSuggestionsResponse extends CommonResponse {
    suggestions?: AdvisorSuggestion[];
  }

  // 表列信息
  interface TableColumn {
    name: string;
    type: string;
    null?: string;
    key?: string;
    default?: string;
    extra?: string;
    comment?: string;
  }

  interface GetTableColumnsResponse extends CommonResponse {
    columns?: TableColumn[];
  }

  // MyBatis生成相关
  interface GenerateMybatisXMLRequest {
    ddl: string;
  }

  interface GenerateMybatisXMLResponse extends CommonResponse {
    data?: {
      xml: string;
    };
  }
}

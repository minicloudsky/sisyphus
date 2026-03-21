# Sisyphus API 文档

## 概述

Sisyphus 是一个数据库智能平台，提供完整的 RESTful API 和 gRPC 接口。所有 API 都基于 Protobuf 定义，支持 HTTP 和 gRPC 双协议。

## 基础信息

- **Base URL**: `http://localhost:8000`
- **API Version**: `v1`
- **Content-Type**: `application/json`
- **认证方式**: JWT Token（Bearer Token）

## 认证

大部分 API 需要 JWT Token 认证。在请求头中添加：

```
Authorization: Bearer <token>
```

## API 列表

### 1. 认证服务 (AuthService)

#### 1.1 用户注册

**接口**: `POST /api/v1/auth/register`

**请求体**:
```json
{
  "username": "string",
  "password": "string",
  "email": "string",
  "real_name": "string",
  "phone": "string",
  "department": "string"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "username": "string",
    "email": "string",
    "real_name": "string",
    "phone": "string",
    "department": "string",
    "status": 1
  }
}
```

#### 1.2 用户登录

**接口**: `POST /api/v1/auth/login`

**请求体**:
```json
{
  "username": "string",
  "password": "string"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "string",
      "email": "string",
      "real_name": "string",
      "phone": "string",
      "department": "string",
      "status": 1
    }
  }
}
```

### 2. SQL工单服务 (SQLOrderService)

#### 2.1 创建SQL工单

**接口**: `POST /api/v1/sql-orders`

**请求体**:
```json
{
  "instance_id": 1,
  "database_name": "test_db",
  "sql_content": "CREATE TABLE test (id INT PRIMARY KEY)",
  "order_type": 1,
  "need_backup": true,
  "remark": "创建测试表"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "work_id": "20240117001",
    "instance_id": 1,
    "database_name": "test_db",
    "sql_content": "CREATE TABLE test (id INT PRIMARY KEY)",
    "order_type": 1,
    "status": 0,
    "risk_level": 1,
    "need_backup": true,
    "remark": "创建测试表",
    "created_at": "2024-01-17T10:00:00Z"
  }
}
```

#### 2.2 获取SQL工单列表

**接口**: `GET /api/v1/sql-orders?page=1&page_size=10&status=0`

**查询参数**:
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认10）
- `status`: 工单状态（可选，0:待审核, 1:审核通过, 2:审核驳回, 3:执行中, 4:执行成功, 5:执行失败）
- `submitter_id`: 提交人ID（可选）

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "list": [...],
  "page_response": {
    "page": 1,
    "page_size": 10,
    "total": 100
  }
}
```

#### 2.3 审核通过SQL工单

**接口**: `PUT /api/v1/sql-orders/{id}/approve`

**响应**:
```json
{
  "code": 200,
  "message": "审核通过"
}
```

#### 2.4 驳回SQL工单

**接口**: `PUT /api/v1/sql-orders/{id}/reject`

**请求体**:
```json
{
  "reason": "SQL存在风险"
}
```

#### 2.5 执行SQL工单

**接口**: `POST /api/v1/sql-orders/{id}/execute`

**请求体**:
```json
{
  "execute_method": 0
}
```

`execute_method`: 0-自动选择, 1-gh-ost, 2-goInception

### 3. 数据查询服务 (QueryService)

#### 3.1 MySQL查询

**接口**: `POST /api/v1/query/mysql`

**请求体**:
```json
{
  "instance_id": 1,
  "database": "test_db",
  "sql": "SELECT * FROM users LIMIT 10",
  "limit": 100
}
```

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "columns": ["id", "username", "email"],
    "rows": [
      {"id": 1, "username": "user1", "email": "user1@example.com"},
      {"id": 2, "username": "user2", "email": "user2@example.com"}
    ],
    "count": 2,
    "time_seconds": 0.001
  }
}
```

#### 3.2 Redis查询

**接口**: `POST /api/v1/query/redis`

**请求体**:
```json
{
  "instance_id": 1,
  "command": "GET key1"
}
```

#### 3.3 导出查询结果

**接口**: `POST /api/v1/query/export`

**请求体**:
```json
{
  "instance_id": 1,
  "database": "test_db",
  "sql": "SELECT * FROM users",
  "format": "excel"
}
```

`format`: excel, csv, json

### 4. Binlog回滚服务 (BinlogService)

#### 4.1 解析Binlog

**接口**: `POST /api/v1/binlog/parse`

**请求体**:
```json
{
  "instance_id": 1,
  "database_name": "test_db",
  "start_time": "2024-01-17T10:00:00Z",
  "stop_time": "2024-01-17T11:00:00Z",
  "binlog_file": "mysql-bin.000001",
  "table_list": ["users", "orders"]
}
```

### 5. Redis分析服务 (RedisService)

#### 5.1 分析热key

**接口**: `POST /api/v1/redis/analyze/hotkeys`

**请求体**:
```json
{
  "instance_id": 1,
  "time_window_seconds": 60,
  "threshold": 100
}
```

#### 5.2 分析大key

**接口**: `POST /api/v1/redis/analyze/bigkeys`

**请求体**:
```json
{
  "instance_id": 1,
  "size_threshold": 1048576
}
```

### 6. 数据归档服务 (ArchiveService)

#### 6.1 获取归档配置列表

**接口**: `GET /api/v1/archive/configs?page=1&page_size=10`

#### 6.2 创建归档配置

**接口**: `POST /api/v1/archive/configs`

**请求体**:
```json
{
  "instance_id": 1,
  "source_database": "source_db",
  "source_table": "source_table",
  "dest_instance_id": 2,
  "dest_database": "archive_db",
  "dest_table": "archive_table",
  "archive_field": "created_at",
  "retention_days": 90,
  "archive_condition": "created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)",
  "is_enabled": true,
  "cron_expression": "0 2 * * *"
}
```

### 7. 实例管理服务 (InstanceService)

#### 7.1 获取实例列表

**接口**: `GET /api/v1/instances?page=1&page_size=10&db_type=mysql&env=prod`

#### 7.2 创建实例

**接口**: `POST /api/v1/instances`

**请求体**:
```json
{
  "name": "生产MySQL-01",
  "db_type": "mysql",
  "host": "192.168.1.100",
  "port": 3306,
  "username": "root",
  "password": "password",
  "env": "prod",
  "region": "cn-beijing",
  "description": "生产环境主库"
}
```

### 8. gh-ost管理服务 (GhOstService)

#### 8.1 获取执行进度

**接口**: `GET /api/v1/ghost/progress?work_id=20240117001`

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "work_id": "20240117001",
    "table_name": "users",
    "total_rows": 1000000,
    "copied_rows": 500000,
    "progress_percent": 50.0,
    "estimated_seconds_remaining": 3600,
    "status": "running",
    "message": "正在执行中"
  }
}
```

#### 8.2 暂停执行

**接口**: `PUT /api/v1/ghost/throttle`

**请求体**:
```json
{
  "table_name": "users",
  "work_id": "20240117001"
}
```

#### 8.3 恢复执行

**接口**: `PUT /api/v1/ghost/no-throttle`

#### 8.4 停止执行

**接口**: `POST /api/v1/ghost/stop`

**请求体**:
```json
{
  "work_id": "20240117001"
}
```

### 9. 权限管理服务 (PermissionService)

#### 9.1 创建权限

**接口**: `POST /api/v1/permissions`

**请求体**:
```json
{
  "name": "SQL工单审核",
  "code": "sql_order:approve",
  "resource": "sql_order",
  "action": "approve",
  "description": "审核SQL工单的权限"
}
```

#### 9.2 创建角色

**接口**: `POST /api/v1/roles`

**请求体**:
```json
{
  "name": "数据库管理员",
  "code": "dba",
  "description": "数据库管理员角色"
}
```

#### 9.3 分配权限给角色

**接口**: `POST /api/v1/roles/{id}/permissions`

**请求体**:
```json
{
  "permission_ids": [1, 2, 3]
}
```

#### 9.4 分配角色给用户

**接口**: `POST /api/v1/users/{id}/roles`

**请求体**:
```json
{
  "role_ids": [1, 2]
}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 响应格式

所有 API 响应都遵循统一的格式：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

对于列表接口，响应格式为：

```json
{
  "code": 200,
  "message": "success",
  "list": [],
  "page_response": {
    "page": 1,
    "page_size": 10,
    "total": 100
  }
}
```

## Protobuf 定义

所有 API 的 Protobuf 定义文件位于 `server/api/proto/v1/` 目录下：

- `common.proto` - 通用消息定义
- `auth.proto` - 认证服务
- `sql_order.proto` - SQL工单服务
- `query.proto` - 数据查询服务
- `binlog.proto` - Binlog回滚服务
- `redis.proto` - Redis分析服务
- `archive.proto` - 数据归档服务
- `instance.proto` - 实例管理服务
- `ghost.proto` - gh-ost管理服务
- `permission.proto` - 权限管理服务

## 生成代码

使用以下命令生成 Protobuf 代码：

```bash
cd server
make proto
```

这将生成：
- `*.pb.go` - Protobuf 消息定义
- `*_grpc.pb.go` - gRPC 服务接口
- `*_http.pb.go` - HTTP 服务接口

## 实现服务

服务实现需要实现生成的接口。例如，`AuthService` 需要实现 `AuthServiceHTTPServer` 接口：

```go
type AuthServiceHTTPServer interface {
    Login(context.Context, *LoginRequest) (*LoginResponse, error)
    Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
}
```

然后在 HTTP 服务器中注册：

```go
v1.RegisterAuthServiceHTTPServer(srv, authService)
```

## 更多信息

- 项目说明（默认英文）: [`README.md`](../README.md) · [中文 `README.zh-CN.md`](../README.zh-CN.md)
- 快速开始: [`QUICKSTART.md`](../QUICKSTART.md) · [中文 `QUICKSTART.zh-CN.md`](../QUICKSTART.zh-CN.md)
- Protobuf: `server/api/proto/v1/`

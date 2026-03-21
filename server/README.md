# Sisyphus Backend

数据库智能平台后端服务

## 技术栈

- Go 1.21+
- go-kratos v2
- GORM
- gRPC + HTTP
- JWT认证
- Redis
- MySQL

## 项目结构

```
server/
├── cmd/sisyphus/          # 应用入口
├── internal/
│   ├── biz/               # 业务逻辑层
│   ├── data/               # 数据访问层
│   ├── service/            # 服务层
│   ├── server/             # 服务器配置
│   ├── middleware/         # 中间件
│   └── conf/               # 配置
├── api/                    # API定义
├── configs/                # 配置文件
└── deploy/                 # 部署相关
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 生成代码

```bash
# 安装wire
go install github.com/google/wire/cmd/wire@latest

# 生成wire代码
wire gen ./cmd/sisyphus

# 生成protobuf代码（如果有）
protoc --go_out=. --go-grpc_out=. api/proto/*.proto
```

### 3. 配置

修改 `configs/config.yaml` 中的数据库和Redis配置

### 4. 运行

```bash
go run cmd/sisyphus/main.go -conf configs/config.yaml
```

## 功能模块

- 用户认证和权限管理（RBAC）
- SQL工单审核和执行
- 数据查询和导出
- Binlog回滚
- Redis热key分析
- 数据归档

## 部署

### Docker

```bash
docker build -t sisyphus/server:latest .
docker run -p 8000:8000 -p 9000:9000 sisyphus/server:latest
```

### Kubernetes

```bash
kubectl apply -f deploy/k8s/
```

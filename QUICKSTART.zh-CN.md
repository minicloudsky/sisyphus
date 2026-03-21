# Sisyphus 快速开始指南

**语言 / Languages:** [English](QUICKSTART.md) | **简体中文**（本文）

## 1. 环境准备

### 检查MySQL和Redis

```bash
# 检查MySQL
brew services list | grep mysql
# 如果未运行，启动MySQL
brew services start mysql

# 检查Redis
redis-cli ping
# 如果未运行，启动Redis
brew services start redis
```

## 2. 初始化数据库

### 方法1: 使用初始化脚本（推荐）

```bash
cd sisyphus
./scripts/setup.sh
```

脚本会提示输入MySQL root密码，然后自动：

- 创建数据库
- 初始化表结构
- 插入测试数据
- 更新配置文件

### 方法2: 手动初始化

```bash
# 1. 创建数据库（需要输入MySQL密码）
mysql -u root -p < sql/init.sql

# 2. 插入测试数据
mysql -u root -p sisyphus < sql/test_data.sql
```

若使用当前仓库的 `migrations/` 脚本，路径应为 `migrations/init.sql` 与 `migrations/test_data.sql`。

## 3. 配置数据库连接

编辑 `server/configs/config.yaml`，修改数据库连接：

```yaml
data:
  database:
    source: root:你的密码@tcp(127.0.0.1:3306)/sisyphus?charset=utf8mb4&parseTime=True&loc=Local
```

如果MySQL无密码，使用：

```yaml
data:
  database:
    source: root@tcp(127.0.0.1:3306)/sisyphus?charset=utf8mb4&parseTime=True&loc=Local
```

## 4. 运行后端服务

```bash
cd server

# 安装依赖
go mod download

# 生成wire代码（如果需要）
make wire

# 运行服务
go run cmd/sisyphus/main.go -conf configs/config.yaml
```

或者使用Makefile：

```bash
make run
```

## 5. 测试

服务启动后，访问：

- HTTP: [http://localhost:8000](http://localhost:8000)
- gRPC: localhost:9000

### 测试账号

- 用户名: `admin` / 密码: `Admin@123`
- 用户名: `dba` / 密码: `Admin@123`
- 用户名: `developer` / 密码: `Admin@123`

## 6. 常见问题

### MySQL连接失败

1. 检查MySQL是否运行: `brew services list | grep mysql`
2. 检查密码是否正确
3. 尝试重置MySQL密码:
  ```bash
   mysql_secure_installation
  ```

### Redis连接失败

1. 检查Redis是否运行: `redis-cli ping`
2. 启动Redis: `brew services start redis`

### Go依赖问题

```bash
cd server
go mod tidy
go mod download
```

### Wire代码生成问题

```bash
cd server
go install github.com/google/wire/cmd/wire@latest
make wire
```

## 7. 下一步

- 项目概览：[README（中文）](README.zh-CN.md) · [English README](README.md)
- 使用手册：[docs/GUIDE.zh-CN.md](docs/GUIDE.zh-CN.md) · [English User Guide](docs/GUIDE.md)
- 参与贡献：[CONTRIBUTING.md](CONTRIBUTING.md)


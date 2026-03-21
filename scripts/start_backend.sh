#!/bin/bash

set -e

echo "=========================================="
echo "启动 Sisyphus 后端服务"
echo "=========================================="

# 进入后端目录
cd "$(dirname "$0")/../server"

# 检查是否已编译
if [ ! -f "./sisyphus" ]; then
    echo "📦 正在编译后端服务..."
    go build -o sisyphus ./cmd/sisyphus
    echo "✅ 编译完成"
else
    echo "✅ 已找到编译好的二进制文件"
fi

# 检查配置文件
if [ ! -f "./configs/config.yaml" ]; then
    echo "❌ 未找到配置文件: configs/config.yaml"
    exit 1
fi

# 检查MySQL连接
echo "🔍 检查MySQL连接..."
MYSQL_USER="root"
MYSQL_PASSWORD=""
MYSQL_HOST="127.0.0.1"
MYSQL_PORT="3306"

if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -e "SELECT 1" &>/dev/null; then
    echo "✅ MySQL 连接正常"
else
    echo "⚠️  MySQL 连接失败，请检查配置"
fi

# 检查Redis连接
echo "🔍 检查Redis连接..."
if redis-cli ping &>/dev/null; then
    echo "✅ Redis 连接正常"
else
    echo "⚠️  Redis 连接失败，请检查Redis是否已启动"
fi

# 启动服务
echo ""
echo "🚀 启动后端服务..."
echo "   HTTP服务: http://localhost:8000"
echo "   gRPC服务: localhost:9000"
echo ""
echo "按 Ctrl+C 停止服务"
echo ""

./sisyphus -conf configs

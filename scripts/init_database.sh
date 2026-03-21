#!/bin/bash

set -e

echo "=========================================="
echo "初始化 Sisyphus 数据库"
echo "=========================================="

# 提示用户输入MySQL密码
read -sp "请输入MySQL root密码（如果为空直接回车）: " MYSQL_PASSWORD
echo ""

MYSQL_USER="root"
MYSQL_HOST="127.0.0.1"
MYSQL_PORT="3306"

# 构建MySQL命令
if [ -z "$MYSQL_PASSWORD" ]; then
    MYSQL_CMD="mysql -h$MYSQL_HOST -P$MYSQL_PORT -u$MYSQL_USER"
else
    MYSQL_CMD="mysql -h$MYSQL_HOST -P$MYSQL_PORT -u$MYSQL_USER -p$MYSQL_PASSWORD"
fi

# 检查MySQL连接
echo "🔍 检查MySQL连接..."
if $MYSQL_CMD -e "SELECT 1" &>/dev/null; then
    echo "✅ MySQL 连接成功"
else
    echo "❌ MySQL 连接失败，请检查："
    echo "   1. MySQL是否已启动"
    echo "   2. root用户密码是否正确"
    exit 1
fi

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 执行初始化脚本
INIT_SQL="$PROJECT_ROOT/migrations/init.sql"
if [ -f "$INIT_SQL" ]; then
    echo "📦 正在初始化数据库..."
    $MYSQL_CMD < "$INIT_SQL"
    echo "✅ 数据库初始化完成"
else
    echo "❌ 未找到初始化脚本: $INIT_SQL"
    exit 1
fi

# 导入测试数据
TEST_DATA_SQL="$PROJECT_ROOT/migrations/test_data.sql"
if [ -f "$TEST_DATA_SQL" ]; then
    echo "📦 正在导入测试数据..."
    $MYSQL_CMD sisyphus < "$TEST_DATA_SQL"
    echo "✅ 测试数据导入完成"
else
    echo "⚠️  未找到测试数据脚本: $TEST_DATA_SQL"
fi

# 验证数据
echo ""
echo "🔍 验证数据..."
USER_COUNT=$($MYSQL_CMD sisyphus -N -e "SELECT COUNT(*) FROM sisyphus_users" 2>/dev/null)
INSTANCE_COUNT=$($MYSQL_CMD sisyphus -N -e "SELECT COUNT(*) FROM sisyphus_database_instances" 2>/dev/null)

echo "   用户数量: $USER_COUNT"
echo "   实例数量: $INSTANCE_COUNT"

echo ""
echo "=========================================="
echo "✅ 数据库初始化完成！"
echo "=========================================="
echo ""
echo "📝 如果MySQL密码不为空，请更新配置文件:"
echo "   sisyphus/server/configs/config.yaml"
echo "   修改 data.database.source 中的密码部分"
echo ""

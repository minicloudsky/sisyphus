#!/bin/bash

# 数据库初始化脚本
# 使用方法: ./init_db.sh [mysql_user] [mysql_password]

MYSQL_USER=${1:-root}
MYSQL_PASSWORD=${2:-root}
MYSQL_HOST=${3:-127.0.0.1}
MYSQL_PORT=${4:-3306}

echo "开始初始化数据库..."
echo "MySQL用户: $MYSQL_USER"
echo "MySQL主机: $MYSQL_HOST:$MYSQL_PORT"

# 执行初始化SQL
mysql -h$MYSQL_HOST -P$MYSQL_PORT -u$MYSQL_USER -p$MYSQL_PASSWORD < "$(dirname "$0")/../migrations/init.sql"

if [ $? -eq 0 ]; then
    echo "数据库初始化成功！"
    
    # 插入测试数据
    echo "插入测试数据..."
    mysql -h$MYSQL_HOST -P$MYSQL_PORT -u$MYSQL_USER -p$MYSQL_PASSWORD sisyphus < "$(dirname "$0")/../migrations/test_data.sql"
    
    if [ $? -eq 0 ]; then
        echo "测试数据插入成功！"
    else
        echo "测试数据插入失败，请检查 test_data.sql 文件"
    fi
else
    echo "数据库初始化失败，请检查MySQL连接和权限"
    exit 1
fi

#!/bin/bash

set -e

echo "=========================================="
echo "Sisyphus 本地环境安装脚本 (macOS)"
echo "=========================================="

# 检查是否安装了Homebrew
if ! command -v brew &> /dev/null; then
    echo "❌ 未检测到Homebrew，正在安装..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
    echo "✅ Homebrew 已安装"
fi

# 安装MySQL
echo ""
echo "📦 安装MySQL..."
if ! command -v mysql &> /dev/null; then
    brew install mysql
    brew services start mysql
    echo "✅ MySQL 已安装并启动"
    echo "⚠️  默认root密码为空，请运行以下命令设置密码："
    echo "   mysql_secure_installation"
else
    echo "✅ MySQL 已安装"
    if ! brew services list | grep -q "mysql.*started"; then
        brew services start mysql
        echo "✅ MySQL 已启动"
    else
        echo "✅ MySQL 正在运行"
    fi
fi

# 安装Redis
echo ""
echo "📦 安装Redis..."
if ! command -v redis-server &> /dev/null; then
    brew install redis
    brew services start redis
    echo "✅ Redis 已安装并启动"
else
    echo "✅ Redis 已安装"
    if ! brew services list | grep -q "redis.*started"; then
        brew services start redis
        echo "✅ Redis 已启动"
    else
        echo "✅ Redis 正在运行"
    fi
fi

# 安装gh-ost
echo ""
echo "📦 安装gh-ost..."
if ! command -v gh-ost &> /dev/null; then
    brew install gh-ost
    echo "✅ gh-ost 已安装"
else
    echo "✅ gh-ost 已安装"
fi

# 安装percona-toolkit (包含pt-archiver)
echo ""
echo "📦 安装percona-toolkit (包含pt-archiver)..."
if ! command -v pt-archiver &> /dev/null; then
    brew install percona-toolkit
    echo "✅ percona-toolkit 已安装"
else
    echo "✅ percona-toolkit 已安装"
fi

# 安装goInception
echo ""
echo "📦 安装goInception..."
GOINCEPTION_VERSION="v1.3.0"
GOINCEPTION_DIR="$HOME/goinception"
GOINCEPTION_BIN="$GOINCEPTION_DIR/goInception"

if [ ! -f "$GOINCEPTION_BIN" ]; then
    echo "正在下载goInception..."
    mkdir -p "$GOINCEPTION_DIR"
    cd "$GOINCEPTION_DIR"
    
    # 检测系统架构
    ARCH=$(uname -m)
    if [ "$ARCH" == "arm64" ]; then
        ARCH="arm64"
    else
        ARCH="amd64"
    fi
    
    # 下载对应架构的二进制文件
    DOWNLOAD_URL="https://github.com/hanchuanchuan/goInception/releases/download/${GOINCEPTION_VERSION}/goInception-${GOINCEPTION_VERSION}-darwin-${ARCH}.tar.gz"
    
    if curl -L -o goInception.tar.gz "$DOWNLOAD_URL" 2>/dev/null; then
        tar -xzf goInception.tar.gz
        rm goInception.tar.gz
        chmod +x goInception
        echo "✅ goInception 已下载到 $GOINCEPTION_DIR"
    else
        echo "⚠️  自动下载失败，请手动下载goInception："
        echo "   访问: https://github.com/hanchuanchuan/goInception/releases"
        echo "   下载对应版本的二进制文件到: $GOINCEPTION_DIR"
    fi
else
    echo "✅ goInception 已存在: $GOINCEPTION_BIN"
fi

# 创建必要的目录
echo ""
echo "📁 创建必要的目录..."
mkdir -p /data/ghost/socket
mkdir -p /data/ghost/log
mkdir -p /data/ghost/panic
echo "✅ 目录已创建"

# 初始化数据库
echo ""
echo "🗄️  初始化数据库..."
MYSQL_USER="root"
MYSQL_PASSWORD=""
MYSQL_HOST="127.0.0.1"
MYSQL_PORT="3306"

# 检查MySQL连接
if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -e "SELECT 1" &>/dev/null; then
    echo "✅ MySQL 连接成功"
    
    # 执行初始化脚本
    INIT_SQL="../migrations/init.sql"
    if [ -f "$INIT_SQL" ]; then
        mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" < "$INIT_SQL"
        echo "✅ 数据库初始化完成"
    else
        echo "⚠️  未找到初始化脚本: $INIT_SQL"
    fi
    
    # 导入测试数据
    TEST_DATA_SQL="../migrations/test_data.sql"
    if [ -f "$TEST_DATA_SQL" ]; then
        mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" sisyphus < "$TEST_DATA_SQL"
        echo "✅ 测试数据导入完成"
    else
        echo "⚠️  未找到测试数据脚本: $TEST_DATA_SQL"
    fi
else
    echo "⚠️  MySQL 连接失败，请检查："
    echo "   1. MySQL是否已启动"
    echo "   2. root用户密码是否正确"
    echo "   3. 可以手动运行: mysql -u root < ../migrations/init.sql"
fi

echo ""
echo "=========================================="
echo "✅ 安装完成！"
echo "=========================================="
echo ""
echo "📝 下一步："
echo "1. 如果MySQL root密码不为空，请更新配置文件:"
echo "   sisyphus/server/configs/config.yaml"
echo ""
echo "2. 如果goInception路径不同，请更新配置文件中的路径"
echo ""
echo "3. 启动goInception（如果需要）:"
echo "   $GOINCEPTION_BIN -config=goInception.conf"
echo ""
echo "4. 编译并启动后端服务:"
echo "   cd sisyphus/server"
echo "   go build -o sisyphus ./cmd/sisyphus"
echo "   ./sisyphus -conf configs"
echo ""

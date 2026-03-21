#!/bin/bash

# 前端开发服务器启动脚本

set -e

echo "=========================================="
echo "启动 Sisyphus 前端开发服务器"
echo "=========================================="

# 检查Node.js版本
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 16 ]; then
    echo "⚠️  Node.js版本过低（当前: $(node -v)），需要 >= 16"
    echo "💡 使用nvm切换版本："
    echo "   source ~/.nvm/nvm.sh && nvm install 18 && nvm use 18"
    exit 1
fi

echo "✅ Node.js版本: $(node -v)"
echo "✅ npm版本: $(npm -v)"
echo ""

# 检查依赖
if [ ! -d "node_modules" ]; then
    echo "📦 安装依赖..."
    npm install
fi

# 启动开发服务器
echo "🚀 启动开发服务器..."
echo ""
echo "💡 提示："
echo "   - 前端服务通常运行在 http://localhost:8000"
echo "   - 如果端口被占用，会自动使用其他端口"
echo "   - 确保后端服务已启动（http://localhost:8000/api）"
echo "   - 按 Ctrl+C 停止服务"
echo ""

npm run dev

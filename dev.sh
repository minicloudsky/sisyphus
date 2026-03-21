#!/usr/bin/env bash
# =============================================================================
# Sisyphus 一键本地开发环境启动脚本
# 用法: ./dev.sh [选项]
#   --backend-only   只启动后端
#   --frontend-only  只启动前端
#   --no-build       跳过重新编译后端
#   --stop           停止所有服务
# =============================================================================

set -euo pipefail

# ── 颜色 ─────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info()    { echo -e "${GREEN}✓${NC}  $*"; }
log_warn()    { echo -e "${YELLOW}⚠${NC}  $*"; }
log_error()   { echo -e "${RED}✗${NC}  $*"; }
log_step()    { echo -e "\n${BOLD}${BLUE}▶${NC}  ${BOLD}$*${NC}"; }
log_section() { echo -e "\n${CYAN}══════════════════════════════════════════${NC}"; echo -e "${CYAN}  $*${NC}"; echo -e "${CYAN}══════════════════════════════════════════${NC}"; }

# ── 路径 ─────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/server"
FRONTEND_DIR="$SCRIPT_DIR/web"
CONFIG_FILE="$BACKEND_DIR/configs/config.yaml"
LOCAL_CONFIG="$BACKEND_DIR/configs/config.local.yaml"
PID_DIR="/tmp/sisyphus"
BACKEND_PID="$PID_DIR/backend.pid"
FRONTEND_PID="$PID_DIR/frontend.pid"
BACKEND_LOG="$PID_DIR/backend.log"
FRONTEND_LOG="$PID_DIR/frontend.log"

# ── 参数解析 ──────────────────────────────────────────────────────────────────
OPT_BACKEND_ONLY=false
OPT_FRONTEND_ONLY=false
OPT_NO_BUILD=false
OPT_STOP=false

for arg in "$@"; do
  case $arg in
    --backend-only)  OPT_BACKEND_ONLY=true ;;
    --frontend-only) OPT_FRONTEND_ONLY=true ;;
    --no-build)      OPT_NO_BUILD=true ;;
    --stop)          OPT_STOP=true ;;
    -h|--help)
      echo "用法: $0 [--backend-only] [--frontend-only] [--no-build] [--stop]"
      exit 0 ;;
    *) echo "未知参数: $arg"; exit 1 ;;
  esac
done

mkdir -p "$PID_DIR"

# ── 停止服务 ──────────────────────────────────────────────────────────────────
stop_services() {
  log_section "停止 Sisyphus 服务"
  local stopped=0

  if [ -f "$BACKEND_PID" ]; then
    pid=$(cat "$BACKEND_PID")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" && log_info "后端服务已停止 (PID $pid)"
      stopped=$((stopped+1))
    fi
    rm -f "$BACKEND_PID"
  fi

  if [ -f "$FRONTEND_PID" ]; then
    pid=$(cat "$FRONTEND_PID")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      log_info "前端服务已停止 (PID $pid)"
      stopped=$((stopped+1))
    fi
    rm -f "$FRONTEND_PID"
  fi

  # 兜底：清理残留进程
  pkill -f "sisyphus.*-conf configs" 2>/dev/null && log_info "清理残留后端进程" || true
  pkill -f "umi dev" 2>/dev/null && log_info "清理残留前端进程" || true

  if [ $stopped -eq 0 ]; then
    log_warn "没有找到正在运行的 Sisyphus 服务"
  else
    log_info "已停止 $stopped 个服务"
  fi
  exit 0
}

if $OPT_STOP; then
  stop_services
fi

# ── 退出时清理 ────────────────────────────────────────────────────────────────
cleanup() {
  echo ""
  log_section "收到退出信号，停止所有服务..."

  if [ -f "$BACKEND_PID" ]; then
    pid=$(cat "$BACKEND_PID" 2>/dev/null || true)
    [ -n "$pid" ] && kill "$pid" 2>/dev/null && log_info "后端已停止" || true
    rm -f "$BACKEND_PID"
  fi

  if [ -f "$FRONTEND_PID" ]; then
    pid=$(cat "$FRONTEND_PID" 2>/dev/null || true)
    if [ -n "$pid" ]; then
      kill "$pid" 2>/dev/null || true
      log_info "前端已停止"
    fi
    rm -f "$FRONTEND_PID"
  fi

  log_info "服务已全部停止"
  exit 0
}
trap cleanup INT TERM

# ── 检查命令是否存在 ──────────────────────────────────────────────────────────
require_cmd() {
  if ! command -v "$1" &>/dev/null; then
    log_error "未找到命令: $1  →  $2"
    return 1
  fi
}

# ── 1. 环境检查 ───────────────────────────────────────────────────────────────
log_section "检查运行环境"

ENV_OK=true

if ! $OPT_FRONTEND_ONLY; then
  require_cmd go "请安装 Go: https://golang.org/dl/" || ENV_OK=false
  GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
  log_info "Go 版本: $GO_VERSION"
fi

if ! $OPT_BACKEND_ONLY; then
  require_cmd node "请安装 Node.js 18+: https://nodejs.org" || ENV_OK=false
  require_cmd npm "请安装 npm" || ENV_OK=false
  NODE_VERSION=$(node --version 2>/dev/null || echo "未知")
  log_info "Node 版本: $NODE_VERSION"
fi

require_cmd mysql "请安装 MySQL 8.0+" || ENV_OK=false
require_cmd redis-cli "请安装 Redis" || ENV_OK=false

$ENV_OK || { log_error "环境检查失败，请安装缺失的依赖"; exit 1; }

# ── 2. 启动基础服务（MySQL / Redis）──────────────────────────────────────────
log_section "检查基础服务"

# MySQL (optional: DEV_MYSQL_USER / DEV_MYSQL_PASSWORD for non-default local auth)
log_step "MySQL"
MYSQL_RUNNING=false
_mysql_ping() {
  mysql -h 127.0.0.1 -P 3306 --connect-timeout=3 "$@" -e "SELECT 1" &>/dev/null 2>&1
}
if _mysql_ping -u root; then
  MYSQL_RUNNING=true
elif [ -n "${DEV_MYSQL_PASSWORD:-}" ]; then
  u="${DEV_MYSQL_USER:-root}"
  if _mysql_ping -u "$u" -p"$DEV_MYSQL_PASSWORD"; then
    MYSQL_RUNNING=true
  fi
fi

if $MYSQL_RUNNING; then
  log_info "MySQL 已在运行"
else
  log_warn "MySQL 未运行，尝试启动..."
  if command -v brew &>/dev/null; then
    brew services start mysql 2>/dev/null || brew services start mysql@8.4 2>/dev/null || true
    sleep 3
    if _mysql_ping -u root || { [ -n "${DEV_MYSQL_PASSWORD:-}" ] && _mysql_ping -u "${DEV_MYSQL_USER:-root}" -p"$DEV_MYSQL_PASSWORD"; }; then
      log_info "MySQL 启动成功"
      MYSQL_RUNNING=true
    else
      log_warn "MySQL 启动失败（后端可能无法正常工作）"
      log_warn "手动启动: brew services start mysql"
    fi
  else
    log_warn "请手动启动 MySQL: sudo systemctl start mysql"
  fi
fi

# Redis
log_step "Redis"
if redis-cli -h 127.0.0.1 -p 6379 ping &>/dev/null 2>&1; then
  log_info "Redis 已在运行"
else
  log_warn "Redis 未运行，尝试启动..."
  if command -v brew &>/dev/null; then
    brew services start redis 2>/dev/null || true
    sleep 2
  else
    redis-server --daemonize yes 2>/dev/null || true
    sleep 1
  fi

  if redis-cli -h 127.0.0.1 -p 6379 ping &>/dev/null 2>&1; then
    log_info "Redis 启动成功"
  else
    log_warn "Redis 启动失败（后端可能无法正常工作）"
    log_warn "手动启动: brew services start redis"
  fi
fi

# ── 3. 确认配置文件 ───────────────────────────────────────────────────────────
if ! $OPT_FRONTEND_ONLY; then
  log_step "检查配置文件"
  # 优先使用 config.local.yaml（本地覆盖配置）
  if [ -f "$LOCAL_CONFIG" ]; then
    CONF_ARG="$BACKEND_DIR/configs/config.local.yaml"
    log_info "使用本地配置: configs/config.local.yaml"
  elif [ -f "$CONFIG_FILE" ]; then
    CONF_ARG="$BACKEND_DIR/configs"
    log_info "使用默认配置: configs/config.yaml"
  else
    log_error "找不到配置文件: $CONFIG_FILE"
    exit 1
  fi
fi

# ── 4. 编译后端 ───────────────────────────────────────────────────────────────
if ! $OPT_FRONTEND_ONLY; then
  log_step "编译后端"
  cd "$BACKEND_DIR"

  if $OPT_NO_BUILD && [ -f "./bin/sisyphus" ]; then
    log_info "跳过编译（使用已有二进制）"
  else
    echo "  正在编译..."
    if go build -o bin/sisyphus ./cmd/sisyphus 2>&1; then
      log_info "编译成功: server/bin/sisyphus"
    else
      log_error "编译失败，请检查代码"
      exit 1
    fi
  fi
fi

# ── 5. 安装前端依赖 ───────────────────────────────────────────────────────────
if ! $OPT_BACKEND_ONLY; then
  log_step "前端依赖"
  cd "$FRONTEND_DIR"

  if [ ! -d "node_modules" ]; then
    echo "  node_modules 不存在，安装依赖..."
    npm install --silent && log_info "依赖安装完成"
  else
    log_info "依赖已就绪"
  fi
fi

# ── 6. 启动服务 ───────────────────────────────────────────────────────────────
log_section "启动 Sisyphus 服务"

# 后端
if ! $OPT_FRONTEND_ONLY; then
  log_step "启动后端 (HTTP :8000 / gRPC :9000)"
  cd "$BACKEND_DIR"

  # 杀掉旧进程
  if [ -f "$BACKEND_PID" ]; then
    old=$(cat "$BACKEND_PID")
    kill "$old" 2>/dev/null || true
    sleep 1
  fi

  ./bin/sisyphus -conf "$CONF_ARG" >"$BACKEND_LOG" 2>&1 &
  echo $! > "$BACKEND_PID"
  BACKEND_PID_VAL=$!

  # 等待后端就绪（最多 15 秒）
  echo -n "  等待后端就绪"
  for i in $(seq 1 15); do
    if curl -sf http://localhost:8000/healthz &>/dev/null 2>&1 || \
       curl -sf http://localhost:8000/v1/health &>/dev/null 2>&1 || \
       lsof -i :8000 -sTCP:LISTEN &>/dev/null 2>&1; then
      echo ""
      log_info "后端已就绪 (PID $BACKEND_PID_VAL)"
      break
    fi
    sleep 1
    echo -n "."
    if [ $i -eq 15 ]; then
      echo ""
      log_warn "后端启动超时，查看日志: tail -f $BACKEND_LOG"
      tail -20 "$BACKEND_LOG" 2>/dev/null || true
    fi
  done
fi

# 前端
if ! $OPT_BACKEND_ONLY; then
  log_step "启动前端 (:8001)"
  cd "$FRONTEND_DIR"

  # 杀掉旧进程
  if [ -f "$FRONTEND_PID" ]; then
    old=$(cat "$FRONTEND_PID")
    kill "$old" 2>/dev/null || true
    sleep 1
  fi

  npm run dev >"$FRONTEND_LOG" 2>&1 &
  FE_PID=$!
  echo $FE_PID > "$FRONTEND_PID"

  # 等待前端就绪（最多 30 秒）
  echo -n "  等待前端就绪"
  for i in $(seq 1 30); do
    if curl -sf http://localhost:8001 &>/dev/null 2>&1 || \
       grep -q "Local:" "$FRONTEND_LOG" 2>/dev/null; then
      echo ""
      log_info "前端已就绪 (PID $FE_PID)"
      break
    fi
    sleep 1
    echo -n "."
    if [ $i -eq 30 ]; then
      echo ""
      log_warn "前端启动较慢，查看日志: tail -f $FRONTEND_LOG"
    fi
  done
fi

# ── 7. 启动摘要 ───────────────────────────────────────────────────────────────
log_section "Sisyphus 已启动 🚀"

if ! $OPT_FRONTEND_ONLY; then
  echo -e "  ${BOLD}后端 API${NC}   → http://localhost:8000"
  echo -e "  ${BOLD}gRPC${NC}       → localhost:9000"
fi
if ! $OPT_BACKEND_ONLY; then
  echo -e "  ${BOLD}前端页面${NC}   → http://localhost:8001"
fi

echo ""
echo -e "  ${BOLD}日志文件${NC}"
! $OPT_FRONTEND_ONLY && echo -e "    后端: tail -f $BACKEND_LOG"
! $OPT_BACKEND_ONLY  && echo -e "    前端: tail -f $FRONTEND_LOG"

echo ""
echo -e "  ${BOLD}Dev login${NC}    See QUICKSTART.md (seed users; change passwords in production)"
echo ""
echo -e "  ${YELLOW}按 Ctrl+C 停止所有服务${NC}"
echo -e "  ${YELLOW}或运行: $0 --stop${NC}"
echo ""

# ── 8. 等待 & 日志跟踪 ────────────────────────────────────────────────────────
# 如果两个服务都启动了，同时 tail 两个日志
if ! $OPT_BACKEND_ONLY && ! $OPT_FRONTEND_ONLY; then
  # 用 tail -f 同时监控两个日志直到收到信号
  tail -f "$BACKEND_LOG" "$FRONTEND_LOG" 2>/dev/null &
  TAIL_PID=$!
  wait $TAIL_PID
elif ! $OPT_FRONTEND_ONLY; then
  tail -f "$BACKEND_LOG" &
  TAIL_PID=$!
  wait $TAIL_PID
elif ! $OPT_BACKEND_ONLY; then
  tail -f "$FRONTEND_LOG" &
  TAIL_PID=$!
  wait $TAIL_PID
else
  wait
fi

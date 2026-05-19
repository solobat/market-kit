#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
用法:
  ./deploy/deploy_ubuntu_pm2.sh [options]

可选:
  --dir             项目目录，默认脚本所在仓库根目录
  --backend-port    后端监听端口，默认 18120
  --skip-install    跳过 apt / pm2 安装，只构建并启动

例子:
  ./deploy/deploy_ubuntu_pm2.sh
  ./deploy/deploy_ubuntu_pm2.sh --backend-port 18120
EOF
}

DIR=""
BACKEND_PORT="18120"
SKIP_INSTALL="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir) DIR="$2"; shift 2 ;;
    --backend-port) BACKEND_PORT="$2"; shift 2 ;;
    --skip-install) SKIP_INSTALL="1"; shift 1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "未知参数: $1" >&2; usage; exit 2 ;;
  esac
done

if ! [[ "$BACKEND_PORT" =~ ^[0-9]+$ ]]; then
  echo "--backend-port 必须是数字" >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIR="${DIR:-$DEFAULT_DIR}"
NODE_UPGRADED="0"

ensure_go_in_path() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi

  local candidates=(
    /usr/local/go/bin/go
    /usr/local/bin/go
    /snap/bin/go
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    if [[ -x "$candidate" ]]; then
      export PATH="$(dirname "$candidate"):$PATH"
      return 0
    fi
  done

  return 1
}

node_major_version() {
  if ! command -v node >/dev/null 2>&1; then
    echo "0"
    return
  fi
  node -p "Number(process.versions.node.split('.')[0])"
}

install_node_runtime_if_needed() {
  local major
  major="$(node_major_version)"
  if [[ "$major" -ge 20 ]]; then
    return 0
  fi

  if [[ "$major" -eq 0 ]]; then
    echo "未检测到 Node.js，将安装 Node.js 22"
  else
    echo "当前 Node.js 版本过低: $(node -v)，将升级到 Node.js 22"
  fi

  curl -fsSL https://deb.nodesource.com/setup_22.x -o /tmp/nodesource_setup.sh
  sudo -E bash /tmp/nodesource_setup.sh
  sudo apt install -y nodejs
  hash -r
  NODE_UPGRADED="1"
}

echo "[1/5] 安装基础依赖"
if [[ "$SKIP_INSTALL" != "1" ]]; then
  sudo apt update
  sudo apt install -y git curl build-essential

  install_node_runtime_if_needed

  if [[ "$NODE_UPGRADED" == "1" ]]; then
    sudo npm i -g pm2 pnpm
  elif ! command -v pm2 >/dev/null 2>&1; then
    command -v npm >/dev/null 2>&1 || { echo "未检测到 npm，请先安装 Node.js 20+ / npm" >&2; exit 1; }
    sudo npm i -g pm2
  fi
  if [[ "$NODE_UPGRADED" != "1" ]] && ! command -v pnpm >/dev/null 2>&1; then
    command -v npm >/dev/null 2>&1 || { echo "未检测到 npm，请先安装 Node.js 20+ / npm" >&2; exit 1; }
    sudo npm i -g pnpm
  fi
else
  echo "已跳过系统依赖安装"
fi

echo "[2/5] 检查 Go / pnpm / PM2"
if ! ensure_go_in_path; then
  echo "未检测到 go。请先安装 Go。" >&2
  exit 1
fi
if ! command -v node >/dev/null 2>&1; then
  echo "未检测到 node。请先安装 Node.js 20+。" >&2
  exit 1
fi
NODE_MAJOR="$(node -p "Number(process.versions.node.split('.')[0])")"
if [[ "$NODE_MAJOR" -lt 20 ]]; then
  echo "当前 Node.js 版本过低: $(node -v)。请先安装 Node.js 20+。" >&2
  exit 1
fi
command -v pnpm >/dev/null 2>&1 || { echo "未检测到 pnpm" >&2; exit 1; }
command -v pm2 >/dev/null 2>&1 || { echo "未检测到 pm2" >&2; exit 1; }

echo "[3/5] 进入项目目录 $DIR"
cd "$DIR"
if [[ ! -f go.mod || ! -f cmd/market-kit-server/main.go ]]; then
  echo "未在 $DIR 找到 market-kit 仓库文件，请确认 --dir" >&2
  exit 1
fi

if [[ ! -f .env ]]; then
  echo "未找到 .env，将使用内置默认值。建议从 .env.production.example 复制一份再调整。"
fi

echo "[4/5] 启动 / 重启 PM2"
chmod +x ./pm2-start.sh ./pm2-restart.sh ./pm2-stop.sh ./pm2-status.sh ./pm2-logs.sh ./pm2-save.sh
export MARKET_KIT_HTTP_ADDR="127.0.0.1:${BACKEND_PORT}"
if pm2 describe market-kit >/dev/null 2>&1; then
  ./pm2-restart.sh
else
  ./pm2-start.sh
fi

echo "[5/5] 健康检查"
curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/healthz"
echo ""
echo "完成。常用命令:"
echo "  ./pm2-status.sh"
echo "  ./pm2-logs.sh lines 100"
echo "  ./pm2-save.sh"

#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

load_env_file_if_present() {
  local env_file="$1"
  if [ -f "$env_file" ]; then
    set -a
    # shellcheck disable=SC1090
    . "$env_file"
    set +a
    echo "已加载环境变量: $env_file"
  fi
}

ensure_go_in_path() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi

  local candidates=(
    /usr/local/go/bin/go
    /usr/local/bin/go
    /opt/homebrew/bin/go
    /snap/bin/go
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    if [ -x "$candidate" ]; then
      export PATH="$(dirname "$candidate"):$PATH"
      return 0
    fi
  done

  return 1
}

echo "启动 market-kit..."
echo ""

if ! command -v pm2 >/dev/null 2>&1; then
  echo "PM2 未安装: npm install -g pm2"
  exit 1
fi

if ! ensure_go_in_path; then
  echo "未检测到 go，请先安装 Go"
  exit 1
fi

load_env_file_if_present "$SCRIPT_DIR/.env"

mkdir -p bin logs

BUILD_VERSION="${MARKET_KIT_BUILD_VERSION:-local}"
BUILD_COMMIT="${MARKET_KIT_BUILD_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_TIME="${MARKET_KIT_BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

echo "构建后端..."
go build \
  -ldflags "-X github.com/solobat/market-kit/server.BuildVersion=${BUILD_VERSION} -X github.com/solobat/market-kit/server.BuildCommit=${BUILD_COMMIT} -X github.com/solobat/market-kit/server.BuildTime=${BUILD_TIME}" \
  -o "$SCRIPT_DIR/bin/market-kit-server" \
  ./cmd/market-kit-server
echo "后端构建完成"
echo ""

echo "启动 PM2..."
pm2 start ecosystem.config.cjs --only market-kit --update-env

echo ""
echo "已启动"
echo "  ./pm2-status.sh   查看状态"
echo "  ./pm2-logs.sh     查看日志"
echo "  ./pm2-restart.sh  重启服务"

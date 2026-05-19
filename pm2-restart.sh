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

ensure_node_runtime() {
  if ! command -v node >/dev/null 2>&1; then
    echo "未检测到 node，请先安装 Node.js 20+"
    exit 1
  fi

  local major
  major="$(node -p "Number(process.versions.node.split('.')[0])")"
  if [ "$major" -lt 20 ]; then
    echo "当前 Node.js 版本过低: $(node -v)，请安装 Node.js 20+"
    exit 1
  fi
}

if ! command -v pm2 >/dev/null 2>&1; then
  echo "PM2 未安装: npm install -g pm2"
  exit 1
fi

if ! ensure_go_in_path; then
  echo "未检测到 go，请先安装 Go"
  exit 1
fi

ensure_node_runtime

if ! command -v pnpm >/dev/null 2>&1; then
  echo "未检测到 pnpm，请先安装 pnpm 或启用 corepack"
  exit 1
fi

load_env_file_if_present "$SCRIPT_DIR/.env"

mkdir -p bin logs

echo "构建前端..."
(
  cd "$SCRIPT_DIR/frontend"
  pnpm install --frozen-lockfile
  pnpm build
)
echo "前端构建完成"

BUILD_VERSION="${MARKET_KIT_BUILD_VERSION:-local}"
BUILD_COMMIT="${MARKET_KIT_BUILD_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_TIME="${MARKET_KIT_BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

echo "构建后端..."
go build \
  -ldflags "-X github.com/solobat/market-kit/server.BuildVersion=${BUILD_VERSION} -X github.com/solobat/market-kit/server.BuildCommit=${BUILD_COMMIT} -X github.com/solobat/market-kit/server.BuildTime=${BUILD_TIME}" \
  -o "$SCRIPT_DIR/bin/market-kit-server" \
  ./cmd/market-kit-server
echo "后端构建完成"

if ! pm2 describe market-kit >/dev/null 2>&1; then
  echo "未发现 market-kit 进程，转为启动..."
  exec ./pm2-start.sh
fi

echo "重启 PM2 进程..."
pm2 restart ecosystem.config.cjs --only market-kit --update-env
echo "已重启"

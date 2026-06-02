#!/usr/bin/env bash
# Usage: bash scripts/setup-nginx-market-kit-api.sh
# Installs the /market-kit-api path-prefix nginx snippet for a shared API domain.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SNIPPET_NAME="market-kit-api.conf"
SNIPPET_SRC="${SCRIPT_DIR}/nginx-market-kit-api.conf"
NGINX_SNIPPETS="/etc/nginx/snippets"
if [ ! -d "$NGINX_SNIPPETS" ]; then
  NGINX_SNIPPETS="/etc/nginx"
fi
SNIPPET_DEST="${NGINX_SNIPPETS}/${SNIPPET_NAME}"

echo "=== 1. Write nginx snippet ==="
if [ ! -f "$SNIPPET_SRC" ]; then
  echo "Error: cannot find $SNIPPET_SRC"
  exit 1
fi

if [ -w "$(dirname "$SNIPPET_DEST")" ]; then
  mkdir -p "$(dirname "$SNIPPET_DEST")"
  cp "$SNIPPET_SRC" "$SNIPPET_DEST"
  echo "Written: $SNIPPET_DEST"
else
  echo "sudo is required to write the nginx snippet:"
  echo "  sudo mkdir -p $(dirname "$SNIPPET_DEST")"
  echo "  sudo cp $SNIPPET_SRC $SNIPPET_DEST"
  sudo mkdir -p "$(dirname "$SNIPPET_DEST")"
  sudo cp "$SNIPPET_SRC" "$SNIPPET_DEST"
  echo "Written: $SNIPPET_DEST"
fi

echo ""
echo "=== 2. Add include to the target server ==="
echo "Add this line inside your server { ... } block:"
echo ""
echo "    include $SNIPPET_DEST;"
echo ""
echo "This proxies /market-kit-api and /market-kit-api/* to http://127.0.0.1:18120."
echo "It is intended for an existing shared API domain such as api.immortal.app."
echo ""
echo "=== 3. Validate and reload nginx ==="
echo "Run:"
echo "  sudo nginx -t && sudo nginx -s reload"
echo ""
if command -v nginx >/dev/null 2>&1; then
  if nginx -t 2>/dev/null; then
    echo "Current nginx syntax is valid. After adding the include, run: sudo nginx -s reload"
  else
    echo "Please fix nginx config first, then run: sudo nginx -t && sudo nginx -s reload"
  fi
else
  echo "nginx is not installed or not visible in PATH. Validate manually after install."
fi

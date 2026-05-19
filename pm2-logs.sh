#!/bin/bash

set -e

if ! command -v pm2 >/dev/null 2>&1; then
  echo "PM2 未安装: npm install -g pm2"
  exit 1
fi

case "${1:-all}" in
  all)
    pm2 logs market-kit
    ;;
  lines)
    pm2 logs market-kit --lines "${2:-100}"
    ;;
  err)
    pm2 logs market-kit --err
    ;;
  out)
    pm2 logs market-kit --out
    ;;
  *)
    echo "用法:"
    echo "  ./pm2-logs.sh"
    echo "  ./pm2-logs.sh lines 100"
    echo "  ./pm2-logs.sh err"
    echo "  ./pm2-logs.sh out"
    exit 2
    ;;
esac

#!/bin/bash

set -e

if ! command -v pm2 >/dev/null 2>&1; then
  echo "PM2 未安装: npm install -g pm2"
  exit 1
fi

pm2 save

echo "已保存 PM2 进程列表。首次配置开机自启时继续执行:"
echo "  pm2 startup"

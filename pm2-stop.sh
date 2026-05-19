#!/bin/bash

set -e

if ! command -v pm2 >/dev/null 2>&1; then
  echo "PM2 未安装: npm install -g pm2"
  exit 1
fi

pm2 stop market-kit

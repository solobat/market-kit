const path = require('path')

const root = __dirname

module.exports = {
  apps: [
    {
      name: 'market-kit',
      cwd: root,
      script: path.join(root, 'bin/market-kit-server'),
      interpreter: 'none',
      autorestart: true,
      restart_delay: 5000,
      watch: false,
      time: true,
      out_file: path.join(root, 'logs/market-kit.out.log'),
      error_file: path.join(root, 'logs/market-kit.err.log'),
      env: {
        MARKET_KIT_HTTP_ADDR: process.env.MARKET_KIT_HTTP_ADDR || '127.0.0.1:18120',
        MARKET_KIT_FRONTEND_DIST: process.env.MARKET_KIT_FRONTEND_DIST || path.join(root, 'frontend/dist'),
        MARKET_KIT_REQUEST_TIMEOUT: process.env.MARKET_KIT_REQUEST_TIMEOUT || '12s',
        MARKET_KIT_SYNC_SOURCES_PATH: process.env.MARKET_KIT_SYNC_SOURCES_PATH || '',
      },
    },
  ],
}

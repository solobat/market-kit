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
        MARKET_KIT_DISCOVERY_CACHE_TTL: process.env.MARKET_KIT_DISCOVERY_CACHE_TTL || '5m',
        MARKET_KIT_ADMIN_CODE: process.env.MARKET_KIT_ADMIN_CODE || '',
        MARKET_KIT_SYNC_SOURCES_PATH: process.env.MARKET_KIT_SYNC_SOURCES_PATH || '',
        MARKET_KIT_SLIPSTREAM_DISCOVERY_URL: process.env.MARKET_KIT_SLIPSTREAM_DISCOVERY_URL || '',
        MARKET_KIT_SLIPSTREAM_ADMIN_CODE: process.env.MARKET_KIT_SLIPSTREAM_ADMIN_CODE || '',
        MARKET_KIT_AUTOSYNC_ENABLED: process.env.MARKET_KIT_AUTOSYNC_ENABLED || 'true',
        MARKET_KIT_AUTOSYNC_INTERVAL: process.env.MARKET_KIT_AUTOSYNC_INTERVAL || '5m',
        MARKET_KIT_AUTOSYNC_SOURCE: process.env.MARKET_KIT_AUTOSYNC_SOURCE || '',
        MARKET_KIT_RUNTIME_REGISTRY_PATH: process.env.MARKET_KIT_RUNTIME_REGISTRY_PATH || path.join(root, 'data/runtime_generated_registry.json'),
      },
    },
  ],
}

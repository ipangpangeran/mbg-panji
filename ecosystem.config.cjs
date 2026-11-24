// PM2 Configuration for Panji MBG (Native Go Binary — Ultra Lightweight)
// Usage: pm2 startOrReload ecosystem.config.cjs --env production
module.exports = {
  apps: [
    {
      name: 'panji-mbg',
      script: './panji-mbg',
      cwd: __dirname,
      interpreter: 'none', // Binary Go langsung dieksekusi tanpa Node interpreter
      instances: 1,        // Wajib 1 instance karena SQLite single-writer
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '150M',
      env: {
        PORT: '8080',
        DB_PATH: './data/app.db',
        STATIC_DIR: '.',
        TZ: 'Asia/Jakarta',
        GIN_MODE: 'release',
        ADMIN_PIN: '123456', // Hanya terpakai jika data/app.db belum ada
      },
      env_production: {
        PORT: '8080',
        DB_PATH: './data/app.db',
        STATIC_DIR: '.',
        TZ: 'Asia/Jakarta',
        GIN_MODE: 'release',
      },
    },
  ],
};

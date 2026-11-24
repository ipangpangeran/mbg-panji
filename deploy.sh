#!/usr/bin/env bash
# ==============================================================================
# Panji MBG — One-Command Production Deploy Script
# ------------------------------------------------------------------------------
# Cara pakai dari direktori mana saja di server Ubuntu:
#   /var/www/panji-mbg/deploy.sh
#
# Atau pasang shortcut global sekali saja:
#   sudo ln -sf /var/www/panji-mbg/deploy.sh /usr/local/bin/deploy-mbg
#   deploy-mbg
# ==============================================================================

set -Eeuo pipefail

# Otomatis mendeteksi folder project (mengikuti lokasi asli script walau dipanggil via symlink).
# Bisa juga di-override manual: APP_DIR=/custom/path ./deploy.sh
SCRIPT_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
DEFAULT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
APP_DIR="${APP_DIR:-$DEFAULT_DIR}"
BRANCH="${BRANCH:-main}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/healthz}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

info() { echo -e "${CYAN}==>${NC} $*"; }
ok()   { echo -e "${GREEN}✔${NC} $*"; }
warn() { echo -e "${YELLOW}⚠${NC} $*"; }
err()  { echo -e "${RED}✖${NC} $*" >&2; exit 1; }

echo -e "${GREEN}======================================================${NC}"
echo -e "${GREEN}   🚀 Deploying Panji MBG (${BRANCH})${NC}"
echo -e "${GREEN}======================================================${NC}"

# 1. Masuk ke folder aplikasi
[[ -d "$APP_DIR" ]] || err "Folder aplikasi tidak ditemukan: $APP_DIR"
cd "$APP_DIR"
info "Working directory: $(pwd)"

# Pastikan folder data persisten tersedia
mkdir -p data backups

# 2. Backup database otomatis sebelum deploy (jika sqlite3 & app.db tersedia)
if [[ -f "data/app.db" ]]; then
  TS="$(date +%Y%m%d-%H%M%S)"
  if command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 "data/app.db" ".backup 'backups/pre-deploy-${TS}.db'"
    ok "Database di-backup ke backups/pre-deploy-${TS}.db"
    # Pertahankan maksimal 10 file pre-deploy backup terakhir agar disk tetap lega
    ls -1t backups/pre-deploy-*.db 2>/dev/null | tail -n +11 | xargs -r rm -f
  else
    cp -f "data/app.db" "backups/pre-deploy-${TS}.db"
    warn "sqlite3 belum terinstal; menggunakan copy file biasa ke backups/pre-deploy-${TS}.db"
  fi
fi

# 3. Simpan perubahan lokal (misal ADMIN_PIN atau custom port di docker-compose.yml) agar git pull tidak gagal
STASHED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  warn "Terdeteksi perubahan lokal (mis. docker-compose.yml), menyimpan sementara via git stash..."
  git stash push -m "auto-stash-before-deploy-$(date +%s)" >/dev/null
  STASHED=1
fi

# 4. Pull kode terbaru dari GitHub
OLD_COMMIT="$(git rev-parse --short HEAD)"
info "Menarik update terbaru dari origin/${BRANCH} (commit saat ini: ${OLD_COMMIT})..."
git fetch origin "$BRANCH"
git pull --ff-only origin "$BRANCH"
NEW_COMMIT="$(git rev-parse --short HEAD)"

# Kembalikan konfigurasi lokal bila tadi di-stash
if [[ "$STASHED" -eq 1 ]]; then
  info "Mengembalikan konfigurasi lokal (git stash pop)..."
  git stash pop >/dev/null || warn "Konflik kecil saat stash pop — cek 'git status' setelah deploy."
fi

chmod +x deploy.sh 2>/dev/null || true

if [[ "$OLD_COMMIT" == "$NEW_COMMIT" ]]; then
  info "Kode sudah di commit terbaru (${NEW_COMMIT}), tetap memastikan container up-to-date..."
else
  ok "Berhasil update kode: ${OLD_COMMIT} -> ${NEW_COMMIT}"
  git log --oneline "${OLD_COMMIT}..${NEW_COMMIT}"
fi

# 5. Build & restart container via Docker Compose (mendukung v2 'docker compose' maupun v1 'docker-compose')
if docker compose version >/dev/null 2>&1; then
  DC=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  DC=(docker-compose)
else
  err "Docker Compose tidak ditemukan di server ini!"
fi

info "Building & restarting container dengan '${DC[*]} up -d --build'..."
"${DC[@]}" up -d --build

# 6. Bersihkan dangling images bekas build sebelumnya
info "Membersihkan image Docker lama yang tidak terpakai..."
docker image prune -f >/dev/null

# 7. Verifikasi Healthcheck aplikasi
info "Mengecek healthcheck aplikasi (${HEALTH_URL})..."
READY=0
for i in {1..15}; do
  if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 1
done

echo ""
if [[ "$READY" -eq 1 ]]; then
  ok "DEPLOY BERHASIL! Panji MBG aktif di commit ${NEW_COMMIT} 🎉"
  "${DC[@]}" ps
else
  warn "Container berjalan namun ${HEALTH_URL} belum merespons (cek port jika Anda mengubah port host)."
  "${DC[@]}" ps
  "${DC[@]}" logs --tail=25
  exit 1
fi

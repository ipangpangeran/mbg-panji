#!/usr/bin/env bash
# ==============================================================================
# Panji MBG — Fast Native Go + PM2 Deploy Script (Zero Docker Overhead)
# ------------------------------------------------------------------------------
# Keunggulan dibanding Docker:
# 1. Cache kompilasi Go (~/.cache/go-build) tersimpan permanen di host:
#    - Build pertama: ~30-45 detik (CPU dibatasi nice + GOMAXPROCS=1 agar aman)
#    - Deploy ke-2 dst: HANYA ~1-2 DETIK!
# 2. Jika hanya index.html / logo.jpg / qris.jpg yang berubah -> SKIP compile
#    sama sekali (0 detik, langsung aktif tanpa perlu restart!).
# 3. Dikelola oleh PM2 bersama aplikasi lain di server (RAM ~12 MB).
# ==============================================================================

set -Eeuo pipefail

# Pastikan PATH menyertakan lokasi Go & PM2 standar Ubuntu
export PATH="/usr/local/go/bin:$HOME/go/bin:/usr/local/bin:$PATH"

SCRIPT_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
DEFAULT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
APP_DIR="${APP_DIR:-$DEFAULT_DIR}"
BRANCH="${BRANCH:-main}"
APP_NAME="${APP_NAME:-panji-mbg}"
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
echo -e "${GREEN}   ⚡ Deploying Panji MBG — Native Go + PM2 (${BRANCH})${NC}"
echo -e "${GREEN}======================================================${NC}"

# 0. Cek Go & PM2 (auto-install Go 1.22 resmi jika belum ada di Ubuntu)
if ! command -v go >/dev/null 2>&1; then
  warn "Go belum terinstal di host. Mengunduh Go 1.22.10 resmi (sekali saja)..."
  ARCH="$(dpkg --print-architecture 2>/dev/null || uname -m)"
  case "$ARCH" in
    amd64|x86_64) GO_ARCH="amd64" ;;
    arm64|aarch64) GO_ARCH="arm64" ;;
    *) err "Arsitektur $ARCH tidak dikenali. Instal Go manual terlebih dahulu." ;;
  esac
  TMP_TAR="/tmp/go1.22.10.linux-${GO_ARCH}.tar.gz"
  curl -fsSL "https://go.dev/dl/go1.22.10.linux-${GO_ARCH}.tar.gz" -o "$TMP_TAR"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "$TMP_TAR"
  sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
  rm -f "$TMP_TAR"
  ok "Go berhasil diinstal: $(go version)"
fi

command -v pm2 >/dev/null 2>&1 || err "PM2 tidak ditemukan! Pastikan menjalankan dengan user yang memiliki PM2."

# 1. Masuk ke folder aplikasi
[[ -d "$APP_DIR" ]] || err "Folder aplikasi tidak ditemukan: $APP_DIR"
cd "$APP_DIR"
info "Working directory: $(pwd)"
mkdir -p data backups

# 2. Matikan & bersihkan container Docker lama jika sebelumnya pernah jalan via Docker
if command -v docker >/dev/null 2>&1 && docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "panji-mbg"; then
  warn "Container Docker lama 'panji-mbg' terdeteksi — menghentikan agar port 8080 bebas untuk PM2..."
  docker rm -f panji-mbg >/dev/null 2>&1 || true
  # Pastikan file data/app.db bekas Docker dimiliki oleh user saat ini (bukan root)
  if [[ ! -w "data" ]] || { [[ -f "data/app.db" ]] && [[ ! -w "data/app.db" ]]; }; then
    sudo chown -R "$(id -u):$(id -g)" data backups
  fi
  ok "Migrasi dari Docker ke Native PM2 selesai (database data/app.db tetap aman)."
fi

# 3. Backup database otomatis sebelum update
if [[ -f "data/app.db" ]]; then
  TS="$(date +%Y%m%d-%H%M%S)"
  if command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 "data/app.db" ".backup 'backups/pre-deploy-${TS}.db'"
  else
    cp -f "data/app.db" "backups/pre-deploy-${TS}.db"
    [[ -f "data/app.db-wal" ]] && cp -f "data/app.db-wal" "backups/pre-deploy-${TS}.db-wal"
  fi
  ls -1t backups/pre-deploy-*.db 2>/dev/null | tail -n +11 | xargs -r rm -f
  ok "Snapshot database tersimpan di backups/pre-deploy-${TS}.db"
fi

# 4. Stash perubahan lokal (mis. ecosystem.config.cjs) bila ada, lalu git pull
STASHED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  git stash push -m "auto-stash-before-deploy-$(date +%s)" >/dev/null
  STASHED=1
fi

OLD_COMMIT="$(git rev-parse --short HEAD)"
git fetch origin "$BRANCH"
git pull --ff-only origin "$BRANCH"
NEW_COMMIT="$(git rev-parse --short HEAD)"

if [[ "$STASHED" -eq 1 ]]; then
  git stash pop >/dev/null || warn "Konflik kecil saat stash pop — cek 'git status'."
fi
chmod +x deploy.sh 2>/dev/null || true

# 5. Cek apakah perlu compile ulang Go atau cukup update file statis saja
NEED_BUILD=0
if [[ ! -x "./panji-mbg" ]]; then
  NEED_BUILD=1
elif [[ "$OLD_COMMIT" != "$NEW_COMMIT" ]]; then
  ok "Update commit: ${OLD_COMMIT} -> ${NEW_COMMIT}"
  git log --oneline "${OLD_COMMIT}..${NEW_COMMIT}"
  if git diff --name-only "$OLD_COMMIT" "$NEW_COMMIT" | grep -qE '(\.go$|^go\.(mod|sum)$)'; then
    NEED_BUILD=1
  fi
fi

if [[ "$NEED_BUILD" -eq 1 ]]; then
  info "Meng-compile binary Go (memakai cache host + low CPU priority agar server tetap enteng)..."
  START_T=$SECONDS
  # nice -n 19 + GOMAXPROCS=1 menjaga agar aplikasi lain (Deflow/WAHA) tidak rebutan CPU
  GOTOOLCHAIN=local CGO_ENABLED=0 GOMAXPROCS=1 nice -n 19 \
    go build -trimpath -ldflags="-s -w" -o panji-mbg.new .
  mv -f panji-mbg.new panji-mbg
  ok "Compile selesai dalam $((SECONDS - START_T)) detik!"
else
  ok "Tidak ada perubahan kode Go (hanya file statis/konfigurasi) — skip compile (0 detik)!"
fi

# 6. Start atau Restart proses di PM2
info "Memuat proses '${APP_NAME}' di PM2..."
if pm2 describe "$APP_NAME" >/dev/null 2>&1; then
  if [[ "$NEED_BUILD" -eq 1 ]]; then
    pm2 restart "$APP_NAME" --update-env
  else
    ok "Binary tidak berubah & index.html dibaca langsung dari disk — tidak perlu restart!"
  fi
else
  pm2 start ecosystem.config.cjs --env production
fi
pm2 save >/dev/null 2>&1 || true

# 7. Smoke test /healthz
info "Mengecek healthcheck (${HEALTH_URL})..."
READY=0
for i in {1..15}; do
  if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 0.5
done

echo ""
if [[ "$READY" -eq 1 ]]; then
  ok "DEPLOY SELESAI! Panji MBG aktif di PM2 (commit ${NEW_COMMIT}) 🚀"
  pm2 status "$APP_NAME"
else
  warn "Endpoint ${HEALTH_URL} belum merespons. Cek log PM2:"
  pm2 logs "$APP_NAME" --lines 25 --nostream
  exit 1
fi

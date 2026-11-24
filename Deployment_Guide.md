# Panduan Deployment Production — Panji MBG (Ubuntu + Docker + Nginx + HTTPS)

Panduan *step-by-step* yang sudah disesuaikan sepenuhnya dengan arsitektur repository **`https://github.com/ipangpangeran/mbg-panji.git`** dan domain **`panji.ipangpangeran.com`**.

---

## 1. Prasyarat Server (Ubuntu 22.04 / 24.04 LTS)

Pastikan DNS domain `panji.ipangpangeran.com` (A Record) sudah diarahkan ke IP publik VPS kamu.

Jika server masih baru, jalankan instalasi **Docker Engine + Docker Compose v2 + Nginx + Git + SQLite3**:

```bash
sudo apt update && sudo apt install -y ca-certificates curl gnupg git nginx sqlite3

# Tambah GPG key & repository resmi Docker
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker & Compose plugin (perintah: docker compose)
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Izinkan user menjalankan docker tanpa sudo (logout & login kembali setelah ini)
sudo usermod -aG docker $USER
```

Verifikasi instalasi:
```bash
docker --version
docker compose version
nginx -v
```

---

## 2. Clone Repository Aplikasi

Tidak perlu membuat ulang `Dockerfile` atau `docker-compose.yml` secara manual karena semuanya sudah tersedia dan terkonfigurasi di repository:

```bash
sudo mkdir -p /var/www/panji-mbg
sudo chown $USER:$USER /var/www/panji-mbg
cd /var/www/panji-mbg

# Clone dari GitHub
git clone https://github.com/ipangpangeran/mbg-panji.git .

# Buat folder penyimpanan persisten untuk database SQLite (app.db)
mkdir -p data
```

> **Catatan Arsitektur Penting:**
> - Container berjalan di `WORKDIR /app` dengan `DB_PATH=/app/data/app.db` yang terhubung ke volume host `./data:/app/data`, sehingga data pesanan & konfigurasi **tidak akan hilang** saat container di-*rebuild*.
> - File statis `index.html`, `logo.jpg`, dan `qris.jpg` otomatis ikut disalin ke `/app/` saat proses *Docker build*.
> - Port container di-bind ke `127.0.0.1:8080:8080` supaya port `8080` aman dari akses langsung internet (hanya bisa diakses lewat Nginx).

---

## 3. Set PIN Admin Awal

Sebelum pertama kali menyalakan container, buka `docker-compose.yml` untuk mengganti PIN 6-digit default (`123456`):

```bash
nano docker-compose.yml
```

Ubah nilai `ADMIN_PIN`:
```yaml
    environment:
      - TZ=Asia/Jakarta
      - ADMIN_PIN=889900   # Ganti dengan 6 digit angka rahasia milikmu!
```
*(Variabel `ADMIN_PIN` ini hanya dipakai saat file `data/app.db` pertama kali dibuat. Selanjutnya penggantian PIN bisa dilakukan langsung dari Dashboard Admin).*

---

## 4. Build & Jalankan Container

```bash
cd /var/www/panji-mbg
docker compose up -d --build
```

Pastikan container berjalan sehat (`healthy`) dan endpoint internal merespons:
```bash
docker compose ps
docker compose logs -f --tail=30   # Tekan Ctrl+C untuk keluar dari log
curl http://127.0.0.1:8080/healthz # Harus mengembalikan: {"ok":true}
```

---

## 5. Konfigurasi Nginx Reverse Proxy

Buat konfigurasi Virtual Host Nginx:
```bash
sudo nano /etc/nginx/sites-available/panji-mbg
```

Tempelkan konfigurasi berikut:
```nginx
server {
    listen 80;
    server_name panji.ipangpangeran.com; # Sesuaikan jika memakai domain/subdomain lain

    # Wajib agar upload gambar QRIS baru dari Admin Dashboard tidak kena error 413
    client_max_body_size 4m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }
}
```

Aktifkan konfigurasi dan reload Nginx:
```bash
sudo ln -sf /etc/nginx/sites-available/panji-mbg /etc/nginx/sites-enabled/panji-mbg
sudo nginx -t
sudo systemctl reload nginx
```

---

## 6. Pasang SSL / HTTPS Gratis (Wajib untuk Production)

HTTPS sangat penting agar:
1. Header `X-Admin-Pin` saat admin mengelola warung terenkripsi penuh.
2. Fitur **Salin Rekap WA** (Clipboard API) diizinkan berjalan oleh browser HP modern (Chrome/Safari mewajibkan *HTTPS secure context*).

Jalankan Certbot:
```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d panji.ipangpangeran.com
```
*(Pilih opsi redirect HTTP ke HTTPS jika ditanya oleh Certbot — sertifikat akan diperpanjang otomatis).*

---

## 7. Konfigurasi Firewall (UFW)

```bash
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw status
# Jika UFW belum aktif, jalankan: sudo ufw enable
```

---

## 8. Checklist Pengujian Production

1. **Halaman Pelanggan:** Buka `https://panji.ipangpangeran.com`
   - Cek badge status `Warung Buka` dan sisa kuota.
   - Pilih menu utama (+ qty), add-ons (+ qty), opsi custom, dan klik metode **QRIS** (pastikan gambar QRIS muncul).
   - Kirim 1 pesanan percobaan.
2. **Halaman Admin:** Buka `https://panji.ipangpangeran.com/#admin`
   - Login menggunakan 6-digit PIN kamu.
   - Pastikan pesanan percobaan muncul di tabel *Live*.
   - Uji tombol **Salin Rekap WA**, **Export Excel**, lalu klik **Reset Data Hari Ini** agar siap dipakai operasional.

---

## 9. Operasional Rutin: Update, Backup, & Troubleshooting

### A. Update Aplikasi (Deploy Perubahan Baru dari GitHub)
Setiap kali ada update kode di branch `main`:
```bash
cd /var/www/panji-mbg
git pull origin main
docker compose up -d --build
docker image prune -f
```

### B. Backup Database SQLite dengan Aman
Karena SQLite berjalan dengan mode **WAL (Write-Ahead Logging)**, gunakan perintah `.backup` bawaan `sqlite3` agar backup selalu utuh walau ada pelanggan yang sedang memesan:

```bash
mkdir -p /var/www/panji-mbg/backups
sqlite3 /var/www/panji-mbg/data/app.db ".backup '/var/www/panji-mbg/backups/app-$(date +%F).db'"
```

Untuk **backup otomatis setiap malam jam 23:00 WIB** (menyimpan 14 hari terakhir):
```bash
(crontab -l 2>/dev/null; echo "0 23 * * * mkdir -p /var/www/panji-mbg/backups && sqlite3 /var/www/panji-mbg/data/app.db \".backup '/var/www/panji-mbg/backups/app-\$(date +\%F).db'\" && find /var/www/panji-mbg/backups -name 'app-*.db' -mtime +14 -delete") | crontab -
```

### C. Reset PIN Admin Jika Lupa
Jika lupa PIN 6-digit, reset langsung melalui host server:
```bash
sqlite3 /var/www/panji-mbg/data/app.db "UPDATE Config SET value='123456' WHERE key='admin_pin';"
```

### D. Cek Masalah Umum
| Gejala | Cara Cek & Solusi |
| :--- | :--- |
| `502 Bad Gateway` di Nginx | Cek `docker compose ps` & `docker compose logs panji-mbg`. Pastikan container *Up* di `127.0.0.1:8080`. |
| Terkunci `Terlalu banyak percobaan` saat login | Proteksi anti *brute-force* aktif selama 5 menit setelah 5x salah PIN. Untuk mereset segera: `docker compose restart panji-mbg`. |
| Gambar QRIS gagal di-upload (Error 413) | Pastikan baris `client_max_body_size 4m;` sudah ada di blok `server` Nginx lalu `sudo systemctl reload nginx`. |

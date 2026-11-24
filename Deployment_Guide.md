# Panduan Deployment Production — Panji MBG (Native Go + PM2 + Nginx + Cloudflare)

Panduan resmi menjalankan **Panji MBG** secara **Native Binary dengan PM2** (tanpa overhead Docker) agar proses deploy super cepat (**0–2 detik**), hemat RAM (**~12 MB**), dan tidak membebani CPU server yang berbagi dengan aplikasi lain.

---

## ⚡ Kenapa Native Go + PM2 Jauh Lebih Efektif daripada Docker?

1. **Cache Kompilasi Permanen (`~/.cache/go-build`):**
   - Di Docker, cache kompilasi SQLite terbuang setiap rebuild sehingga CPU naik 99% selama ~60 detik.
   - Di Native Host, Go hanya meng-compile `modernc.org/sqlite` **sekali saja di awal**. Deploy berikutnya hanya memakan waktu **1–2 detik**!
2. **Smart Skip Compile (0 Detik):**
   - Script `deploy.sh` otomatis mendeteksi file apa yang berubah. Jika kamu hanya mengubah tampilan `index.html`, `logo.jpg`, atau `qris.jpg`, script **melewati proses compile & restart sama sekali (0 detik)** karena file statis langsung dibaca dari disk!
3. **CPU-Friendly (`nice -n 19` + `GOMAXPROCS=1`):**
   - Saat ada perubahan di `main.go`, proses `go build` dijalankan dengan prioritas CPU paling rendah sehingga aplikasi lain di server (seperti `deflow-backend` atau `waha`) tidak akan pernah terganggu.
4. **Satu Komando di PM2:**
   - Berjalan berdampingan rapi di `pm2 list` bersama aplikasi lain, lengkap dengan *auto-restart* saat crash atau saat server *reboot*.

---

## 1. Migrasi / Setup Pertama Kali di Server Ubuntu

Jika sebelumnya kamu sudah sempat menyalakan versi Docker di `/var/www/panji-mbg`, script `deploy.sh` sudah pintar: **dia otomatis mematikan container Docker lama, menjaga kepemilikan database `data/app.db` tetap utuh, menginstal Go 1.22 resmi (jika belum ada), lalu memindahkannya ke PM2!**

Cukup login sebagai user kamu (misal `ipangpangeran`, user yang sama tempat `pm2` berjalan), lalu jalankan:

```bash
# Pastikan folder dimiliki oleh user kamu (bukan root)
sudo chown -R $USER:$USER /var/www/panji-mbg

# Masuk ke folder, tarik script terbaru, & pasang shortcut global
cd /var/www/panji-mbg
git pull origin main
chmod +x deploy.sh
sudo ln -sf /var/www/panji-mbg/deploy.sh /usr/local/bin/deploy-mbg

# Jalankan deploy pertama kali!
deploy-mbg
```

*(Catatan: Khusus jalankan `deploy-mbg` yang **pertama kali**, Go akan diunduh otomatis dan mengisi cache SQLite selama ~30–40 detik dengan beban CPU dibatasi aman. Mulai deploy ke-2 dan seterusnya, kecepatannya langsung **0–2 detik**!).*

---

## 2. Deploy Update Selanjutnya (Cukup 1 Kata!)

Setiap kali selesai `git push` dari laptop ke GitHub, cukup login SSH ke server dan ketik dari direktori mana saja:

```bash
deploy-mbg
```

Alur yang dikerjakan otomatis oleh `deploy-mbg`:
1. Pindah ke `/var/www/panji-mbg`
2. *Auto-backup* database SQLite ke `backups/pre-deploy-*.db` (menyimpan 10 backup terakhir)
3. Menarik commit terbaru dari GitHub (`git pull origin main`)
4. Mengecek apakah `main.go` berubah → compile cepat pakai cache host (atau *skip* 0 detik jika hanya HTML/aset yang berubah)
5. Menyalakan / me-restart proses `panji-mbg` di PM2 + `pm2 save`
6. Mengecek *healthcheck* `http://127.0.0.1:8080/healthz`

---

## 3. Perintah Sehari-hari di PM2

```bash
pm2 status                 # Cek status semua aplikasi (deflow-backend, panji-mbg, dll)
pm2 logs panji-mbg         # Lihat log akses real-time Panji MBG
pm2 restart panji-mbg      # Restart manual (0.1 detik)
pm2 monit                  # Monitor pemakaian RAM & CPU real-time
```

Jika ingin mengubah port atau environment variable produksi, edit file `ecosystem.config.cjs` lalu jalankan:
```bash
pm2 restart ecosystem.config.cjs --env production --update-env
pm2 save
```

---

## 4. Konfigurasi Nginx + Cloudflare (`mbgpanji.my.id`)

File `/etc/nginx/sites-available/panji-mbg` tetap sama persis (meneruskan ke `127.0.0.1:8080`):

```nginx
server {
    listen 80;
    server_name mbgpanji.my.id www.mbgpanji.my.id;

    client_max_body_size 4M;
    real_ip_header CF-Connecting-IP;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $http_x_forwarded_proto;
        proxy_read_timeout 60s;
    }
}
```

---

## 5. Backup Harian & Reset PIN Admin

### Backup Otomatis via Cron (Jam 23:00 WIB)
```bash
(crontab -l 2>/dev/null; echo "0 23 * * * mkdir -p /var/www/panji-mbg/backups && cp /var/www/panji-mbg/data/app.db /var/www/panji-mbg/backups/daily-\$(date +\%F).db && find /var/www/panji-mbg/backups -name 'daily-*.db' -mtime +14 -delete") | crontab -
```

### Reset PIN Admin Jika Lupa
```bash
sudo apt install -y sqlite3   # jika sqlite3 belum ada
sqlite3 /var/www/panji-mbg/data/app.db "UPDATE Config SET value='123456' WHERE key='admin_pin';"
```

*(Catatan: File `Dockerfile` dan `docker-compose.yml` tetap disimpan di repository sebagai opsi alternatif jika suatu saat ingin dipindah ke server khusus Docker).*

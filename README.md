<div align="center">

# 🍱 Panji MBG

**Aplikasi Web Pemesanan Makanan Harian — Cepat, Ringan, & Tanpa Ribet**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-Web_Framework-008ECF?style=flat-square)](https://gin-gonic.com/)
[![SQLite](https://img.shields.io/badge/SQLite-Pure_Go_(No_CGO)-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://pkg.go.dev/modernc.org/sqlite)
[![Alpine.js](https://img.shields.io/badge/Alpine.js-3.14-8BC0D0?style=flat-square&logo=alpinedotjs&logoColor=black)](https://alpinejs.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white)](https://www.docker.com/)

Aplikasi pemesanan katering/warung makan berbasis **Go (Gin) + SQLite + Tailwind/Alpine.js** dalam satu binary ringan. Dirancang agar pelanggan bisa memesan dalam hitungan detik dari HP tanpa login, sementara dapur/admin dapat mengatur kuota, menu, rekap WhatsApp, dan export Excel secara *real-time*.

</div>

---

## ✨ Fitur Utama

### 👤 Pelanggan (Tanpa Login)
- **Status Real-Time:** Indikator langsung `🟢 Warung Buka` / `🔴 Warung Tutup` beserta sisa kuota porsi hari ini.
- **Multi-Item & Multi-Qty:** Satu orang bisa memesan lebih dari satu porsi sekaligus atau kombinasi beberapa menu utama & *add-ons* menggunakan tombol stepper `−` / `+`.
- **Kategori Custom Fleksibel:**
  - *Quick chips* sekali klik (contoh: `Tanpa Nasi`, `Tanpa Kerupuk`, `Tanpa Sayur/Tumis`, `Nasi Setengah`).
  - Kolom catatan bebas untuk detail spesifik (contoh: *"1 porsi ayam tanpa nasi, sambal pisah"*).
- **Kalkulasi Harga Otomatis:** Otomatis membaca label harga pada nama menu seperti `(15k)` atau `(3k)` untuk menampilkan estimasi total tagihan.
- **Pembayaran Cash & QRIS:** Menampilkan kartu **QRIS resmi MBG APS** lengkap dengan tombol unduh gambar QRIS agar mudah di-*scan* dari galeri HP.

### 🔐 Admin Dashboard (Protected 6-Digit PIN)
- **PIN Keypad Modern:** Login aman menggunakan 6-digit PIN dengan proteksi *brute-force* (otomatis *lockout* 5 menit setelah 5x salah PIN).
- **Kontrol Operasional Instan:** *Switch* Buka/Tutup warung dan pengaturan kuota porsi global harian dengan indikator *progress bar*.
- **Manajemen Menu Dinamis:** Tambah, ubah, atau hapus daftar **Menu Utama**, **Add-ons**, dan **Opsi Kategori Custom** kapan saja tanpa menyentuh kode.
- **Pengaturan QRIS & PIN:** Ubah gambar QRIS (via URL atau *upload* file langsung) serta ganti PIN Admin dari dashboard.
- **Laporan & Rekap Otomatis:**
  - Ringkasan total porsi terjual, jumlah pemesan, hitungan per menu utama, *add-ons*, pesanan *custom*, dan metode bayar.
  - **Salin Rekap WA:** Menyalin format teks rapi siap *paste* ke grup WhatsApp dapur/pelanggan.
  - **Export Excel (`.xlsx`):** Mengunduh laporan lengkap berisi 2 *sheet* (*Pesanan* & *Ringkasan*) lengkap dengan *styling* tabel dan *auto-filter*.
  - **Reset Data Harian:** Menghapus daftar pesanan hari ini (nomor urut kembali ke `#001`) tanpa menghapus konfigurasi menu.

---

## 🖼️ Preview Desain UI

| Form Pelanggan | Login PIN Admin | Dashboard Admin |
| :---: | :---: | :---: |
| ![Customer View](stitch_warung_mbg_order_system/customer_order_view/screen.png) | ![Admin Login](stitch_warung_mbg_order_system/admin_login/screen.png) | ![Admin Dashboard](stitch_warung_mbg_order_system/admin_dashboard/screen.png) |

---

## 🛠️ Tech Stack & Arsitektur

| Komponen | Teknologi | Keterangan |
| :--- | :--- | :--- |
| **Backend** | Go 1.21+ (`github.com/gin-gonic/gin`) | Single-file REST API + static file server |
| **Database** | SQLite (`modernc.org/sqlite`) | Pure Go SQLite driver (tanpa CGO, mode WAL aktif) |
| **Excel Engine** | `github.com/xuri/excelize/v2` | Generator laporan `.xlsx` multi-sheet di memori |
| **Frontend** | HTML5 + Tailwind CSS + Alpine.js | Glassmorphism UI responsif (*mobile-first*), tanpa *build step* Node.js |
| **Deployment** | Docker Multi-Stage + Nginx | Image akhir berbasis `alpine:latest` (~20 MB) |

```text
mbg-panji/
├── main.go                # Seluruh logika backend: SQLite, Auth PIN, REST API, & Export Excel
├── index.html             # Single-Page App (Pelanggan & Admin Dashboard)
├── logo.jpg               # Logo resmi MBG APS (Header & Favicon)
├── qris.jpg               # Gambar QRIS default (MBG APS, Layanan Katering)
├── ecosystem.config.cjs   # Konfigurasi PM2 production (Native Go binary, RAM ~12 MB)
├── deploy.sh              # Script 1-klik auto-deploy Native Go + PM2 (0-2 detik!)
├── run-dev.ps1            # Script helper menjalankan server lokal di Windows
├── Dockerfile             # (Opsional) Alternatif deployment via container Docker
├── docker-compose.yml     # (Opsional) Konfigurasi Docker Compose
├── go.mod / go.sum      # Dependensi Go
└── data/                # (Otomatis dibuat) Lokasi file SQLite app.db
```

---

## 💻 Menjalankan di Lokal (Development — Tanpa Docker)

Untuk pengembangan di PC lokal (Windows / macOS / Linux), **tidak perlu Docker**. Cukup pastikan **Go 1.21+** sudah terpasang.

### Di Windows (PowerShell)
```powershell
git clone https://github.com/ipangpangeran/mbg-panji.git
cd mbg-panji

# Jalankan dengan script dev:
.\run-dev.ps1

# Atau jalankan langsung:
go run .
```
> Jika PowerShell menolak eksekusi script, jalankan sekali: `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`

### Di Linux / macOS
```bash
git clone https://github.com/ipangpangeran/mbg-panji.git
cd mbg-panji
TZ=Asia/Jakarta GIN_MODE=debug go run .
```

Buka di browser:
- **Halaman Pemesanan:** `http://localhost:8080`
- **Halaman Admin:** `http://localhost:8080/#admin` *(PIN awal default: `123456`)*

> **Tips format penulisan harga menu di Admin:**
> Akhiri nama menu dengan tanda kurung harga seperti `(15k)`, `(3k)`, atau `(Rp 15.000)`, dan pisahkan deskripsi isi paket dengan ` - `.
> Contoh: `Paket Ayam - Nasi + Ayam Rendang + Tumis Buncis + Krupuk (15k)` — UI otomatis memisahkan judul, rincian lauk, serta menghitung estimasi total belanja pelanggan.

---

## 🔌 Dokumentasi REST API

Seluruh *endpoint* admin dilindungi menggunakan HTTP Header `X-Admin-Pin: <6-digit-pin>`.

| Method | Endpoint | Akses | Deskripsi |
| :--- | :--- | :---: | :--- |
| `GET` | `/api/config` | Publik | Mengambil status buka/tutup, kuota, sisa slot, daftar menu, opsi custom, & URL QRIS (PIN tidak pernah diekspos) |
| `POST` | `/api/login` | Publik | Verifikasi 6-digit PIN admin (`{"pin": "123456"}`) |
| `POST` | `/api/order` | Publik | Mengirim pesanan baru (validasi status buka & sisa kuota porsi secara atomik) |
| `POST` | `/api/config` | **Admin** | Update parsial konfigurasi (`is_open`, `global_quota`, `main_menus`, `side_menus`, `custom_options`, `qr_url`, `admin_pin`) |
| `GET` | `/api/orders` | **Admin** | Daftar seluruh pesanan diurutkan dari yang terbaru |
| `DELETE` | `/api/orders` | **Admin** | Reset/hapus semua pesanan hari ini & kembalikan nomor urut ke `#001` |
| `GET` | `/api/summary` | **Admin** | Statistik total porsi, jumlah transaksi, rekap per item, custom, & metode bayar |
| `GET` | `/api/export/excel` | **Admin** | Unduh file `.xlsx` (*Sheet 1: Pesanan*, *Sheet 2: Ringkasan*) |
| `GET` | `/healthz` | Publik | Healthcheck container & koneksi database |

<details>
<summary><b>Contoh Payload <code>POST /api/order</code></b></summary>

```json
{
  "name": "Budi Santoso",
  "main_menus": [
    {
      "name": "Paket Ayam - Nasi + Ayam Rendang + Tumis Buncis Labu Siam + Krupuk (15k)",
      "qty": 2
    }
  ],
  "side_menus": [
    { "name": "Martabak Tahu 2pcs (3k)", "qty": 1 },
    { "name": "Orek Tempe (3k)", "qty": 2 }
  ],
  "custom_options": ["Tanpa Kerupuk"],
  "custom_note": "1 porsi tanpa nasi, sambal dipisah",
  "payment_method": "qris"
}
```
</details>

---

## ⚙️ Variabel Environment

| Variabel | Default | Keterangan |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port HTTP tempat server berjalan |
| `DB_PATH` | `data/app.db` | Lokasi penyimpanan file database SQLite (`/app/data/app.db` di dalam Docker) |
| `STATIC_DIR` | `.` | Direktori file statis `index.html` dan `qris.jpg` |
| `TZ` | `Asia/Jakarta` | Zona waktu pencatatan jam pesanan & penamaan file export |
| `ADMIN_PIN` | `123456` | PIN 6-digit awal (**hanya dipakai saat database pertama kali dibuat**) |
| `GIN_MODE` | `release` | Mode Gin (`debug` untuk lokal, `release` untuk production) |

---

## 🚀 Deployment ke Production (Rekomendasi: Native Go + PM2 + Nginx)

Untuk server VPS yang berbagi resource dengan aplikasi lain, **sangat direkomendasikan menjalankan binary Go langsung via PM2 (tanpa Docker)** menggunakan script otomatis [deploy.sh](deploy.sh):
- **Deploy ke-2 dst cuma 0–2 detik** (karena cache kompilasi SQLite tersimpan di host, dan perubahan `index.html`/aset otomatis *skip* compile).
- **Hemat CPU & RAM:** Dibatasi `GOMAXPROCS=1` + `nice -n 19` saat compile, dan hanya memakan **~12 MB RAM** saat melayani 50+ pengguna bersamaan.

```bash
# Setup pertama kali di server Ubuntu (otomatis install Go jika belum ada & migrasi dari Docker):
sudo mkdir -p /var/www/panji-mbg && sudo chown -R $USER:$USER /var/www/panji-mbg
cd /var/www/panji-mbg
git clone https://github.com/ipangpangeran/mbg-panji.git .
chmod +x deploy.sh
sudo ln -sf /var/www/panji-mbg/deploy.sh /usr/local/bin/deploy-mbg

# Jalankan deploy! (Ke depannya cukup ketik 1 kata ini setiap habis git push):
deploy-mbg
```

Panduan lengkap konfigurasi Nginx + Cloudflare serta alternatif Docker tersedia di **[Deployment_Guide.md](Deployment_Guide.md)**.

---

## 🔄 Update & Backup di Server

### Update ke Versi Terbaru dari GitHub (1 Perintah via `deploy.sh`)
Pasang *shortcut* sekali saja di server:
```bash
chmod +x /var/www/panji-mbg/deploy.sh
sudo ln -sf /var/www/panji-mbg/deploy.sh /usr/local/bin/deploy-mbg
```
Selanjutnya setiap habis `git push`, cukup jalankan dari direktori mana saja:
```bash
deploy-mbg
```

### Backup Database SQLite
Seluruh data pesanan & konfigurasi tersimpan aman di `./data/app.db`:
```bash
sudo apt install -y sqlite3
sqlite3 /opt/mbg-panji/data/app.db ".backup '/opt/mbg-panji/backup-$(date +%F).db'"
```

### Reset PIN Admin (Jika Lupa)
Jalankan dari host server pada folder project:
```bash
sqlite3 data/app.db "UPDATE Config SET value='123456' WHERE key='admin_pin';"
```

---

<div align="center">
  <sub>Dibuat dengan ❤️ untuk operasional <b>Panji MBG (MBG APS, Layanan Katering)</b></sub>
</div>

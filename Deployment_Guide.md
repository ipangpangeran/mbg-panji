# Deployment Panji MBG on Ubuntu (Docker + Nginx)

## 1. Persiapan Server
Pastikan server sudah punya:
- Docker & Docker Compose installed.
- Nginx installed and running.
- Git installed (opsional, untuk clone repo nanti).

## 2. Setup Aplikasi
```bash
# Buat direktori aplikasi
mkdir -p /var/www/panji-mbg
cd /var/www/panji-mbg

# Buat folder data untuk persistence database
mkdir data
```

## 3. Konfigurasi Docker
Buat `docker-compose.yml`:

```yaml
version: '3.8'

services:
  panji-mbg:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Jakarta
    restart: unless-stopped
```

Buat `Dockerfile`:

```dockerfile
# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main .

# Stage 2: Run
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY index.html .
EXPOSE 8080
CMD ["./main"]
```

## 4. Build & Run
```bash
docker-compose up -d --build
```
Cek logs:
```bash
docker-compose logs -f
```

## 5. Konfigurasi Nginx
Edit file Nginx Virtual Host (misal di `/etc/nginx/sites-available/panji-mbg`):

```nginx
server {
    listen 80;
    server_name panji.ipangpangeran.com; # Ganti dengan domain kamu

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable site:
```bash
sudo ln -s /etc/nginx/sites-available/panji-mbg /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## 6. Testing
- Buka browser: `http://panji.ipangpangeran.com`
- Admin Login: Gunakan PIN default (biasanya `123456` atau sesuai config awal).
- Test Order & Export Excel.

## 7. Backup Data
Data tersimpan di `data/app.db`. Backup folder ini secara berkala!

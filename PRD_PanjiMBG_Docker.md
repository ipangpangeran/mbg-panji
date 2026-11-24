# PRD: Panji MBG (Docker Go + Arc UI)

## 1. Overview
**Nama Aplikasi:** Panji MBG
**Platform:** Docker Container (Go Backend + Static Frontend)
**Database:** SQLite
**Design System:** Arc UI (Modern, Elegant, Responsive)
**Philosophy:** Lightweight, Fast, Minimalist.

## 2. Architecture
- **Backend (Go/Gin):** API REST for data exchange (Config, Orders).
- **Frontend (HTML/Arc UI):** Static files served by Go or Nginx.
- **Database:** `app.db` (SQLite) inside container.
- **Volume:** Mount `app.db` and config to host for persistence.

## 3. User Interface (Customer)
**Goal:** Order without friction.

1.  **Status Header:**
    *   Simple Badge: 🟢 "Warung Buka" / 🔴 "Warung Tutup".
    *   No clock, just status.
2.  **Form Order:**
    *   **Nama:** Text Input (Clear, large font).
    *   **Menu Utama:** Dropdown (Select). Options from DB.
    *   **Add-ons:** Checkboxes (Horizontal or Grid).
    *   **Payment:** Radio Buttons (Cash / QRIS).
        *   If QRIS: Show QR Image (Base64 or URL from DB).
3.  **Submit:**
    *   Button: "Pesan Sekarang".
    *   Success: Simple toast notification "Terima kasih, [Nama]!".
    *   Auto-clear form.
    *   **NO** copy buttons, **NO** extra text.

## 4. Admin Interface (Protected)
**Goal:** Manage status, menus, quotas, and reports.
**Access:** PIN 6 Digits (Numeric only).

1.  **Login:**
    *   Input PIN (6 digits).
2.  **Dashboard:**
    *   **Toggle Status:** Switch "Warung Buka/Tutup".
    *   **Global Quota:** Input Number "Total Slot Hari Ini".
    *   **Menu Config:**
        *   List Input for Main Menus (Add/Remove row).
        *   List Input for Side Menus (Add/Remove row).
        *   Button: "Simpan Konfigurasi".
    *   **QRIS/Settings:** Input URL for QR Image.
3.  **Reporting:**
    *   **Order List:** Table showing Name, Menu, Add-ons, Time, Payment. (Clean, readable).
    *   **Summary Stats:**
        *   Total Orders: X
        *   Paket Ayam: Y
        *   Paket Bandeng: Z
    *   **Actions:**
        *   "Salin Rekap ke WA": Copies formatted text to clipboard.
        *   "Export Excel": Downloads `.xlsx` file of all orders.
    *   **Reset:** "Reset Data Hari Ini" (Clears orders, keeps config).

## 5. API Endpoints
- `GET /api/config`: Get isOpen, quota, menus, pin, qr_url.
- `POST /api/config`: Update config.
- `POST /api/order`: Submit order.
- `GET /api/orders`: Get all orders (Admin only).
- `DELETE /api/orders`: Clear orders (Admin only).
- `GET /api/export/excel`: Download Excel file.

## 6. Tech Stack Details
- **Language:** Go 1.21+
- **Web Framework:** Gin
- **Database Driver:** `modernc.org/sqlite` (Pure Go, no CGO needed, easier Docker build).
- **Excel Generation:** `github.com/xuri/excelize/v2`.
- **Frontend:** Arc UI (CSS/JS via CDN), Alpine.js (for reactivity).

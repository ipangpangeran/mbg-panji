# Jalankan Panji MBG untuk development lokal (Windows, tanpa Docker)
# Pemakaian:  .\run-dev.ps1            (port 8080)
#             .\run-dev.ps1 -Port 9000
param([string]$Port = "8080")

$env:GOTOOLCHAIN = "local"
$env:GIN_MODE    = "debug"
$env:PORT        = $Port
$env:DB_PATH     = "data\app.db"
$env:TZ          = "Asia/Jakarta"

Write-Host "Panji MBG dev  ->  http://localhost:$Port   (admin: http://localhost:$Port/#admin, PIN awal 123456)" -ForegroundColor Green
go run .

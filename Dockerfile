# ---------- Stage 1: build ----------
FROM golang:1.21-alpine AS builder
WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
# modernc.org/sqlite is pure Go -> no CGO needed, fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/panji-mbg .

# ---------- Stage 2: run ----------
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /out/panji-mbg /app/panji-mbg
COPY index.html qris.jpg logo.jpg /app/
RUN mkdir -p /app/data

ENV PORT=8080 \
    DB_PATH=/app/data/app.db \
    STATIC_DIR=/app \
    TZ=Asia/Jakarta \
    GIN_MODE=release

EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

CMD ["/app/panji-mbg"]

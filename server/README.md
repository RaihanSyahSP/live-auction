# Server

Backend Go untuk LELANG.

Fitur yang sudah tersedia:

- autentikasi access token dan refresh token berbasis httpOnly cookie;
- auction lifecycle dan admin endpoints;
- bid dengan transaksi PostgreSQL, row lock, idempotency, dan anti-sniping;
- WebSocket room per auction;
- scheduler penutupan lelang dan restart recovery;
- request logging, metrics Prometheus, dan health check;
- seed data demo;
- load test lokal.

## Prasyarat

- Go sesuai `go.mod`
- PostgreSQL 15+ (PostgreSQL 18 memakai `uuidv7()` native)
- `golang-migrate`
- PowerShell untuk contoh perintah di dokumen ini

Schema memakai `uuidv7()`. PostgreSQL 18 memakai fungsi native; PostgreSQL 15–17 memakai fallback yang dibuat oleh migration.

## Environment

Jalankan semua perintah dari folder `server` agar `.env` ditemukan.

```powershell
cd D:\live-auction\server
Copy-Item .env.example .env
```

Isi `.env`:

```env
PORT=8080
DATABASE_URL=postgres://postgres:root@localhost:5432/lelang?sslmode=disable
APP_ORIGIN=http://localhost:4173
JWT_SECRET=replace-with-a-random-32-byte-secret
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
REFRESH_GRACE_PERIOD=10s
COOKIE_SECURE=false
SEED_ADMIN_EMAIL=admin@example.com
SEED_ADMIN_PASSWORD=change-me
SEED_BIDDER_PASSWORD=change-me
```

`JWT_SECRET` minimal 32 karakter.

## Database

Buat database development:

```powershell
psql -U postgres -h localhost -c "CREATE DATABASE lelang;"
```

Jalankan migration:

```powershell
$env:DATABASE_URL="postgres://postgres:root@localhost:5432/lelang?sslmode=disable"
migrate -path migrations -database $env:DATABASE_URL up
```

Jika `migrate` belum terpasang:

```powershell
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Menjalankan API

```powershell
cd D:\live-auction\server
go run ./cmd/api
```

Server mendengarkan di `PORT`, default `8080`.

Endpoint awal:

```text
GET  /healthz
GET  /metrics
GET  /ws/auctions/{id}
GET  /api/auctions/current
GET  /api/auctions/{id}
POST /api/auctions/{id}/bids
```

`/healthz` melakukan ping database. Jika database tidak tersedia, endpoint mengembalikan `503`.

## Seed

Seed membuat:

- admin demo;
- bidder demo;
- satu auction live;
- satu auction draft.

```powershell
cd D:\live-auction\server
go run ./cmd/seed
```

Seed aman dijalankan berulang. Data dicari berdasarkan email dan lot sebelum dibuat.

## Test

Repository dan seed test memakai database test terpisah.

```powershell
$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"

migrate -path migrations -database $env:TEST_DATABASE_URL up
go test ./...
```

Jika `TEST_DATABASE_URL` kosong, test yang membutuhkan database akan di-skip.

Verifikasi lain:

```powershell
go build ./...
go vet ./...
```

Jika CGO dan race detector tersedia:

```powershell
go test -race ./...
```

## Health dan Metrics

```powershell
Invoke-RestMethod http://localhost:8080/healthz
Invoke-WebRequest http://localhost:8080/metrics
```

Metrics utama:

```text
auction_connected_clients
auction_bids_total
auction_bid_duration_seconds
auction_broadcast_duration_seconds
auction_live_auctions
auth_login_total
auth_refresh_total
```

Metrics memakai registry Prometheus sendiri. Menjalankan Prometheus server atau Grafana tidak wajib untuk development. Docker juga tidak wajib.

## Load Test

Load test hanya untuk local atau staging.

```powershell
cd D:\live-auction\server

$env:SEED_BIDDER_PASSWORD="change-me"

go run ./loadtest `
  -base-url http://localhost:8080 `
  -auction-id "<auction-uuid>" `
  -viewers 50 `
  -bidders 5 `
  -rate 5 `
  -duration 30s
```

Load test akan mencoba login sebagai `loadtest1@example.com` sampai `loadtestN@example.com`. Jika akun belum ada, load test mencoba register. Rate limit autentikasi dapat membatasi jumlah akun yang dibuat sekaligus.

Gunakan `-allow-remote` hanya jika target memang staging yang diizinkan. Load test menolak host yang terlihat seperti production secara default.

## Catatan Auth

- Access token dan refresh token disimpan di cookie `httpOnly`.
- Jangan menyimpan refresh token di `localStorage`.
- Gunakan same-origin reverse proxy atau subdomain same-site agar cookie bekerja dengan baik.
- Untuk HTTPS, set `COOKIE_SECURE=true`.

## Struktur

```text
server/
  cmd/
    api/              entrypoint HTTP + WebSocket
    seed/             entrypoint seed data demo
  internal/
    app/              server, router, scheduler, database pool
    auth/             access token, refresh token, cookie
    controller/       HTTP handler
    service/          aturan bisnis
    repository/       query PostgreSQL
    model/            domain dan DTO
    middleware/       auth, origin, rate limit, logging
    metrics/          Prometheus collectors
    realtime/         room manager dan WebSocket
    seed/             logika seed yang bisa diuji
    exception/        error domain dan mapping HTTP
    helper/           JSON, UUID, test helper
  migrations/
  loadtest/
  .env.example
  go.mod
```

## Dokumen

- Spec backend: `../docs/superpowers/specs/2026-10-20-auction-backend-design.md`
- Plan implementasi: `../docs/superpowers/plans/2026-10-20-auction-backend.md`
- PRD: `../docs/prd/2026-10-20-live-auction-prd.md`

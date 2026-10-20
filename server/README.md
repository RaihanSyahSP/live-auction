# Server

Backend Go untuk LELANG. Isi folder ini masih berupa kerangka. Logika bisnis, handler, repository, dan WebSocket belum diimplementasikan.

## Dokumen

- Spec backend: `docs/superpowers/specs/2026-10-20-auction-backend-design.md`
- Plan implementasi: `docs/superpowers/plans/2026-10-20-auction-backend.md`
- Spec produk dan teknis: `docs/superpowers/specs/2026-10-20-live-auction-design.md`
- PRD: `docs/prd/2026-10-20-live-auction-prd.md`

## Struktur

```text
server/
  cmd/
    api/              entrypoint HTTP + WebSocket
    seed/             seed admin, bidder, dan lelang demo
  internal/
    config/           environment
    app/              server, router, database pool, validasi, wiring
    auth/             access token JWT, refresh token, dan cookie
    controller/       HTTP handler
    service/          aturan bisnis auth, auction, bid, admin
    repository/       query PostgreSQL dan transaksi
    model/
      domain/         entity User, Auction, Bid
      web/            request dan response DTO
    middleware/       auth, admin, logging, recovery
    exception/        error domain dan mapping HTTP
    helper/           JSON, parsing request, test helper
    realtime/         WebSocket client, room manager, event envelope
    auction/          aturan lelang dan lifecycle room
  migrations/         SQL migration
  loadtest/           load test WebSocket dan bid
  docker-compose.yml  PostgreSQL lokal
  .env.example
  go.mod
```

## Referensi

Struktur layer mengikuti `D:\belajar-golang\golang-restful`, tetapi ada perbedaan yang disengaja:

- Router memakai `net/http`, bukan `httprouter`.
- Database memakai `pgx`, bukan `lib/pq`.
- Wiring manual di `cmd/api/main.go`, bukan Wire.
- Error dikembalikan eksplisit, bukan panic/recover.
- Tidak memakai ORM.
- Auth memakai access token JWT 15 menit dan refresh token opaque 7 hari dengan rotasi. Keduanya disimpan di cookie `httpOnly`.

## Perintah

```powershell
go build ./...
go test ./...
go run ./cmd/api
go run ./cmd/seed
```

Dependency belum ditambahkan ke `go.mod`. Ikuti plan untuk menambahkan satu per satu saat task membutuhkannya.

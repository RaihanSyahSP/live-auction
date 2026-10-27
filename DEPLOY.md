# Deploy LELANG ke Render + Supabase

Arsitektur:

```text
Browser
  -> Render Web Service (satu origin)
       -> Caddy
            /       -> Node React Router SSR :3000
            /api/*  -> Go API :8080
            /ws/*   -> Go WebSocket :8080
            /healthz -> Go health check
       -> Supabase PostgreSQL
```

Database tidak perlu satu domain dengan web. Yang harus satu origin hanya web, API, dan WebSocket. Karena semuanya lewat satu Render service, cookie auth tetap bekerja.

## 1. Cek Supabase

Di SQL Editor Supabase:

```sql
select version();
```

Schema LELANG memakai `uuidv7()`. Migration menyediakan fallback untuk PostgreSQL sebelum 18, jadi Supabase PostgreSQL 15+ tetap bisa dipakai.

Ambil connection string dari Supabase. Untuk Render yang umumnya IPv4, pakai **Session Pooler** (port `5432`) agar tidak bergantung pada koneksi IPv6:

- `DATABASE_URL`: Session Pooler;
- `MIGRATION_DATABASE_URL`: Session Pooler atau Direct Connection jika Render bisa menjangkau IPv6.

Transaction Pooler (port `6543`) bisa dipakai, tetapi backend Go memakai prepared statement pgx. Kalau memilih transaction pooler, pastikan konfigurasi prepared statement kompatibel dan test login/bid.

Gunakan `sslmode=require`.

Jika koneksi database ditolak, cek connection limit Supabase dan turunkan `DB_MAX_CONNS` (default production 10).

## 2. Import repository ke Render

1. Push repository ke GitHub/GitLab.
2. Di Render pilih **New -> Blueprint**.
3. Hubungkan repository.
4. Render membaca `render.yaml`.
5. Isi environment yang diminta:
   - `DATABASE_URL`: pooler Supabase;
   - `MIGRATION_DATABASE_URL`: session pooler Supabase, atau direct jika IPv6 tersedia;
   - seed email/password jika ingin menjalankan seed.
6. Deploy.

Jika tidak memakai Blueprint, buat manual:

```text
New -> Web Service
Runtime: Docker
Dockerfile Path: ./Dockerfile
Health Check Path: /healthz
Plan: Free
```

## 3. Environment Render

Wajib:

```env
DATABASE_URL=postgresql://...pooler.../postgres?sslmode=require
MIGRATION_DATABASE_URL=postgresql://...session-pooler.../postgres?sslmode=require
COOKIE_SECURE=true
JWT_SECRET=<random minimal 32 byte>
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
REFRESH_GRACE_PERIOD=10s
DB_MAX_CONNS=10
```

`APP_ORIGIN`:

- jika memakai domain Render, boleh dikosongkan karena entrypoint mengambil `RENDER_EXTERNAL_URL`;
- jika memakai custom domain, set:

```env
APP_ORIGIN=https://domainkamu.com
```

Seed hanya untuk demo:

```env
SEED_ADMIN_EMAIL=admin@example.com
SEED_ADMIN_PASSWORD=<password kuat>
SEED_BIDDER_PASSWORD=<password kuat>
```

## 4. Migration dan seed

Saat container start, entrypoint menjalankan:

```text
/app/bin/migrate up
```

Kemudian Caddy, Node SSR, dan Go API dijalankan oleh supervisor.

Setelah deploy sehat, buat admin/demo data lewat Render Shell:

```sh
/app/bin/seed
```

Kalau Render Shell tidak tersedia di plan free, jalankan seed dari lokal dengan direct connection Supabase:

```powershell
cd server
$env:DATABASE_URL="postgresql://...direct.../postgres?sslmode=require"
go run ./cmd/seed
```

Hapus atau rotasi password seed setelah selesai.

## 5. Cek deploy

```text
https://nama-service.onrender.com/healthz
```

Harus `200`. Lalu buka:

```text
https://nama-service.onrender.com
```

Login admin, buat draft, start lelang, lalu coba bid dari akun bidder.

## 6. Custom domain

Tambahkan domain di Render, lalu ubah:

```env
APP_ORIGIN=https://domainkamu.com
COOKIE_SECURE=true
```

Redeploy setelah mengubah env.

Jangan pisahkan web dan API ke dua hostname dengan konfigurasi saat ini. Cookie auth memakai host-only cookie dan WebSocket memakai origin halaman.

## 7. Batasan

- Render free service tidur setelah idle dan cold start bisa lama.
- WebSocket putus saat service tidur.
- Scheduler lelang berhenti saat service tidur.
- Supabase free bisa pause setelah periode tidak aktif.
- Auto-migration di entrypoint aman untuk satu instance. Untuk multi-instance, pindahkan migration ke job terpisah.
- Untuk lelang live serius, gunakan Render berbayar atau VPS/Oracle Cloud Always Free.

## 8. Build lokal

```powershell
docker build -t lelang .
```

Test container dengan database lokal:

```powershell
docker run --rm -p 10000:10000 `
  -e PORT=10000 `
  -e DATABASE_URL="postgres://postgres:root@host.docker.internal:5432/live_auction?sslmode=disable" `
  -e APP_ORIGIN="http://localhost:10000" `
  -e COOKIE_SECURE=false `
  -e JWT_SECRET="local-development-secret-at-least-32-bytes" `
  lelang
```

Buka `http://localhost:10000`.

## Troubleshooting

### `function uuidv7() does not exist`

Migration sekarang membuat fallback `public.uuidv7()`. Pastikan image yang dipakai adalah versi terbaru dan database belum dalam kondisi dirty.

### `Dirty database version`

Terjadi karena migration sebelumnya gagal di tengah jalan. Untuk database demo/fresh yang **belum berisi data penting**, reset objek LELANG di SQL Editor Supabase:

```sql
drop table if exists bids cascade;
drop table if exists auctions cascade;
drop table if exists refresh_tokens cascade;
drop table if exists users cascade;
drop table if exists schema_migrations;
drop function if exists public.uuidv7();
```

Lalu redeploy Render. Migration akan membuat ulang function, tabel, dan `schema_migrations` dari awal.

Jangan jalankan reset di atas jika sudah ada data produksi. Untuk data yang sudah penting, hentikan deploy dan periksa objek/tabel yang sudah terbuat sebelum melakukan perubahan manual.

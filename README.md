# Lelang

Prototipe UI ruang lelang real-time untuk elektronik bekas, lengkap dengan login, daftar, akun, hasil lelang, dan admin demo. Fokus tahap ini: alur bid, harga berjalan, ledger, presence, lifecycle lelang, dan sinkronisasi antar halaman. Backend Go dan PostgreSQL menyusul sesuai spec.

## Isi repo

- `prototype/`: prototipe statis HTML, CSS, JavaScript. Tanpa build step.
- `prototype/index.html`: ruang lelang.
- `prototype/login.html` dan `register.html`: masuk dan daftar.
- `prototype/account.html`: profil dan bid pengguna.
- `prototype/results.html`: arsip hasil lelang.
- `prototype/admin.html`: halaman admin dengan gate admin.
- `server/`: kerangka backend Go. Logika bisnis belum diisi.
- `DESIGN.md`: arah visual, palet, tipografi, dan aturan desain.
- `docs/prd/2026-10-20-live-auction-prd.md`: PRD produk, fitur, prioritas, metrik, dan rencana rilis.
- `docs/superpowers/specs/2026-10-20-live-auction-design.md`: desain produk dan teknis utama.
- `docs/superpowers/specs/2026-10-20-auction-backend-design.md`: spec backend, schema, auth, API, WebSocket.
- `docs/superpowers/plans/2026-10-20-auction-backend.md`: plan implementasi backend langkah demi langkah.

## Backend

Kerangka ada di `server/`. Struktur mengikuti layer repo referensi `golang-restful`, tetapi memakai `net/http`, `pgx`, dan manual wiring.

Auth backend memakai access token JWT 15 menit dan refresh token opaque 7 hari dengan rotasi. Keduanya disimpan di cookie `httpOnly`. Semua ID memakai `uuid` dengan default `uuidv7()` di PostgreSQL 18. Detail ada di spec backend.

## Jalankan

Dari root repo:

```powershell
python -m http.server 4173 --directory prototype
```

Buka `http://localhost:4173`.

Alternatif tanpa Python:

```powershell
npx serve prototype -l 4173
```


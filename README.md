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

Auth backend memakai access token JWT 15 menit dan refresh token opaque 7 hari dengan rotasi. Keduanya disimpan di cookie `httpOnly`. Detail ada di spec backend.

Mulai dari Task 1 di plan. Jangan menambahkan dependency sebelum task membutuhkannya.

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

## Akun demo

Semua akun demo memakai password `demo123`.

| Email | Nama | Peran |
|---|---|---|
| `raihan@example.com` | Raihan | Pembeli |
| `dewi@example.com` | Dewi | Pembeli |
| `admin@example.com` | Admin Demo | Admin |

Auth ini hanya untuk prototipe. Akun dan password disimpan di `localStorage`, sesi di `sessionStorage`, dan tidak ada pengiriman ke server. Jangan memakai password asli atau data pribadi. Auth produksi memakai backend, bcrypt, dan cookie httpOnly.

## Coba alur real-time

1. Buka `login.html` di dua tab.
2. Masuk sebagai Raihan di tab pertama dan Dewi di tab kedua. Sesi disimpan per tab, jadi dua tab bisa menjadi penawar berbeda.
3. Pasang bid di tab pertama, lalu bid lebih tinggi di tab kedua.
4. Tab pertama harus menerima harga baru dan toast "Kamu di-outbid".

Data disinkronkan lewat `BroadcastChannel` dan `localStorage`. Sinkronisasi hanya berlaku antar tab di browser dan origin yang sama. Tidak ada server.

## Coba alur akun

1. Guest membuka ruang lelang dan hanya bisa menonton. Tombol bid diganti dengan ajakan masuk.
2. Masuk lewat form atau tombol akun demo. Sesi bertahan saat halaman dimuat ulang.
3. Buka `account.html` untuk melihat profil dan bid milik akun tersebut.
4. Buka `results.html` untuk melihat arsip lelang yang sudah selesai.
5. Daftar akun baru lewat `register.html`. Akun baru memakai role pembeli dan langsung masuk.
6. Keluar lewat tombol "Keluar" di header atau halaman akun.

## Coba alur admin

Buka `http://localhost:4173/admin.html` atau klik "Admin demo" di footer ruang lelang.

1. Halaman admin meminta akun admin. Klik "Masuk sebagai admin demo" atau masuk dengan `admin@example.com`.
2. Halaman admin menampilkan satu lelang live dan satu draft.
3. End lelang live, lalu Start draft. Hanya satu lelang yang boleh live.
4. Buka draft lewat Preview sebelum dimulai. Bid dinonaktifkan selama masih draft.
5. Setelah Start, buka ruang lewat "Buka ruang", login sebagai pembeli, lalu pasang bid. Harga dan jumlah bid di admin ikut berubah.
6. End lelang dari admin. Ruang lelang dan halaman hasil menampilkan pemenang serta harga akhir.
7. Buat draft baru lewat form, lalu hapus draft jika perlu.
8. "Reset semua demo" mengembalikan data awal.

Gate admin memakai sesi demo lokal. Otorisasi sebenarnya baru berlaku saat backend dibuat, termasuk aturan bahwa admin tidak boleh bid pada lelangnya sendiri.

## Yang berfungsi

- Login, daftar, sesi per tab, role pembeli dan admin.
- Guest bisa menonton, bid memerlukan login.
- Bid dengan validasi minimum dan kelipatan.
- Harga, bidder tertinggi, countdown, dan ledger yang sinkron antar tab.
- Presence jumlah tab yang aktif.
- Notifikasi outbid.
- Anti-sniping: bid di 60 detik terakhir memperpanjang sesi 60 detik, maksimal 5 kali.
- Lelang berakhir otomatis setelah 10 menit.
- Hasil pemenang, harga akhir, dan state tidak terjual.
- Halaman akun dengan profil dan bid pengguna.
- Halaman hasil lelang untuk lelang yang sudah selesai.
- Konten lot, spesifikasi, kondisi, dan kelengkapan mengikuti data auction.
- Admin membuat draft, preview, start, end, dan hapus draft.
- Gate admin dengan akun demo.
- Hanya satu lelang live pada satu waktu.
- Harga dan jumlah bid di admin ikut berubah saat ada bid dari ruang lelang.
- Reset demo dan simulasi penantang.
- State kosong, loading, error, dan pemulihan state rusak.
- Fokus keyboard terlihat, target sentuh minimal 44px, dan layout 320px sampai desktop.

## Yang belum ada

- Backend, database, autentikasi produksi, dan otorisasi admin di server.
- Aturan "admin tidak boleh bid" belum bisa ditegakkan tanpa akun nyata.
- Password demo disimpan sebagai teks biasa di localStorage. Jangan dipakai untuk produksi.
- Pembayaran, ongkir, dan upload foto. Halaman memakai ilustrasi dan label "Foto unit belum tersedia".
- Rate limit, idempotency, dan audit trail di server.

Semua data di halaman adalah data demo dan diberi label "Data demo".

## Verifikasi

Skrip cek browser memakai Playwright dan Chrome yang sudah terpasang. Jalankan server statis lebih dulu, lalu:

```powershell
cd prototype/tests
npm install
npm test
```

Jika memakai Edge:

```powershell
$env:PW_CHANNEL="msedge"; npm test
```

Ada 53 cek browser: 20 untuk ruang lelang, 19 untuk admin, dan 14 untuk auth/akun. Cakupannya meliputi login, daftar, sesi, role, guest, dua tab, bid valid dan tidak valid, outbid, presence, akun, hasil lelang, reset, state ended dan unsold, pemulihan state rusak, gate admin, fokus, ukuran target, label input, dan overflow di 320px, 390px, serta 768px. Screenshot tersimpan di `prototype/tests/shots/` dan tidak di-commit.

Hasil terakhir: PASS, tanpa error console atau page error.

## Berikutnya

Backend dikerjakan manual mengikuti plan di `docs/superpowers/plans/2026-10-20-auction-backend.md`. Kerangka package dan dokumen sudah tersedia. Prototipe tetap menjadi acuan UI dan perilaku.

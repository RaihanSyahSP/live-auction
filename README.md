# Lelang

Prototipe UI ruang lelang real-time untuk elektronik bekas, lengkap dengan halaman admin demo. Fokus tahap ini: alur bid, harga berjalan, ledger, presence, lifecycle lelang, dan sinkronisasi antara ruang lelang dan admin. Backend Go dan PostgreSQL menyusul sesuai spec.

## Isi repo

- `prototype/`: prototipe statis HTML, CSS, JavaScript. Tanpa build step.
- `prototype/index.html`: ruang lelang.
- `prototype/admin.html`: halaman admin demo.
- `DESIGN.md`: arah visual, palet, tipografi, dan aturan desain.
- `docs/superpowers/specs/2026-10-20-live-auction-design.md`: desain produk dan teknis untuk implementasi berikutnya.

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

## Coba alur real-time

1. Buka halaman di dua tab.
2. Isi nama berbeda di tiap tab. Nama disimpan di `sessionStorage`, jadi setiap tab bisa menjadi penawar berbeda.
3. Pasang bid di tab pertama, lalu bid lebih tinggi di tab kedua.
4. Tab pertama harus menerima harga baru dan toast "Kamu di-outbid".

Data disinkronkan lewat `BroadcastChannel` dan `localStorage`. Sinkronisasi hanya berlaku antar tab di browser dan origin yang sama. Tidak ada server.

## Coba alur admin

Buka `http://localhost:4173/admin.html` atau klik "Admin demo" di footer ruang lelang.

1. Halaman admin menampilkan satu lelang live dan satu draft.
2. End lelang live, lalu Start draft. Hanya satu lelang yang boleh live.
3. Buka draft lewat Preview sebelum dimulai. Bid dinonaktifkan selama masih draft.
4. Setelah Start, buka ruang lewat "Buka ruang", lalu pasang bid. Harga dan jumlah bid di admin ikut berubah.
5. End lelang dari admin. Ruang lelang menampilkan pemenang dan harga akhir.
6. Buat draft baru lewat form, lalu hapus draft jika perlu.
7. "Reset semua demo" mengembalikan data awal.

Admin di prototipe ini tidak memakai autentikasi. Ini disengaja dan diberi label di halaman. Autentikasi dan otorisasi baru masuk saat backend dibuat.

## Yang berfungsi

- Bid dengan validasi minimum dan kelipatan.
- Harga, bidder tertinggi, countdown, dan ledger yang sinkron antar tab.
- Presence jumlah tab yang aktif.
- Notifikasi outbid.
- Anti-sniping: bid di 60 detik terakhir memperpanjang sesi 60 detik, maksimal 5 kali.
- Lelang berakhir otomatis setelah 10 menit.
- Hasil pemenang, harga akhir, dan state tidak terjual.
- Admin membuat draft, preview, start, end, dan hapus draft.
- Hanya satu lelang live pada satu waktu.
- Harga dan jumlah bid di admin ikut berubah saat ada bid dari ruang lelang.
- Reset demo dan simulasi penantang.
- State kosong, loading, error, dan pemulihan state rusak.
- Fokus keyboard terlihat, target sentuh minimal 44px, dan layout 320px sampai desktop.

## Yang belum ada

- Backend, database, autentikasi, dan otorisasi admin.
- Aturan "admin tidak boleh bid" belum bisa ditegakkan tanpa akun nyata.
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

Cek mencakup dua tab, bid valid dan tidak valid, outbid, presence, reset, simulasi penantang, state ended dan unsold, pemulihan state rusak, fokus, ukuran target, label input, dan overflow di 320px, 390px, serta 768px. Cek admin mencakup create, preview, start, end, delete, sinkronisasi harga dan pemenang dengan ruang lelang, validasi form, serta reset. Screenshot tersimpan di `prototype/tests/shots/` dan tidak di-commit.

Hasil terakhir: PASS, tanpa error console atau page error.

## Berikutnya

Implementasi backend mengikuti spec: Go, PostgreSQL, REST untuk command bid, WebSocket untuk event, actor per room, auth cookie, dan admin lifecycle. Prototipe ini menjadi acuan UI dan perilaku sebelum backend ditambahkan.

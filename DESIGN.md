# DESIGN.md

## Identitas

Nama kerja: LELANG (nama sementara, belum final).
Produk: ruang lelang live untuk elektronik bekas.
Audiens: pembeli yang ingin melihat kondisi barang, harga berjalan, dan riwayat bid tanpa distraksi marketplace.

## Arah

Katalog lelang editorial. Kertas hangat, tinta, satu aksen vermilion. Halaman harus terasa seperti lembar lot tercetak yang kebetulan hidup, bukan dashboard trading.

## Palet

- Kertas: `#F4EFE5`
- Tinta: `#1C1A17`
- Tinta muted: `#5E574D`
- Garis: `#8E8475` (border non-teks, minimal 3:1 terhadap kertas)
- Aksen vermilion: `#B33A2B`
- Permukaan: `#FBF8F1`
- Tint aksen: `#F0D9D2` (hanya untuk kilasan perubahan harga)

Alasan: palet kertas dan tinta membawa karakter balai lelang. Vermilion dipakai hanya untuk aksi bid utama dan status live yang nyata.

## Tipografi

- Display: Georgia, serif. Alasan: suara katalog dan lembar lot.
- UI/body: Segoe UI, sans. Alasan: terbaca di ukuran kecil dan familiar untuk form.
- Angka: Segoe UI dengan `font-variant-numeric: tabular-nums`. Alasan: harga sejajar di ledger.

Tanpa webfont. Sistem font menjaga prototipe tetap jalan offline.

## Layout

- Masthead dengan nomor lot dan status live yang nyata.
- Lembar lot: placeholder foto dan salinan lot.
- Area lelang dua kolom: konsol bid (sticky di desktop) dan ledger bid.
- Satu kolom di bawah 960px. Masthead dipadatkan di bawah 640px.

## Tema

Light only. Alasan: kertas katalog adalah identitasnya; versi gelap akan menjadi produk yang berbeda.

## Design Read

Reading this as: live auction room for used electronics buyers, in an editorial auction catalog visual language, dial ENERGY 2 / RHYTHM 2 / MOTION 1.

## Dial

ENERGY 2 / RHYTHM 2 / MOTION 1.

## Motion

- Perubahan harga: satu kilasan tint singkat untuk menunjuk angka baru.
- Tidak ada animasi berulang.
- Hormati `prefers-reduced-motion`.

## Aturan

- Satu aksen: vermilion untuk aksi bid utama dan status live.
- Tanpa gradient, glass, glow, grid dekoratif, atau bentuk pill.
- Tanpa statistik, testimoni, atau logo palsu.
- Data demo diberi label "Data demo" di halaman.
- Foto unit memakai placeholder yang jujur karena belum ada aset asli.

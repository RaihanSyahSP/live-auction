import type { PublicChrome } from "~/components/layout/public-chrome";
import AuctionPage from "~/features/auction/auction-page";

export const handle = {
  publicChrome: {
    note: "Ruang lelang elektronik bekas",
    footerNote:
      "Bid di 60 detik terakhir memperpanjang sesi 60 detik, maksimal 5 kali.",
    footerActions: [
      { label: "Hasil lelang", to: "/results" },
      { label: "Akun", to: "/account" },
      { label: "Admin", to: "/admin" },
    ],
  } satisfies PublicChrome,
};

export function meta() {
  return [
    { title: "LELANG: Ruang lelang live" },
    {
      name: "description",
      content: "Ruang lelang live untuk elektronik bekas.",
    },
  ];
}

export default function Home() {
  return <AuctionPage />;
}

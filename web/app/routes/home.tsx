import type { PublicChrome } from "~/components/layout/public-chrome";
import AuctionPage from "~/features/auction/auction-page";

export const handle = {
  publicChrome: {
    note: "Ruang lelang elektronik bekas",
    lotLabel: "LOT 001",
    demoTag: "Data demo",
    footerNote:
      "Bid di 60 detik terakhir memperpanjang sesi 60 detik, maksimal 5 kali.",
  } satisfies PublicChrome,
};

export function meta() {
  return [
    { title: "LELANG: Sony Alpha A7 II" },
    {
      name: "description",
      content: "Ruang lelang live untuk elektronik bekas.",
    },
  ];
}

export default function Home() {
  return <AuctionPage />;
}

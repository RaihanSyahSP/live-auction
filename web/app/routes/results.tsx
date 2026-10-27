import type { PublicChrome } from "~/components/layout/public-chrome";
import ResultsPage from "~/features/results/results-page";

export const handle = {
  publicChrome: {
    note: "Arsip hasil lelang",
    lotLabel: "HASIL",
    footerNote: "Arsip lelang yang sudah selesai.",
    footerActions: [
      { label: "Buka ruang", to: "/" },
      { label: "Admin", to: "/admin" },
    ],
  } satisfies PublicChrome,
};

export function meta() {
  return [
    { title: "Hasil Lelang: LELANG" },
    {
      name: "description",
      content: "Arsip hasil lelang elektronik bekas.",
    },
  ];
}

export default function ResultsRoute() {
  return <ResultsPage />;
}

import type { PublicChrome } from "~/components/layout/public-chrome";
import ResultsPage from "~/features/results/results-page";

export const handle = {
  publicChrome: {
    note: "Arsip hasil lelang",
    lotLabel: "HASIL",
    footerNote: "Data demo. Hasil lelang belum diambil dari backend.",
    footerActions: [
      { label: "Buka ruang", to: "/" },
      { label: "Admin demo", to: "/admin" },
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

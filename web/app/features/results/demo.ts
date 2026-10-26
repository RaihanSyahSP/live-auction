export type DemoResult = {
  id: number;
  lot: string;
  title: string;
  subtitle: string;
  winner: string;
  finalPrice: number | null;
  bidCount: number;
  endedAt: string;
};

export const demoResults: DemoResult[] = [
  {
    id: 3,
    lot: "003",
    title: "Canon EOS R",
    subtitle: "Body only",
    winner: "Bidder Demo",
    finalPrice: 12_100_000,
    bidCount: 8,
    endedAt: "2026-10-23T09:20:00+07:00",
  },
  {
    id: 4,
    lot: "004",
    title: "Nikon Z6",
    subtitle: "Body only",
    winner: "Peserta 02",
    finalPrice: 9_600_000,
    bidCount: 5,
    endedAt: "2026-10-23T08:55:00+07:00",
  },
];

import {
  type DemoAuction,
  initialAuction,
  type Spec,
} from "~/features/auction/demo";

export type AdminStatus = "live" | "draft" | "ended";

export type AdminAuction = DemoAuction & {
  id: number;
  status: AdminStatus;
  durationSeconds: number;
  winner: string;
  finalPrice: number | null;
};

export type DraftInput = {
  title: string;
  subtitle: string;
  description: string;
  condition: string;
  includes: string[];
  specs: Spec[];
  startingPrice: number;
  minIncrement: number;
  durationMinutes: number;
};

export const initialAdminAuctions: AdminAuction[] = [
  {
    ...initialAuction,
    id: 1,
    status: "live",
    durationSeconds: 600,
    winner: "",
    finalPrice: null,
  },
  {
    id: 2,
    lot: "002",
    title: "Fujifilm X-T2",
    subtitle: "Body only",
    description:
      "Mirrorless APS-C 24 MP dengan dial fisik dan layar lipat. Body saja, tanpa lensa. Kondisi mekanis normal, ada bekas pakai halus di bodi.",
    condition: "Used, 7/10",
    includes: ["Body kamera", "Baterai", "Charger"],
    specs: [
      { label: "Sensor", value: "APS-C 24 MP" },
      { label: "Mount", value: "Fujifilm X" },
      { label: "Stabilisasi", value: "Tidak ada" },
      { label: "Video", value: "4K" },
    ],
    startingPrice: 8_500_000,
    minIncrement: 100_000,
    currentPrice: 8_500_000,
    highestBidder: "",
    bids: [],
    viewerCount: 0,
    status: "draft",
    durationSeconds: 600,
    winner: "",
    finalPrice: null,
  },
];

const rank: Record<AdminStatus, number> = {
  live: 0,
  draft: 1,
  ended: 2,
};

export function sortAuctions(auctions: AdminAuction[]) {
  return [...auctions].sort(
    (a, b) => rank[a.status] - rank[b.status] || b.id - a.id,
  );
}

export function formatDuration(seconds: number) {
  return `${Math.round(seconds / 60)} menit`;
}

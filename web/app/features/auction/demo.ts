export type Bid = {
  id: number;
  name: string;
  amount: number;
  at: string;
};

export type Spec = {
  label: string;
  value: string;
};

export type DemoAuction = {
  lot: string;
  title: string;
  subtitle: string;
  description: string;
  condition: string;
  includes: string[];
  specs: Spec[];
  startingPrice: number;
  minIncrement: number;
  currentPrice: number;
  highestBidder: string;
  bids: Bid[];
  viewerCount: number;
};

export const DEMO_DURATION_SECONDS = 10 * 60;
export const MAX_EXTENSIONS = 5;
export const ANTI_SNIPE_SECONDS = 60;

export const initialAuction: DemoAuction = {
  lot: "001",
  title: "Sony Alpha A7 II",
  subtitle: "Body only",
  description:
    "Kamera full-frame 24 MP dengan stabilisasi 5-axis. Body saja, tanpa lensa. Dial dan tombol responsif, layar belakang punya baret halus. Baterai masih menyala normal.",
  condition: "Used, 8/10",
  includes: ["Body kamera", "Baterai NP-FW50", "Charger", "Tutup body"],
  specs: [
    { label: "Sensor", value: "Full-frame 24 MP" },
    { label: "Mount", value: "Sony E-mount" },
    { label: "Stabilisasi", value: "5-axis in-body" },
    { label: "Video", value: "1080p" },
  ],
  startingPrice: 6_500_000,
  minIncrement: 100_000,
  currentPrice: 7_200_000,
  highestBidder: "Peserta 03",
  bids: [
    { id: 3, name: "Peserta 03", amount: 7_200_000, at: "09:58:20" },
    { id: 2, name: "Peserta 02", amount: 6_900_000, at: "09:57:20" },
    { id: 1, name: "Peserta 01", amount: 6_700_000, at: "09:56:20" },
  ],
  viewerCount: 1,
};

export const idr = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});

export const numberFormat = new Intl.NumberFormat("id-ID");

export function formatTime(date = new Date()) {
  return new Intl.DateTimeFormat("id-ID", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
    .format(date)
    .replaceAll(".", ":");
}

export function formatRemaining(totalSeconds: number) {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export function getMinimumBid(auction: DemoAuction) {
  return auction.bids.length > 0
    ? auction.currentPrice + auction.minIncrement
    : auction.startingPrice;
}

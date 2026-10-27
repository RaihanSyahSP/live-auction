import type { Auction, AuctionStatus, Spec } from "../auction/types";

export type AdminAuction = {
  id: string;
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
  status: AuctionStatus;
  durationSeconds: number;
  winnerId: string | null;
  finalPrice: number | null;
  createdAt: string;
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

export function toAdminAuction(auction: Auction): AdminAuction {
  return {
    id: auction.id,
    lot: auction.lot,
    title: auction.title,
    subtitle: auction.subtitle ?? "",
    description: auction.description ?? "",
    condition: auction.condition,
    includes: auction.includes ?? [],
    specs: auction.specs ?? [],
    startingPrice: auction.starting_price,
    minIncrement: auction.min_increment,
    currentPrice: auction.current_price,
    status: auction.status,
    durationSeconds: auction.duration_seconds,
    winnerId: auction.winner_id ?? null,
    finalPrice: auction.final_price ?? null,
    createdAt: auction.created_at,
  };
}

const rank: Record<AuctionStatus, number> = {
  live: 0,
  draft: 1,
  ended: 2,
};

export function sortAuctions(auctions: AdminAuction[]) {
  return [...auctions].sort(
    (a, b) =>
      rank[a.status] - rank[b.status] || b.createdAt.localeCompare(a.createdAt),
  );
}

export function formatDuration(seconds: number) {
  return `${Math.round(seconds / 60)} menit`;
}

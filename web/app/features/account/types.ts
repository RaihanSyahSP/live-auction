import type { AuctionStatus } from "~/features/auction/types";

export type AccountBidStatus = "leading" | "outbid" | "won" | "lost";

export type AccountBidAuction = {
  id: string;
  lot: string;
  title: string;
  subtitle: string;
  status: AuctionStatus;
  current_price: number;
  final_price?: number | null;
  ends_at?: string;
  ended_at?: string;
};

export type AccountBid = {
  id: string;
  amount: number;
  created_at: string;
  status: AccountBidStatus;
  auction: AccountBidAuction;
};

export type AccountBidsPagination = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

export type AccountBidsResponse = {
  bids: AccountBid[];
  pagination: AccountBidsPagination;
};

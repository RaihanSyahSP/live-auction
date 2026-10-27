export interface Spec {
  label: string;
  value: string;
}

export type AuctionStatus = "draft" | "live" | "ended";

export interface Auction {
  id: string;
  created_by: string;
  lot: string;
  title: string;
  subtitle?: string;
  description?: string;
  condition: string;
  includes: string[];
  specs: Spec[];
  image_urls: string[];
  starting_price: number;
  min_increment: number;
  duration_seconds: number;
  current_price: number;
  highest_bidder_id?: string | null;
  status: AuctionStatus;
  started_at?: string;
  ends_at?: string;
  ended_at?: string;
  extension_count?: number;
  version: number;
  winner_id?: string | null;
  final_price?: number | null;
  created_at: string;
}

export interface Bid {
  id: string;
  auction_id: string;
  user_id: string;
  amount: number;
  display_name: string;
  created_at: string;
}

export interface AuctionDetail {
  auction: Auction;
  bids: Bid[];
  server_time: string;
}

export interface PlaceBidResponse {
  auction: Auction;
  bid: Bid;
  replay: boolean;
  extended: boolean;
  previous_highest_user_id?: string | null;
}

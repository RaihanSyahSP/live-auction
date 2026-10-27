import type { Auction } from "./types";

export const ANTI_SNIPE_SECONDS = 60;
export const MAX_EXTENSIONS = 5;

export function getMinimumBid(auction: Auction) {
  return auction.highest_bidder_id
    ? auction.current_price + auction.min_increment
    : auction.starting_price;
}

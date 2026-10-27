import assert from "node:assert/strict";
import test from "node:test";
import { applyAuctionEvent } from "./realtime.ts";
import { getMinimumBid } from "./rules.ts";
import type { Auction, AuctionDetail, Bid } from "./types.ts";

const auction: Auction = {
  id: "11111111-1111-4111-8111-111111111111",
  created_by: "22222222-2222-4222-8222-222222222222",
  lot: "LOT-ABC12345",
  title: "Kamera uji",
  subtitle: "Body only",
  description: "Kamera untuk pengujian live auction.",
  condition: "Used, 8/10",
  includes: ["Baterai"],
  specs: [{ label: "Sensor", value: "Full-frame" }],
  image_urls: [],
  starting_price: 1_000_000,
  min_increment: 50_000,
  duration_seconds: 600,
  current_price: 1_000_000,
  status: "live",
  extension_count: 0,
  version: 1,
  created_at: "2026-10-27T10:00:00Z",
};

const bid: Bid = {
  id: "33333333-3333-4333-8333-333333333333",
  auction_id: auction.id,
  user_id: "44444444-4444-4444-8444-444444444444",
  amount: 1_000_000,
  display_name: "Bidder Uji",
  created_at: "2026-10-27T10:01:00Z",
};

const detail: AuctionDetail = {
  auction,
  bids: [],
  server_time: "2026-10-27T10:01:00Z",
};

test("getMinimumBid uses starting price until the first bid", () => {
  assert.equal(getMinimumBid(auction), 1_000_000);

  assert.equal(
    getMinimumBid({
      ...auction,
      highest_bidder_id: bid.user_id,
      current_price: 1_100_000,
    }),
    1_150_000,
  );
});

test("applyAuctionEvent prepends one bid and ignores duplicate bid ids", () => {
  const event = {
    type: "bid" as const,
    auction_id: auction.id,
    epoch: "epoch-1",
    seq: 2,
    version: 2,
    server_time: "2026-10-27T10:01:01Z",
    payload: {
      auction: { ...auction, current_price: 1_100_000, version: 2 },
      bid: { ...bid, amount: 1_100_000 },
      extended: false,
    },
  };

  const next = applyAuctionEvent(detail, event);
  const duplicate = applyAuctionEvent(next, event);

  assert.equal(next.bids.length, 1);
  assert.equal(next.bids[0]?.id, bid.id);
  assert.equal(duplicate.bids.length, 1);
});

test("applyAuctionEvent applies an extension to ends_at and extension count", () => {
  const event = {
    type: "extended" as const,
    auction_id: auction.id,
    epoch: "epoch-1",
    seq: 3,
    version: 3,
    server_time: "2026-10-27T10:01:02Z",
    payload: {
      ends_at: "2026-10-27T10:12:00Z",
      extension_count: 1,
    },
  };

  const next = applyAuctionEvent(detail, event);

  assert.equal(next.auction.ends_at, "2026-10-27T10:12:00Z");
  assert.equal(next.auction.extension_count, 1);
});

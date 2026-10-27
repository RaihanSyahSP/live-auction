import type { Auction, AuctionDetail, Bid } from "./types";

export type AuctionSnapshot = {
  type: "snapshot";
  epoch: string;
  seq: number;
  version: number;
  server_time: string;
  auction: Auction;
  bids: Bid[];
  viewer_count: number;
};

type EventBase = {
  auction_id: string;
  epoch: string;
  seq: number;
  version: number;
  server_time: string;
};

export type BidEvent = EventBase & {
  type: "bid";
  payload: {
    auction: Auction;
    bid: Bid;
    extended: boolean;
  };
};

export type OutbidEvent = EventBase & {
  type: "outbid";
  payload: {
    auction: Auction;
    bid: Bid;
  };
};

export type ExtendedEvent = EventBase & {
  type: "extended";
  payload: {
    ends_at: string;
    extension_count: number;
  };
};

export type PresenceEvent = EventBase & {
  type: "presence";
  payload: {
    viewer_count: number;
  };
};

export type EndedEvent = EventBase & {
  type: "ended";
  payload: {
    auction: Auction;
  };
};

export type AuctionEvent =
  | BidEvent
  | OutbidEvent
  | ExtendedEvent
  | PresenceEvent
  | EndedEvent;

export function detailFromSnapshot(snapshot: AuctionSnapshot): AuctionDetail {
  return {
    auction: snapshot.auction,
    bids: snapshot.bids,
    server_time: snapshot.server_time,
  };
}

export function applyPlacedBid(
  detail: AuctionDetail,
  auction: Auction,
  bid: Bid,
): AuctionDetail {
  return {
    ...detail,
    auction,
    bids: mergeBid(detail.bids, bid),
  };
}

export function applyAuctionEvent(
  detail: AuctionDetail,
  event: AuctionEvent,
): AuctionDetail {
  if (event.type === "bid") {
    return {
      auction: event.payload.auction,
      bids: mergeBid(detail.bids, event.payload.bid),
      server_time: event.server_time,
    };
  }

  if (event.type === "extended") {
    return {
      ...detail,
      auction: {
        ...detail.auction,
        ends_at: event.payload.ends_at,
        extension_count: event.payload.extension_count,
        version: event.version,
      },
      server_time: event.server_time,
    };
  }

  if (event.type === "ended") {
    return {
      ...detail,
      auction: event.payload.auction,
      server_time: event.server_time,
    };
  }

  return { ...detail, server_time: event.server_time };
}

function mergeBid(bids: Bid[], bid: Bid) {
  if (bids.some((existing) => existing.id === bid.id)) {
    return bids;
  }

  return [bid, ...bids].slice(0, 50);
}

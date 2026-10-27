import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { auctionKeys } from "./query-keys";
import {
  type AuctionEvent,
  type AuctionSnapshot,
  applyAuctionEvent,
  detailFromSnapshot,
} from "./realtime";
import type { AuctionDetail } from "./types";

type AuctionMessage = AuctionSnapshot | AuctionEvent;

export function useAuctionSocket(auctionId: string | null) {
  const queryClient = useQueryClient();
  const [viewerCount, setViewerCount] = useState(0);
  const [outbidAt, setOutbidAt] = useState(0);

  useEffect(() => {
    setViewerCount(0);
    setOutbidAt(0);

    if (!auctionId) {
      return;
    }

    const url = new URL(`/ws/auctions/${auctionId}`, window.location.href);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";

    let socket: WebSocket | null = null;
    let stopped = false;
    let reconnectTimer: number | undefined;
    let attempt = 0;
    let lastSeq = -1;
    let epoch = "";

    const refresh = () => {
      void queryClient.invalidateQueries({ queryKey: auctionKeys.current });
    };

    const connect = () => {
      if (stopped) {
        return;
      }

      socket = new WebSocket(url);
      socket.onopen = () => {
        attempt = 0;
      };
      socket.onmessage = (event) => {
        let message: AuctionMessage;

        try {
          message = JSON.parse(String(event.data)) as AuctionMessage;
        } catch {
          return;
        }

        if (message.type === "snapshot") {
          epoch = message.epoch;
          lastSeq = message.seq;
          setViewerCount(message.viewer_count);
          queryClient.setQueryData<AuctionDetail>(
            auctionKeys.current,
            detailFromSnapshot(message),
          );
          return;
        }

        if (epoch && message.epoch !== epoch) {
          socket?.close();
          return;
        }

        if (lastSeq >= 0 && message.seq > lastSeq + 1) {
          refresh();
        }
        lastSeq = message.seq;

        if (message.type === "presence") {
          setViewerCount(message.payload.viewer_count);
          return;
        }

        if (message.type === "outbid") {
          setOutbidAt(Date.now());
          return;
        }

        queryClient.setQueryData<AuctionDetail>(
          auctionKeys.current,
          (previous) =>
            previous ? applyAuctionEvent(previous, message) : previous,
        );
      };
      socket.onclose = () => {
        if (stopped) {
          return;
        }

        attempt += 1;
        const delay = Math.min(15_000, 2 ** Math.min(attempt, 4) * 1_000);
        reconnectTimer = window.setTimeout(connect, delay);
      };
      socket.onerror = () => socket?.close();
    };

    connect();

    return () => {
      stopped = true;

      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer);
      }

      socket?.close();
    };
  }, [auctionId, queryClient]);

  return { outbidAt, viewerCount };
}

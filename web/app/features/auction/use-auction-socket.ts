import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { authApi } from "~/features/auth/api";
import { auctionKeys } from "./query-keys";
import {
  type AuctionEvent,
  type AuctionSnapshot,
  applyAuctionEvent,
  detailFromSnapshot,
} from "./realtime";
import type { AuctionDetail } from "./types";

type AuctionMessage = AuctionSnapshot | AuctionEvent;

export function useAuctionSocket(
  auctionId: string | null,
  authenticated: boolean,
) {
  const queryClient = useQueryClient();
  const [viewerCount, setViewerCount] = useState(0);
  const [outbidAt, setOutbidAt] = useState(0);
  const [sessionExpired, setSessionExpired] = useState(false);
  const authUnavailableRef = useRef(false);

  useEffect(() => {
    setViewerCount(0);
    setOutbidAt(0);
    setSessionExpired(false);
    authUnavailableRef.current = false;

    if (!auctionId) {
      return;
    }

    const url = new URL(`/ws/auctions/${auctionId}`, window.location.href);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";

    let socket: WebSocket | null = null;
    let stopped = false;
    let connecting = false;
    let reconnectTimer: number | undefined;
    let attempt = 0;
    let lastSeq = -1;
    let epoch = "";

    const refreshAuction = () => {
      void queryClient.invalidateQueries({ queryKey: auctionKeys.current });
    };

    const scheduleReconnect = () => {
      if (stopped || reconnectTimer) {
        return;
      }

      attempt += 1;
      const delay = Math.min(15_000, 2 ** Math.min(attempt, 4) * 1_000);
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = undefined;
        void connect(true);
      }, delay);
    };

    const openSocket = () => {
      try {
        socket = new WebSocket(url);
      } catch {
        scheduleReconnect();
        return;
      }

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
          refreshAuction();
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
        socket = null;

        if (stopped) {
          return;
        }

        scheduleReconnect();
      };
      socket.onerror = () => socket?.close();
    };

    const connect = async (isReconnect: boolean) => {
      if (stopped || connecting) {
        return;
      }

      connecting = true;

      try {
        // The access cookie may have expired while the socket was down.
        if (isReconnect && authenticated && !authUnavailableRef.current) {
          let user = null;

          try {
            user = await authApi.me();
          } catch {
            if (!stopped) {
              scheduleReconnect();
            }
            return;
          }

          if (stopped) {
            return;
          }

          if (!user) {
            authUnavailableRef.current = true;
            setSessionExpired(true);
          } else {
            setSessionExpired(false);
          }
        }

        if (!stopped) {
          openSocket();
        }
      } finally {
        connecting = false;
      }
    };

    void connect(false);

    return () => {
      stopped = true;

      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer);
      }

      socket?.close();
    };
  }, [auctionId, authenticated, queryClient]);

  return { outbidAt, sessionExpired, viewerCount };
}

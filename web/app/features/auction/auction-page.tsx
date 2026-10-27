import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useOutletContext } from "react-router";
import { Button } from "~/components/ui/button";
import type { User } from "~/features/auth/types";
import { ApiError, getErrorMessage } from "~/lib/api-client";
import { auctionApi } from "./api";
import { BidLedger } from "./components/bid-ledger";
import { BidPanel } from "./components/bid-panel";
import { LotSheet } from "./components/lot-sheet";
import { idr } from "./format";
import { auctionKeys } from "./query-keys";
import { applyPlacedBid } from "./realtime";
import { getMinimumBid } from "./rules";
import type { AuctionDetail } from "./types";
import { useAuctionSocket } from "./use-auction-socket";

type BidInput = {
  amount: number;
  auctionId: string;
  idempotencyKey: string;
};

export default function AuctionPage() {
  const { user } = useOutletContext<{ user: User | null }>();
  const queryClient = useQueryClient();
  const [amount, setAmount] = useState(0);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [clockOffset, setClockOffset] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const [priceFlash, setPriceFlash] = useState(false);
  const idempotencyKeyRef = useRef<string | null>(null);
  const previousAuctionIdRef = useRef<string | null>(null);

  const currentQuery = useQuery({
    queryKey: auctionKeys.current,
    queryFn: () => auctionApi.current(),
    enabled: typeof window !== "undefined",
    refetchInterval: (query) => (query.state.data === null ? 15_000 : false),
  });

  const detail = currentQuery.data ?? null;
  const auction = detail?.auction ?? null;
  const auctionId = auction?.id ?? null;
  const { outbidAt, viewerCount } = useAuctionSocket(auctionId);

  const bidMutation = useMutation({
    mutationFn: ({ amount, auctionId, idempotencyKey }: BidInput) =>
      auctionApi.placeBid(auctionId, amount, idempotencyKey),
    onError: (mutationError) => {
      setStatus("");

      if (
        mutationError instanceof ApiError &&
        (mutationError.status === 409 || mutationError.status === 422)
      ) {
        idempotencyKeyRef.current = null;
        void queryClient.invalidateQueries({ queryKey: auctionKeys.current });
      }

      setError(getErrorMessage(mutationError));
    },
    onSuccess: (result) => {
      idempotencyKeyRef.current = null;
      queryClient.setQueryData<AuctionDetail>(
        auctionKeys.current,
        (previous) =>
          previous
            ? applyPlacedBid(previous, result.auction, result.bid)
            : previous,
      );
      setAmount(getMinimumBid(result.auction));
      setPriceFlash(true);
      setError("");
      setStatus(
        result.replay
          ? "Bid ini sudah tercatat sebelumnya."
          : result.extended
            ? "Bid tercatat. Waktu diperpanjang 60 detik."
            : "Bid tercatat. Kamu memegang harga tertinggi.",
      );
    },
  });

  const minimumBid = auction ? getMinimumBid(auction) : 0;
  const endsAt = auction?.ends_at ? Date.parse(auction.ends_at) : null;
  const remaining =
    endsAt === null
      ? 0
      : Math.max(0, Math.ceil((endsAt - (now + clockOffset)) / 1000));
  const isEnded = Boolean(
    auction && (auction.status === "ended" || remaining === 0),
  );
  const isCreator = Boolean(auction && user && auction.created_by === user.id);
  const canBid = Boolean(
    user && auction?.status === "live" && !isCreator && !isEnded,
  );
  const userDisplayName = user ? (user.display_name ?? user.email) : null;
  const highestBidderId =
    auction?.highest_bidder_id ?? auction?.winner_id ?? null;
  const highestBidder = highestBidderId
    ? (detail?.bids.find((bid) => bid.user_id === highestBidderId)
        ?.display_name ?? "")
    : "";
  const serverTime = detail?.server_time;

  useEffect(() => {
    if (!serverTime) {
      return;
    }

    setClockOffset(Date.parse(serverTime) - Date.now());
  }, [serverTime]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!auction || previousAuctionIdRef.current === auction.id) {
      return;
    }

    previousAuctionIdRef.current = auction.id;
    setAmount(getMinimumBid(auction));
    setError("");
    setStatus("");
    idempotencyKeyRef.current = null;
  }, [auction]);

  useEffect(() => {
    if (amount >= minimumBid) {
      return;
    }

    setAmount(minimumBid);
    idempotencyKeyRef.current = null;
  }, [amount, minimumBid]);

  const currentPrice = auction?.current_price;

  useEffect(() => {
    if (currentPrice === undefined) {
      return;
    }

    setPriceFlash(true);
    const timer = window.setTimeout(() => setPriceFlash(false), 700);
    return () => window.clearTimeout(timer);
  }, [currentPrice]);

  useEffect(() => {
    if (!outbidAt) {
      return;
    }

    setError("");
    setStatus("Kamu telah di-outbid. Pasang bid baru untuk memimpin.");
  }, [outbidAt]);

  function handleAmountChange(value: number) {
    idempotencyKeyRef.current = null;
    setAmount(Number.isSafeInteger(value) ? value : 0);
    setError("");
    setStatus("");
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!auction || !canBid || bidMutation.isPending) {
      return;
    }

    if (amount < minimumBid) {
      setError(`Bid minimum ${idr.format(minimumBid)}.`);
      return;
    }

    const idempotencyKey = idempotencyKeyRef.current ?? crypto.randomUUID();
    idempotencyKeyRef.current = idempotencyKey;
    setError("");
    setStatus("");

    bidMutation.mutate({
      amount,
      auctionId: auction.id,
      idempotencyKey,
    });
  }

  if (currentQuery.isPending) {
    return (
      <p className="border border-rule bg-surface px-3.5 py-3 text-sm">
        Memuat lelang...
      </p>
    );
  }

  if (currentQuery.isError) {
    return (
      <div
        className="border border-accent bg-paper px-3.5 py-3 text-sm"
        role="alert"
      >
        <p className="font-semibold">Gagal memuat lelang.</p>
        <p className="mt-1 text-muted">{getErrorMessage(currentQuery.error)}</p>
        <Button
          className="mt-3"
          onClick={() => void currentQuery.refetch()}
          size="sm"
          type="button"
          variant="secondary"
        >
          Coba lagi
        </Button>
      </div>
    );
  }

  if (!auction || !detail) {
    return (
      <section
        aria-labelledby="no-auction-title"
        className="py-[clamp(28px,5vw,56px)]"
      >
        <p className="kicker">Ruang lelang</p>
        <h1
          className="font-serif text-[clamp(2.3rem,5vw,3.6rem)] font-normal leading-none"
          id="no-auction-title"
        >
          Belum ada lelang live
        </h1>
        <p className="mt-3 max-w-[48ch] text-[0.95rem] text-muted">
          Admin belum memulai lelang. Halaman ini akan mencoba memuat ulang
          secara berkala.
        </p>
        <Link
          className="mt-5 inline-flex min-h-11 items-center justify-center rounded-xs border border-ink px-3 py-2 text-sm font-medium text-ink hover:bg-ink hover:text-paper"
          to="/results"
        >
          Lihat hasil lelang
        </Link>
      </section>
    );
  }

  return (
    <>
      <LotSheet auction={auction} />

      <section
        aria-labelledby="auction-heading"
        className="grid gap-gutter py-[clamp(28px,5vw,56px)] lg:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]"
      >
        <h2 className="sr-only" id="auction-heading">
          Ruang bid
        </h2>

        <BidPanel
          amount={amount}
          bidCount={detail.bids.length}
          canBid={canBid}
          currentPrice={auction.current_price}
          durationSeconds={auction.duration_seconds}
          error={error}
          extensionCount={auction.extension_count ?? 0}
          finalPrice={auction.final_price ?? null}
          highestBidder={highestBidder}
          increment={auction.min_increment}
          isCreator={isCreator}
          isEnded={isEnded}
          isSubmitting={bidMutation.isPending}
          minimumBid={minimumBid}
          onAmountChange={handleAmountChange}
          onSubmit={handleSubmit}
          priceFlash={priceFlash}
          remaining={remaining}
          status={status}
          userDisplayName={userDisplayName}
          viewerCount={viewerCount}
        />

        <BidLedger bids={detail.bids} />
      </section>
    </>
  );
}

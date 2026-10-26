import { type FormEvent, useEffect, useState } from "react";
import { BidLedger } from "./components/bid-ledger";
import { BidPanel } from "./components/bid-panel";
import { LotSheet } from "./components/lot-sheet";
import {
  ANTI_SNIPE_SECONDS,
  DEMO_DURATION_SECONDS,
  formatTime,
  getMinimumBid,
  idr,
  initialAuction,
  MAX_EXTENSIONS,
} from "./demo";

export default function AuctionPage() {
  const [auction, setAuction] = useState(initialAuction);
  const [amount, setAmount] = useState(() => getMinimumBid(initialAuction));
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [endsAt, setEndsAt] = useState<number | null>(null);
  const [remaining, setRemaining] = useState(DEMO_DURATION_SECONDS);
  const [extensionCount, setExtensionCount] = useState(0);
  const [isEnded, setIsEnded] = useState(false);
  const [priceFlash, setPriceFlash] = useState(false);

  useEffect(() => {
    setEndsAt(Date.now() + DEMO_DURATION_SECONDS * 1000);
  }, []);

  useEffect(() => {
    if (endsAt === null) {
      return;
    }

    const update = () => {
      const next = Math.max(0, Math.ceil((endsAt - Date.now()) / 1000));
      setRemaining(next);
      if (next === 0) {
        setIsEnded(true);
      }
    };

    update();
    const timer = window.setInterval(update, 1000);
    return () => window.clearInterval(timer);
  }, [endsAt]);

  useEffect(() => {
    if (!priceFlash) {
      return;
    }

    const timer = window.setTimeout(() => setPriceFlash(false), 700);
    return () => window.clearTimeout(timer);
  }, [priceFlash]);

  const minimumBid = getMinimumBid(auction);

  function handleAmountChange(value: number) {
    setAmount(Number.isSafeInteger(value) ? value : 0);
    setError("");
    setStatus("");
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (isEnded || remaining === 0) {
      setIsEnded(true);
      setError("Lelang sudah selesai.");
      return;
    }

    if (!Number.isSafeInteger(amount) || amount <= 0) {
      setError("Masukkan nominal bid yang valid.");
      return;
    }

    if (amount < minimumBid) {
      setError(`Bid minimum ${idr.format(minimumBid)}.`);
      return;
    }

    const newBid = {
      id: Math.max(0, ...auction.bids.map((bid) => bid.id)) + 1,
      name: "Kamu (demo)",
      amount,
      at: formatTime(),
    };

    setAuction((previous) => ({
      ...previous,
      bids: [newBid, ...previous.bids],
      currentPrice: amount,
      highestBidder: newBid.name,
    }));
    setAmount(amount + auction.minIncrement);
    setError("");
    setPriceFlash(true);

    const extended =
      endsAt !== null &&
      endsAt - Date.now() <= ANTI_SNIPE_SECONDS * 1000 &&
      extensionCount < MAX_EXTENSIONS;

    if (extended) {
      setEndsAt((previous) => (previous === null ? null : previous + 60_000));
      setExtensionCount((previous) => previous + 1);
      setStatus(
        "Bid tercatat. Waktu diperpanjang 60 detik. Kamu memegang harga tertinggi.",
      );
      return;
    }

    setStatus("Bid tercatat. Kamu memegang harga tertinggi.");
  }

  return (
    <>
      <p className="mt-4 border border-rule bg-surface px-3.5 py-3 text-sm">
        <strong>Data demo.</strong> Harga, countdown, dan bid diproses lokal di
        browser. Tidak ada data yang dikirim ke server.
      </p>

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
          bidCount={auction.bids.length}
          currentPrice={auction.currentPrice}
          error={error}
          extensionCount={extensionCount}
          highestBidder={auction.highestBidder}
          increment={auction.minIncrement}
          isEnded={isEnded}
          minimumBid={minimumBid}
          onAmountChange={handleAmountChange}
          onSubmit={handleSubmit}
          priceFlash={priceFlash}
          remaining={remaining}
          status={status}
          viewerCount={auction.viewerCount}
        />

        <BidLedger bids={auction.bids} />
      </section>
    </>
  );
}

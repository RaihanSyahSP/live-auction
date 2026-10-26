import { useState } from "react";
import { AdminAuctionCard } from "./components/admin-auction-card";
import { DraftForm } from "./components/draft-form";
import {
  type AdminAuction,
  type DraftInput,
  initialAdminAuctions,
  sortAuctions,
} from "./demo";

type Feedback = {
  kind: "error" | "info";
  message: string;
};

export default function AdminPage() {
  const [auctions, setAuctions] = useState(initialAdminAuctions);
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  const liveAuction = auctions.find((auction) => auction.status === "live");
  const hasLive = Boolean(liveAuction);
  const sortedAuctions = sortAuctions(auctions);
  const liveCount = auctions.filter(
    (auction) => auction.status === "live",
  ).length;
  const draftCount = auctions.filter(
    (auction) => auction.status === "draft",
  ).length;
  const endedCount = auctions.filter(
    (auction) => auction.status === "ended",
  ).length;

  function handleStart(id: number) {
    if (hasLive) {
      setFeedback({
        kind: "error",
        message: "Masih ada lelang live. Akhiri lelang itu dulu.",
      });
      return;
    }

    setAuctions((previous) =>
      previous.map((auction) =>
        auction.id === id ? { ...auction, status: "live" } : auction,
      ),
    );
    setFeedback({
      kind: "info",
      message: "Lelang dimulai. Buka ruang lelang untuk melihatnya.",
    });
  }

  function handleEnd(id: number) {
    if (
      !window.confirm(
        "Akhiri lelang sekarang? Pemenang ditentukan dari bid tertinggi.",
      )
    ) {
      return;
    }

    setAuctions((previous) =>
      previous.map((auction) =>
        auction.id === id
          ? {
              ...auction,
              status: "ended",
              winner: auction.highestBidder || "Tidak terjual",
              finalPrice: auction.bids.length > 0 ? auction.currentPrice : null,
            }
          : auction,
      ),
    );
    setFeedback({ kind: "info", message: "Lelang diakhiri." });
  }

  function handleDelete(id: number) {
    if (!window.confirm("Hapus draft ini?")) {
      return;
    }

    setAuctions((previous) => previous.filter((auction) => auction.id !== id));
    setFeedback({ kind: "info", message: "Draft dihapus." });
  }

  function handleCreate(input: DraftInput) {
    const nextId = Math.max(0, ...auctions.map((auction) => auction.id)) + 1;
    const nextLotNumber =
      auctions.reduce((max, auction) => {
        return Math.max(max, Number.parseInt(auction.lot, 10) || 0);
      }, 0) + 1;
    const { durationMinutes, ...rest } = input;
    const draft: AdminAuction = {
      ...rest,
      id: nextId,
      lot: String(nextLotNumber).padStart(3, "0"),
      currentPrice: input.startingPrice,
      highestBidder: "",
      bids: [],
      viewerCount: 0,
      status: "draft",
      durationSeconds: durationMinutes * 60,
      winner: "",
      finalPrice: null,
    };

    setAuctions((previous) => [...previous, draft]);
  }

  return (
    <>
      <p className="mt-4 border border-rule bg-surface px-3.5 py-3 text-sm">
        <strong>Mode demo admin.</strong> Perubahan di halaman ini hanya
        tersimpan di memori browser dan tidak mengubah halaman lelang publik.
      </p>

      <div className="grid items-start gap-gutter py-[clamp(28px,5vw,56px)] lg:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
        <section aria-labelledby="auction-list-title">
          <div className="flex flex-col gap-4 border-b-2 border-ink pb-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="kicker">Ruang kerja</p>
              <h1
                className="font-serif text-[clamp(2.2rem,4vw,3.4rem)] font-normal leading-none"
                id="auction-list-title"
              >
                Lelang
              </h1>
              <p className="mt-2 text-[0.82rem] text-muted tabular-nums">
                {liveCount} live · {draftCount} draft · {endedCount} selesai
              </p>
            </div>
            <a
              className="inline-flex min-h-11 items-center justify-center rounded-xs border border-ink px-3 py-2 text-sm font-medium text-ink hover:bg-ink hover:text-paper"
              href="#draft-form"
            >
              Buat draft
            </a>
          </div>

          <p className="mt-3 text-[0.78rem] text-muted">
            Hanya satu lelang live pada satu waktu. Akhiri lelang live sebelum
            memulai draft lain.
          </p>

          {feedback ? (
            <p
              className={
                feedback.kind === "error"
                  ? "field-error"
                  : "mt-3 border border-rule bg-paper px-3 py-2 text-sm"
              }
              role={feedback.kind === "error" ? "alert" : "status"}
            >
              {feedback.message}
            </p>
          ) : null}

          {sortedAuctions.length > 0 ? (
            <div className="mt-4 grid gap-3">
              {sortedAuctions.map((auction) => (
                <AdminAuctionCard
                  auction={auction}
                  hasLive={hasLive}
                  key={auction.id}
                  onDelete={handleDelete}
                  onEnd={handleEnd}
                  onStart={handleStart}
                />
              ))}
            </div>
          ) : (
            <div className="mt-4 border border-rule bg-surface p-5 text-[0.9rem] text-muted">
              <p>Belum ada lelang.</p>
              <a className="text-link mt-2" href="#draft-form">
                Buat draft pertama
              </a>
            </div>
          )}
        </section>

        <section
          aria-labelledby="draft-title"
          className="lg:sticky lg:top-6"
          id="draft-form"
        >
          <DraftForm onCreate={handleCreate} />
        </section>
      </div>
    </>
  );
}

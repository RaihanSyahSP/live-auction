import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "~/components/ui/button";
import { auctionApi } from "~/features/auction/api";
import { getErrorMessage } from "~/lib/api-client";
import { AdminAuctionCard } from "./components/admin-auction-card";
import { DraftForm } from "./components/draft-form";
import { type DraftInput, sortAuctions, toAdminAuction } from "./model";

const auctionsKey = ["admin", "auctions"] as const;

type AdminAction = {
  type: "start" | "end" | "delete";
  id: string;
};

type Feedback = {
  kind: "error" | "info";
  message: string;
};

export default function AdminPage() {
  const queryClient = useQueryClient();
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  const auctionsQuery = useQuery({
    queryKey: auctionsKey,
    queryFn: async () => (await auctionApi.listAdmin()).map(toAdminAuction),
    enabled: typeof window !== "undefined",
  });

  const actionMutation = useMutation({
    mutationFn: async ({ type, id }: AdminAction) => {
      if (type === "start") {
        await auctionApi.startAdmin(id);
        return;
      }

      if (type === "end") {
        await auctionApi.endAdmin(id);
        return;
      }

      await auctionApi.deleteAdmin(id);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auctionsKey }),
  });

  const createMutation = useMutation({
    mutationFn: (input: DraftInput) => auctionApi.createAdmin(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auctionsKey }),
  });

  const auctions = auctionsQuery.data ?? [];
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

  async function handleStart(id: string) {
    if (hasLive) {
      setFeedback({
        kind: "error",
        message: "Masih ada lelang live. Akhiri lelang itu dulu.",
      });
      return;
    }

    setFeedback(null);

    try {
      await actionMutation.mutateAsync({ type: "start", id });
      setFeedback({
        kind: "info",
        message: "Lelang dimulai. Buka ruang lelang untuk melihatnya.",
      });
    } catch (error) {
      setFeedback({ kind: "error", message: getErrorMessage(error) });
    }
  }

  async function handleEnd(id: string) {
    if (
      !window.confirm(
        "Akhiri lelang sekarang? Pemenang ditentukan dari bid tertinggi.",
      )
    ) {
      return;
    }

    setFeedback(null);

    try {
      await actionMutation.mutateAsync({ type: "end", id });
      setFeedback({ kind: "info", message: "Lelang diakhiri." });
    } catch (error) {
      setFeedback({ kind: "error", message: getErrorMessage(error) });
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm("Hapus draft ini?")) {
      return;
    }

    setFeedback(null);

    try {
      await actionMutation.mutateAsync({ type: "delete", id });
      setFeedback({ kind: "info", message: "Draft dihapus." });
    } catch (error) {
      setFeedback({ kind: "error", message: getErrorMessage(error) });
    }
  }

  async function handleCreate(input: DraftInput) {
    await createMutation.mutateAsync(input);
  }

  return (
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

        {auctionsQuery.isPending ? (
          <p className="mt-4 border border-rule bg-surface px-3.5 py-3 text-sm">
            Memuat lelang...
          </p>
        ) : null}

        {auctionsQuery.isError ? (
          <div
            className="mt-4 border border-accent bg-paper px-3.5 py-3 text-sm"
            role="alert"
          >
            <p className="font-semibold">Gagal memuat lelang.</p>
            <p className="mt-1 text-muted">
              {getErrorMessage(auctionsQuery.error)}
            </p>
            <Button
              className="mt-3"
              onClick={() => void auctionsQuery.refetch()}
              size="sm"
              type="button"
              variant="secondary"
            >
              Coba lagi
            </Button>
          </div>
        ) : null}

        {auctionsQuery.isSuccess ? (
          sortedAuctions.length > 0 ? (
            <div className="mt-4 grid gap-3">
              {sortedAuctions.map((auction) => (
                <AdminAuctionCard
                  auction={auction}
                  busy={actionMutation.isPending}
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
          )
        ) : null}
      </section>

      <section
        aria-labelledby="draft-title"
        className="lg:sticky lg:top-6"
        id="draft-form"
      >
        <DraftForm onCreate={handleCreate} />
      </section>
    </div>
  );
}

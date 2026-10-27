import { Link } from "react-router";
import { Button } from "~/components/ui/button";
import { Card } from "~/components/ui/card";
import { idr } from "~/features/auction/demo";
import type { AdminAuction } from "../model";
import { formatDuration } from "../model";

type AdminAuctionCardProps = {
  auction: AdminAuction;
  busy: boolean;
  hasLive: boolean;
  onDelete: (id: string) => void | Promise<void>;
  onEnd: (id: string) => void | Promise<void>;
  onStart: (id: string) => void | Promise<void>;
};

export function AdminAuctionCard({
  auction,
  busy,
  hasLive,
  onDelete,
  onEnd,
  onStart,
}: AdminAuctionCardProps) {
  const isLive = auction.status === "live";
  const isDraft = auction.status === "draft";
  const isEnded = auction.status === "ended";
  const statusLabel = isLive ? "LIVE" : isDraft ? "DRAFT" : "SELESAI";

  return (
    <Card
      className={
        isLive
          ? "border-ink p-[18px]"
          : isEnded
            ? "border-rule-soft p-[18px]"
            : "border-rule p-[18px]"
      }
    >
      <div className="flex items-center justify-between gap-3">
        <p
          className={
            isLive
              ? "text-[0.7rem] font-bold tracking-[0.08em] text-accent"
              : "text-[0.7rem] font-bold tracking-[0.08em] text-muted"
          }
        >
          {statusLabel}
        </p>
        <p className="text-[0.7rem] tracking-[0.08em] text-muted">
          LOT {auction.lot}
        </p>
      </div>

      <h2 className="mt-2.5 font-serif text-[1.55rem] font-normal leading-[1.15]">
        {auction.title}
      </h2>
      {auction.subtitle ? (
        <p className="mt-0.5 text-[0.9rem] italic text-muted">
          {auction.subtitle}
        </p>
      ) : null}

      <dl className="mt-3.5 grid grid-cols-2 gap-2 border-t border-rule-soft pt-3">
        <div>
          <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
            Harga
          </dt>
          <dd className="mt-0.5 text-[0.88rem] tabular-nums">
            {idr.format(auction.currentPrice)}
          </dd>
        </div>
        <div>
          <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
            Kelipatan
          </dt>
          <dd className="mt-0.5 text-[0.88rem] tabular-nums">
            {idr.format(auction.minIncrement)}
          </dd>
        </div>

        {isDraft ? (
          <div className="col-span-2">
            <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
              Durasi
            </dt>
            <dd className="mt-0.5 text-[0.88rem]">
              {formatDuration(auction.durationSeconds)}
            </dd>
          </div>
        ) : null}

        {isLive ? (
          <div className="col-span-2">
            <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
              Status
            </dt>
            <dd className="mt-0.5 text-[0.88rem]">
              Sedang live · {formatDuration(auction.durationSeconds)}
            </dd>
          </div>
        ) : null}

        {isEnded ? (
          <>
            <div>
              <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
                Pemenang
              </dt>
              <dd
                className="mt-0.5 text-[0.88rem]"
                title={auction.winnerId ?? undefined}
              >
                {auction.winnerId ? "Sudah ditetapkan" : "Tidak terjual"}
              </dd>
            </div>
            <div>
              <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
                Harga akhir
              </dt>
              <dd className="mt-0.5 text-[0.88rem] tabular-nums">
                {auction.finalPrice === null
                  ? "Tidak ada"
                  : idr.format(auction.finalPrice)}
              </dd>
            </div>
          </>
        ) : null}
      </dl>

      <div className="mt-4 flex flex-wrap gap-2">
        {isDraft ? (
          <>
            <Button
              disabled={busy || hasLive}
              onClick={() => onStart(auction.id)}
              size="sm"
              title={hasLive ? "Akhiri lelang live dulu." : undefined}
              type="button"
              variant="secondary"
            >
              Start lelang
            </Button>
            <Button
              disabled={busy}
              onClick={() => onDelete(auction.id)}
              size="sm"
              type="button"
              variant="danger"
            >
              Hapus
            </Button>
          </>
        ) : null}

        {isLive ? (
          <>
            <Link
              className="inline-flex min-h-11 items-center justify-center rounded-xs border border-ink px-3 py-2 text-sm font-medium text-ink hover:bg-ink hover:text-paper"
              to="/"
            >
              Buka ruang
            </Link>
            <Button
              disabled={busy}
              onClick={() => onEnd(auction.id)}
              size="sm"
              type="button"
              variant="danger"
            >
              End lelang
            </Button>
          </>
        ) : null}
      </div>
    </Card>
  );
}

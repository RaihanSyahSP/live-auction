import { idr } from "~/features/auction/format";
import type { DemoResult } from "../demo";

const dateTimeFormat = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "Asia/Jakarta",
});

type ResultCardProps = {
  result: DemoResult;
};

export function ResultCard({ result }: ResultCardProps) {
  return (
    <article className="border border-rule bg-surface p-[18px]">
      <div className="flex items-center justify-between gap-3">
        <p className="text-[0.7rem] font-bold tracking-[0.08em] text-muted">
          SELESAI
        </p>
        <p className="text-[0.7rem] tracking-[0.08em] text-muted">
          LOT {result.lot}
        </p>
      </div>

      <h2 className="mt-2.5 font-serif text-[1.55rem] font-normal leading-[1.15]">
        {result.title}
      </h2>
      {result.subtitle ? (
        <p className="mt-0.5 text-[0.9rem] italic text-muted">
          {result.subtitle}
        </p>
      ) : null}

      <dl className="mt-3.5 grid grid-cols-2 gap-2 border-t border-rule-soft pt-3">
        <div>
          <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
            Pemenang
          </dt>
          <dd className="mt-0.5 text-[0.88rem]">{result.winner}</dd>
        </div>
        <div>
          <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
            Harga akhir
          </dt>
          <dd className="mt-0.5 text-[0.88rem] tabular-nums">
            {result.finalPrice === null
              ? "Tidak ada"
              : idr.format(result.finalPrice)}
          </dd>
        </div>
        <div>
          <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
            Jumlah bid
          </dt>
          <dd className="mt-0.5 text-[0.88rem]">{result.bidCount} bid</dd>
        </div>
        <div>
          <dt className="text-[0.65rem] uppercase tracking-[0.08em] text-muted">
            Selesai
          </dt>
          <dd className="mt-0.5 text-[0.88rem]">
            {dateTimeFormat.format(new Date(result.endedAt))}
          </dd>
        </div>
      </dl>
    </article>
  );
}

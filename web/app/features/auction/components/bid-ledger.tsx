import type { Bid } from "../demo";
import { idr } from "../demo";

type BidLedgerProps = {
  bids: Bid[];
};

export function BidLedger({ bids }: BidLedgerProps) {
  return (
    <aside aria-labelledby="ledger-heading" className="min-w-0">
      <div className="flex items-baseline justify-between gap-4 border-b-2 border-ink pb-2.5">
        <h2
          className="font-serif text-[1.7rem] font-normal leading-normal"
          id="ledger-heading"
        >
          Riwayat bid
        </h2>
        <p className="text-[0.8rem] text-muted tabular-nums">
          {bids.length} bid
        </p>
      </div>

      {bids.length > 0 ? (
        <ol className="max-h-[480px] overflow-y-auto">
          {bids.map((bid) => (
            <li
              className="grid grid-cols-[64px_minmax(0,1fr)_auto] items-baseline gap-3 border-b border-rule-soft py-3"
              key={bid.id}
            >
              <span className="text-[0.75rem] text-muted tabular-nums">
                {bid.at}
              </span>
              <span className="break-words text-[0.92rem]">{bid.name}</span>
              <span className="text-[0.95rem] font-semibold tabular-nums">
                {idr.format(bid.amount)}
              </span>
            </li>
          ))}
        </ol>
      ) : (
        <p className="border-b border-rule-soft py-[18px] text-[0.9rem] text-muted">
          Belum ada bid. Bid pertama akan muncul di sini.
        </p>
      )}

      <p className="mt-3 text-[0.75rem] text-muted">
        Urutan terbaru di atas. Waktu memakai jam perangkat ini.
      </p>
    </aside>
  );
}

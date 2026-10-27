import { formatLot } from "../format";
import type { Auction } from "../types";

type LotSheetProps = {
  auction: Auction;
};

export function LotSheet({ auction }: LotSheetProps) {
  return (
    <section
      aria-labelledby="lot-title"
      className="grid gap-gutter border-b border-rule-soft py-[clamp(28px,5vw,56px)] lg:grid-cols-[minmax(0,1.04fr)_minmax(0,0.96fr)]"
    >
      <div>
        <figure className="relative flex aspect-[4/3] min-h-[360px] flex-col items-center justify-center border border-rule bg-[#ede4d5] p-8 max-sm:aspect-[5/4] max-sm:min-h-0 max-sm:p-6">
          <span className="absolute top-3.5 left-4 border border-rule px-1.5 py-0.5 text-[0.68rem] tracking-[0.08em] text-muted">
            {formatLot(auction.lot)}
          </span>
          <svg
            aria-hidden="true"
            className="w-[min(82%,520px)] text-[#7a6f60]"
            fill="none"
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            viewBox="0 0 640 420"
          >
            <rect height="216" rx="10" width="448" x="96" y="128" />
            <path d="M150 128 L178 88 H268 L296 128" />
            <circle cx="420" cy="236" r="82" />
            <circle cx="420" cy="236" r="54" />
            <circle cx="420" cy="236" r="20" />
            <rect height="34" rx="4" width="76" x="132" y="164" />
            <circle cx="170" cy="100" r="7" />
            <path d="M520 168 h12" />
          </svg>
          <figcaption className="absolute right-4 bottom-3.5 left-4 text-[0.75rem] text-muted">
            Foto unit belum tersedia. Prototipe memakai ilustrasi.
          </figcaption>
        </figure>

        <div className="mt-3.5 grid gap-2 border-t border-rule-soft pt-3.5 text-[0.9rem]">
          <p>
            <span className="inline-block min-w-[88px] text-label uppercase text-muted">
              Kondisi
            </span>{" "}
            {auction.condition}
          </p>
          <p>
            <span className="inline-block min-w-[88px] text-label uppercase text-muted">
              Kelengkapan
            </span>{" "}
            {auction.includes.join(", ")}
          </p>
        </div>
      </div>

      <div>
        <p className="kicker">Elektronik bekas</p>
        <h1
          className="max-w-[12ch] font-serif text-display font-normal"
          id="lot-title"
        >
          {auction.title}{" "}
          {auction.subtitle ? (
            <span className="mt-2.5 block text-[0.48em] tracking-normal text-muted italic">
              {auction.subtitle}
            </span>
          ) : null}
        </h1>
        <p className="mt-6 max-w-[52ch] text-[1.02rem] leading-[1.65]">
          {auction.description}
        </p>

        <dl className="mt-7 border-t border-rule-soft">
          {auction.specs.map((spec) => (
            <div
              className="grid grid-cols-[minmax(100px,0.4fr)_1fr] gap-4 border-b border-rule-soft py-2.5"
              key={spec.label}
            >
              <dt className="text-label uppercase text-muted">{spec.label}</dt>
              <dd className="text-[0.92rem]">{spec.value}</dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}

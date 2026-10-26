import type { FormEvent } from "react";
import { Button } from "~/components/ui/button";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "~/components/ui/field";
import { Input } from "~/components/ui/input";
import { cn } from "~/lib/utils";
import {
  ANTI_SNIPE_SECONDS,
  DEMO_DURATION_SECONDS,
  formatRemaining,
  idr,
  MAX_EXTENSIONS,
  numberFormat,
} from "../demo";

type BidPanelProps = {
  amount: number;
  bidCount: number;
  currentPrice: number;
  error: string;
  extensionCount: number;
  highestBidder: string;
  increment: number;
  isEnded: boolean;
  minimumBid: number;
  onAmountChange: (value: number) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  priceFlash: boolean;
  remaining: number;
  status: string;
  viewerCount: number;
};

export function BidPanel({
  amount,
  bidCount,
  currentPrice,
  error,
  extensionCount,
  highestBidder,
  increment,
  isEnded,
  minimumBid,
  onAmountChange,
  onSubmit,
  priceFlash,
  remaining,
  status,
  viewerCount,
}: BidPanelProps) {
  const quickAmounts = [
    minimumBid,
    minimumBid + increment,
    minimumBid + increment * 4,
  ];
  const totalSeconds = DEMO_DURATION_SECONDS + extensionCount * 60;
  const percent = Math.max(0, Math.min(100, (remaining / totalSeconds) * 100));
  const countdown = isEnded ? "Selesai" : formatRemaining(remaining);

  return (
    <div className="lg:sticky lg:top-6">
      <div className="border border-ink bg-surface p-[clamp(20px,3vw,32px)]">
        <div className="flex items-start justify-between gap-4">
          <p className="text-label uppercase text-muted">Harga saat ini</p>
          <span className="inline-flex items-center gap-1.5 border border-accent px-2 py-1 text-[0.68rem] font-bold uppercase tracking-[0.08em] text-accent">
            <span aria-hidden="true" className="h-[7px] w-[7px] bg-current" />
            {isEnded ? "Selesai" : "Live"}
          </span>
        </div>

        <p
          aria-atomic="true"
          aria-live="polite"
          className={cn(
            "mt-1.5 inline-block text-price font-semibold tabular-nums transition-colors duration-700",
            priceFlash && "bg-accent-tint",
          )}
        >
          {idr.format(currentPrice)}
        </p>
        <p className="text-[0.85rem] text-muted">
          {highestBidder} · {bidCount} bid · {viewerCount} penonton
        </p>

        <div className="mt-6 flex items-end justify-between gap-4 border-t border-rule-soft pt-4">
          <div>
            <p className="text-label uppercase text-muted">Sisa waktu</p>
            <p
              aria-label={
                isEnded
                  ? "Lelang selesai"
                  : `Sisa waktu ${formatRemaining(remaining)}`
              }
              aria-live="off"
              className="mt-1 text-[2rem] font-semibold tracking-[0.02em] tabular-nums"
              role="timer"
            >
              {countdown}
            </p>
          </div>
          {extensionCount > 0 ? (
            <p className="max-w-[20ch] text-right text-[0.78rem] text-accent">
              Diperpanjang {extensionCount} dari {MAX_EXTENSIONS} kali
            </p>
          ) : null}
        </div>
        <div aria-hidden="true" className="mt-2.5 h-[3px] bg-rule-soft">
          <span
            className="block h-full bg-accent transition-[width] duration-[250ms]"
            style={{ width: `${percent}%` }}
          />
        </div>

        <div className="mt-6 border-t border-rule-soft pt-4">
          <p className="text-label uppercase text-muted">Identitas penawar</p>
          <p className="mt-1.5 text-[0.95rem]">
            Bid sebagai <strong>Kamu (demo)</strong>
          </p>
        </div>

        {isEnded ? (
          <div className="mt-6 border-t border-rule-soft pt-4">
            <p className="text-label uppercase text-muted">Hasil lelang</p>
            <p className="mt-2 font-serif text-[1.45rem] leading-[1.25]">
              {highestBidder || "Tidak ada bid"} memenangkan lot ini.
            </p>
            <p className="mt-2 text-[1rem] tabular-nums">
              Harga akhir {idr.format(currentPrice)}
            </p>
          </div>
        ) : (
          <form
            className="mt-6 border-t border-rule-soft pt-4"
            noValidate
            onSubmit={onSubmit}
          >
            <Field>
              <FieldLabel htmlFor="amount-input">Bid kamu</FieldLabel>
              <div className="flex items-center rounded-xs border-[1.5px] border-rule bg-white focus-within:border-ink focus-within:[outline:3px_solid_var(--color-ink)] focus-within:outline-offset-2">
                <span className="pl-3 text-[0.9rem] text-muted">Rp</span>
                <Input
                  aria-describedby={
                    error ? "amount-help bid-error" : "amount-help"
                  }
                  autoComplete="off"
                  className="min-h-11 min-w-0 flex-1 rounded-none border-0 bg-transparent px-2 py-2.5 text-[1rem] tabular-nums focus-visible:outline-none"
                  id="amount-input"
                  inputMode="numeric"
                  name="amount"
                  onChange={(event) => {
                    const digits = event.target.value
                      .replace(/\D/g, "")
                      .slice(0, 13);
                    onAmountChange(digits ? Number(digits) : 0);
                  }}
                  onFocus={(event) => event.currentTarget.select()}
                  type="text"
                  value={amount > 0 ? numberFormat.format(amount) : ""}
                />
              </div>
              <FieldDescription className="text-[0.78rem]" id="amount-help">
                Bid minimum{" "}
                <span className="tabular-nums">{idr.format(minimumBid)}</span>.
                Kelipatan{" "}
                <span className="tabular-nums">{idr.format(increment)}</span>.
              </FieldDescription>

              <fieldset className="mt-3">
                <legend className="sr-only">Pilihan bid cepat</legend>
                <div className="grid grid-cols-3 gap-2">
                  {quickAmounts.map((value) => (
                    <Button
                      className="w-full px-1 text-[0.78rem] tabular-nums"
                      key={value}
                      onClick={() => onAmountChange(value)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      {numberFormat.format(value)}
                    </Button>
                  ))}
                </div>
              </fieldset>

              <Button className="mt-3 w-full" type="submit">
                Pasang bid
              </Button>

              <div aria-live="polite">
                {error ? <FieldError id="bid-error">{error}</FieldError> : null}
                {status ? (
                  <FieldDescription
                    className="mt-2 border border-rule bg-paper px-2.5 py-2 text-ink"
                    role="status"
                  >
                    {status}
                  </FieldDescription>
                ) : null}
              </div>
            </Field>
          </form>
        )}

        <p className="mt-4 border-t border-rule-soft pt-3 text-xs text-muted">
          Mode demo lokal. Bid tidak dikirim ke server. Bid di{" "}
          {ANTI_SNIPE_SECONDS} detik terakhir memperpanjang sesi 60 detik,
          maksimal {MAX_EXTENSIONS} kali.
        </p>
      </div>
    </div>
  );
}

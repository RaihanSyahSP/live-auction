import type { ReactNode } from "react";

type SiteHeaderProps = {
  note: string;
  children?: ReactNode;
};

export function SiteHeader({ note, children }: SiteHeaderProps) {
  return (
    <header className="flex flex-col gap-3.5 border-b-2 border-ink pt-7 pb-[18px] sm:flex-row sm:items-end sm:justify-between sm:gap-6">
      <div>
        <p className="font-serif text-[clamp(1.9rem,4vw,2.7rem)] leading-none tracking-[0.04em]">
          LELANG
        </p>
        <p className="mt-1.5 text-xs text-muted">{note}</p>
      </div>

      {children ? (
        <div className="flex flex-wrap items-center justify-start gap-3.5 sm:justify-end">
          {children}
        </div>
      ) : null}
    </header>
  );
}

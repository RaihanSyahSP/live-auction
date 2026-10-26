import type { ReactNode } from "react";

type SiteFooterProps = {
  children: ReactNode;
  actions?: ReactNode;
};

export function SiteFooter({ children, actions }: SiteFooterProps) {
  return (
    <footer className="flex flex-col gap-8 border-t-2 border-ink pt-6 pb-10 sm:flex-row sm:items-start sm:justify-between">
      <div className="max-w-[62ch] text-xs leading-normal text-muted">
        {children}
      </div>

      {actions ? <div className="shrink-0">{actions}</div> : null}
    </footer>
  );
}

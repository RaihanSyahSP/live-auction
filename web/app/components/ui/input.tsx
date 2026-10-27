import type { ComponentPropsWithRef } from "react";
import { cn } from "~/lib/utils";

export function Input({
  className,
  type,
  ...props
}: ComponentPropsWithRef<"input">) {
  return (
    <input
      className={cn(
        "block min-h-11 w-full rounded-xs border-[1.5px] border-rule bg-white px-3 py-2.5 text-base text-ink transition-colors placeholder:text-muted focus-visible:border-ink disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-accent",
        className,
      )}
      type={type}
      {...props}
    />
  );
}

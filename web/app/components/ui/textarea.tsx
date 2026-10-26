import type { TextareaHTMLAttributes } from "react";
import { cn } from "~/lib/utils";

export function Textarea({
  className,
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      className={cn(
        "block min-h-[110px] w-full resize-y rounded-xs border-[1.5px] border-rule bg-white px-3 py-2.5 text-base text-ink transition-colors placeholder:text-muted focus-visible:border-ink disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

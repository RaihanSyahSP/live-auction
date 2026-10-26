import type { HTMLAttributes } from "react";
import { cn } from "~/lib/utils";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("border border-ink bg-surface", className)} {...props} />
  );
}

export function CardHeader({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex flex-col p-[clamp(22px,4vw,40px)] pb-0", className)}
      {...props}
    />
  );
}

export function CardContent({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("p-[clamp(22px,4vw,40px)] pt-6", className)}
      {...props}
    />
  );
}

import type { ButtonHTMLAttributes } from "react";
import { cn } from "~/lib/utils";

const buttonVariants = {
  primary:
    "border-accent bg-accent font-semibold text-white hover:bg-accent-dark",
  secondary: "border-ink bg-transparent text-ink hover:bg-ink hover:text-paper",
  danger:
    "border-accent bg-transparent text-accent hover:bg-accent hover:text-white",
} as const;

const buttonSizes = {
  default: "min-h-[46px] px-4 py-2.5",
  sm: "min-h-11 px-3 py-2 text-sm",
} as const;

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: keyof typeof buttonVariants;
  size?: keyof typeof buttonSizes;
};

export function Button({
  className,
  variant = "primary",
  size = "default",
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-xs border font-medium transition-colors disabled:cursor-not-allowed disabled:border-rule-soft disabled:bg-rule-soft disabled:text-muted",
        buttonVariants[variant],
        buttonSizes[size],
        className,
      )}
      type={type}
      {...props}
    />
  );
}

export type { ButtonProps };

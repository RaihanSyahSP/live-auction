import type { HTMLAttributes, LabelHTMLAttributes } from "react";
import { cn } from "~/lib/utils";

export function Field({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("flex w-full flex-col gap-1.5", className)} {...props} />
  );
}

export function FieldGroup({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("flex w-full flex-col gap-4", className)} {...props} />
  );
}

type FieldLabelProps = Omit<
  LabelHTMLAttributes<HTMLLabelElement>,
  "htmlFor"
> & {
  htmlFor: string;
};

export function FieldLabel({
  className,
  htmlFor,
  children,
  ...props
}: FieldLabelProps) {
  return (
    <label
      className={cn("block text-[0.78rem] font-semibold text-ink", className)}
      htmlFor={htmlFor}
      {...props}
    >
      {children}
    </label>
  );
}

export function FieldDescription({
  className,
  ...props
}: HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p
      className={cn("text-sm leading-normal text-muted", className)}
      {...props}
    />
  );
}

export function FieldError({
  className,
  children,
  ...props
}: HTMLAttributes<HTMLParagraphElement>) {
  if (!children) {
    return null;
  }

  return (
    <p
      className={cn("text-[0.82rem] font-semibold text-accent", className)}
      role="alert"
      {...props}
    >
      {children}
    </p>
  );
}

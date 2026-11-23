import { HTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export type AlertVariant = "error" | "success" | "warn" | "info";

type AlertProps = HTMLAttributes<HTMLDivElement> & {
  variant?: AlertVariant;
  title?: string;
};

const variantClass: Record<AlertVariant, string> = {
  error: "alert-error",
  success: "alert-success",
  warn: "alert-warn",
  info: "alert-info",
};

const icons: Record<AlertVariant, JSX.Element> = {
  error: (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
      <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M8 5v4M8 11h.01" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  ),
  success: (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
      <path d="M4.5 8.5L7 11l4.5-6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  warn: (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
      <path d="M8 3.5L13.5 12H2.5L8 3.5Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
      <path d="M8 7v2.5M8 11h.01" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  ),
  info: (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden>
      <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.2" />
      <path d="M8 7v4M8 5.5h.01" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  ),
};

export function Alert({ className, variant = "info", title, children, ...props }: AlertProps) {
  return (
    <div className={cn(variantClass[variant], "flex gap-3", className)} role="alert" {...props}>
      <span className="mt-0.5 shrink-0">{icons[variant]}</span>
      <div className="min-w-0 flex-1">
        {title && <p className="mb-0.5 font-medium">{title}</p>}
        {children && <div className={title ? "text-[13px] opacity-90" : ""}>{children}</div>}
      </div>
    </div>
  );
}

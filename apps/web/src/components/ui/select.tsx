import { SelectHTMLAttributes, forwardRef } from "react";
import { cn } from "@/lib/cn";

type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  label?: string;
  hint?: string;
  error?: string;
};

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, label, hint, error, id, children, ...props }, ref) => {
    const selectId = id ?? (label ? label.toLowerCase().replace(/\s+/g, "-") : undefined);

    return (
      <div className="w-full">
        {label && (
          <label htmlFor={selectId} className="label">
            {label}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={selectId}
            className={cn("select mt-0", error && "border-risk/50", className)}
            aria-invalid={error ? true : undefined}
            {...props}
          >
            {children}
          </select>
          <span
            className="pointer-events-none absolute right-3.5 top-1/2 -translate-y-1/2 text-muted"
            aria-hidden
          >
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path d="M3 4.5L6 7.5L9 4.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </span>
        </div>
        {hint && !error && <p className="mt-1.5 text-supporting text-muted">{hint}</p>}
        {error && (
          <p className="mt-1.5 text-supporting text-risk" role="alert">
            {error}
          </p>
        )}
      </div>
    );
  },
);

Select.displayName = "Select";

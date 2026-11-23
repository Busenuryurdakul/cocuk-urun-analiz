import { ButtonHTMLAttributes, forwardRef } from "react";
import { cn } from "@/lib/cn";

type FilterPillProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  active?: boolean;
  count?: number;
};

export const FilterPill = forwardRef<HTMLButtonElement, FilterPillProps>(
  ({ className, active = false, count, children, type = "button", ...props }, ref) => (
    <button
      ref={ref}
      type={type}
      className={cn(active ? "filter-pill-active" : "filter-pill", className)}
      aria-pressed={active}
      {...props}
    >
      {children}
      {count !== undefined && (
        <span
          className={cn(
            "ml-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-semibold",
            active ? "bg-primary/20 text-primary" : "bg-secondary text-muted",
          )}
        >
          {count}
        </span>
      )}
    </button>
  ),
);

FilterPill.displayName = "FilterPill";

import { cn } from "@/lib/cn";
import { AlertTriangleIcon, ClockIcon, ShieldCheckIcon } from "@/components/ui/icons";

export type StatusRibbonVariant =
  | "safe"
  | "caution"
  | "risk"
  | "critical"
  | "pending"
  | "review"
  | "recall";

type StatusRibbonProps = {
  variant: StatusRibbonVariant;
  label?: string;
  className?: string;
  size?: "sm" | "md";
};

const config: Record<
  StatusRibbonVariant,
  { bg: string; text: string; border: string; icon: JSX.Element; defaultLabel: string }
> = {
  safe: {
    bg: "bg-safe/10",
    text: "text-safe",
    border: "border-safe/25",
    icon: <ShieldCheckIcon size={12} />,
    defaultLabel: "Safe",
  },
  caution: {
    bg: "bg-caution/10",
    text: "text-caution",
    border: "border-caution/25",
    icon: <AlertTriangleIcon size={12} />,
    defaultLabel: "Caution",
  },
  risk: {
    bg: "bg-risk/10",
    text: "text-risk",
    border: "border-risk/25",
    icon: <AlertTriangleIcon size={12} />,
    defaultLabel: "High risk",
  },
  critical: {
    bg: "bg-risk/15",
    text: "text-risk",
    border: "border-risk/35",
    icon: <AlertTriangleIcon size={12} />,
    defaultLabel: "Critical",
  },
  pending: {
    bg: "bg-secondary",
    text: "text-muted",
    border: "border-border",
    icon: <ClockIcon size={12} />,
    defaultLabel: "Pending",
  },
  review: {
    bg: "bg-info/10",
    text: "text-info",
    border: "border-info/25",
    icon: <ClockIcon size={12} />,
    defaultLabel: "Review",
  },
  recall: {
    bg: "bg-risk/15",
    text: "text-risk",
    border: "border-risk/40",
    icon: <AlertTriangleIcon size={12} />,
    defaultLabel: "Recall",
  },
};

export function StatusRibbon({ variant, label, className, size = "md" }: StatusRibbonProps) {
  const c = config[variant];
  const displayLabel = label ?? c.defaultLabel;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border font-medium",
        c.bg,
        c.text,
        c.border,
        size === "sm" ? "px-2 py-0.5 text-[10px]" : "px-2.5 py-1 text-[11px]",
        className,
      )}
      role="status"
      aria-label={displayLabel}
    >
      {c.icon}
      <span>{displayLabel}</span>
    </span>
  );
}

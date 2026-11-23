import { cn } from "@/lib/cn";
import { StatusRibbon, StatusRibbonVariant } from "@/components/miyuna/status-ribbon";
import { Button } from "@/components/ui/button";
import { AlertTriangleIcon, ClockIcon } from "@/components/ui/icons";

type PriorityAlertCardProps = {
  severity: StatusRibbonVariant;
  title: string;
  affectedProduct?: string;
  source?: string;
  timestamp?: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
};

const severityAccent: Record<string, string> = {
  critical: "border-l-risk bg-risk/[0.04]",
  risk: "border-l-risk bg-risk/[0.03]",
  caution: "border-l-caution bg-caution/[0.04]",
  recall: "border-l-risk bg-risk/[0.05]",
  safe: "border-l-safe bg-safe/[0.03]",
  pending: "border-l-border bg-secondary/30",
  review: "border-l-info bg-info/[0.03]",
};

export function PriorityAlertCard({
  severity,
  title,
  affectedProduct,
  source,
  timestamp,
  actionLabel,
  onAction,
  className,
}: PriorityAlertCardProps) {
  const isHighPriority = severity === "critical" || severity === "risk" || severity === "recall";

  return (
    <div
      className={cn(
        "rounded-card border border-border/80 border-l-4 p-4 shadow-card transition-all duration-normal hover:-translate-y-0.5 hover:shadow-card-hover sm:p-5",
        severityAccent[severity] ?? severityAccent.pending,
        className,
      )}
      role="article"
      aria-label={`Alert: ${title}`}
    >
      <div className="flex items-start gap-3">
        <span
          className={cn(
            "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl",
            isHighPriority ? "bg-risk/10 text-risk" : "bg-caution/10 text-caution",
          )}
          aria-hidden
        >
          <AlertTriangleIcon size={18} />
        </span>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <StatusRibbon variant={severity} size="sm" />
            {timestamp && (
              <span className="ml-auto flex items-center gap-1 text-[11px] text-muted">
                <ClockIcon size={11} />
                {timestamp}
              </span>
            )}
          </div>
          <h4 className="card-title">{title}</h4>
          <div className="space-y-0.5 text-sm text-text-secondary">
            {affectedProduct && (
              <p>
                <span className="text-muted">Product:</span> {affectedProduct}
              </p>
            )}
            {source && (
              <p>
                <span className="text-muted">Source:</span> {source}
              </p>
            )}
          </div>
          {actionLabel && (
            <Button variant={isHighPriority ? "primary" : "secondary"} size="sm" onClick={onAction}>
              {actionLabel}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

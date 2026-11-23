import { ReactNode } from "react";
import { cn } from "@/lib/cn";
import { AiSparkIcon } from "@/components/ui/icons";
import { Button } from "@/components/ui/button";

type MiyunaInsightCardProps = {
  title: string;
  summary: string;
  confidence?: number;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
  footer?: ReactNode;
};

export function MiyunaInsightCard({
  title,
  summary,
  confidence,
  actionLabel,
  onAction,
  className,
  footer,
}: MiyunaInsightCardProps) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-card-lg border border-primary/15 bg-gradient-to-br from-primary/[0.06] via-surface to-secondary p-5 shadow-card transition-all duration-normal hover:-translate-y-0.5 hover:shadow-card-hover",
        className,
      )}
    >
      <div className="pointer-events-none absolute -right-8 -top-8 h-32 w-32 rounded-full bg-primary/5 blur-2xl" aria-hidden />

      <div className="relative flex items-start gap-3">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-primary/12 text-primary">
          <AiSparkIcon size={16} />
        </span>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-[11px] font-medium uppercase tracking-[0.14em] text-primary">AI Insight</p>
            {confidence !== undefined && (
              <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
                {Math.round(confidence)}% confidence
              </span>
            )}
          </div>
          <h4 className="card-title">{title}</h4>
          <p className="text-sm leading-relaxed text-text-secondary">{summary}</p>
          {(actionLabel || footer) && (
            <div className="flex flex-wrap items-center gap-2 pt-1">
              {actionLabel && (
                <Button variant="secondary" size="sm" onClick={onAction}>
                  {actionLabel}
                </Button>
              )}
              {footer}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

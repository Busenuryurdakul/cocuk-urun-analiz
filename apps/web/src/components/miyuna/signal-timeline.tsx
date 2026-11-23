import { cn } from "@/lib/cn";
import { StatusRibbon, StatusRibbonVariant } from "@/components/miyuna/status-ribbon";

export type SignalEventType =
  | "analysis"
  | "source_fetch"
  | "recall"
  | "safety_finding"
  | "review"
  | "info";

export type SignalTimelineEvent = {
  id: string;
  type: SignalEventType;
  title: string;
  description?: string;
  timestamp: string;
  status?: StatusRibbonVariant;
  source?: string;
};

type SignalTimelineProps = {
  events: SignalTimelineEvent[];
  className?: string;
  maxItems?: number;
};

const typeConfig: Record<
  SignalEventType,
  { dot: string; ring: string; label: string }
> = {
  analysis: { dot: "bg-primary", ring: "ring-primary/20", label: "Analysis" },
  source_fetch: { dot: "bg-info", ring: "ring-info/20", label: "Source" },
  recall: { dot: "bg-risk", ring: "ring-risk/20", label: "Recall" },
  safety_finding: { dot: "bg-caution", ring: "ring-caution/20", label: "Finding" },
  review: { dot: "bg-purple-soft", ring: "ring-primary/20", label: "Review" },
  info: { dot: "bg-mint", ring: "ring-mint/20", label: "Info" },
};

export function SignalTimeline({ events, className, maxItems }: SignalTimelineProps) {
  const visible = maxItems ? events.slice(0, maxItems) : events;

  if (visible.length === 0) {
    return (
      <div className={cn("rounded-card border border-border/60 bg-secondary/50 px-4 py-8 text-center", className)}>
        <p className="text-sm text-muted">No signal events yet</p>
      </div>
    );
  }

  return (
    <ol className={cn("relative space-y-0", className)} aria-label="Signal timeline">
      {visible.map((event, index) => {
        const cfg = typeConfig[event.type];
        const isLast = index === visible.length - 1;

        return (
          <li key={event.id} className="relative flex gap-4 pb-6 last:pb-0">
            {!isLast && (
              <span
                className="absolute left-[11px] top-6 h-[calc(100%-12px)] w-px bg-border"
                aria-hidden
              />
            )}
            <span
              className={cn(
                "relative z-10 mt-1 flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-full ring-4",
                cfg.ring,
              )}
              aria-hidden
            >
              <span className={cn("h-2.5 w-2.5 rounded-full", cfg.dot)} />
            </span>
            <div className="min-w-0 flex-1 pt-0.5">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-[10px] font-medium uppercase tracking-wider text-muted">
                  {cfg.label}
                </span>
                {event.status && <StatusRibbon variant={event.status} size="sm" />}
                <time className="ml-auto text-[11px] text-muted">{event.timestamp}</time>
              </div>
              <p className="mt-1 text-sm font-medium text-ink">{event.title}</p>
              {event.description && (
                <p className="mt-0.5 text-sm text-text-secondary">{event.description}</p>
              )}
              {event.source && (
                <p className="mt-1 text-[11px] text-muted">Source: {event.source}</p>
              )}
            </div>
          </li>
        );
      })}
    </ol>
  );
}

import type { ReactNode } from "react";
import type { AnalysisRun } from "@/lib/analysis-types";
import {
  decisionLabel,
  riskLabel,
  statusLabel,
} from "@/lib/analysis-labels";
import {
  decisionToRibbon,
  formatShortDate,
  recallToRibbon,
  riskToRibbon,
} from "@/lib/product-intelligence";
import { StatusRibbon } from "@/components/miyuna/status-ribbon";
import { cn } from "@/lib/cn";

type IntelligenceStatusStripProps = {
  run?: AnalysisRun | null;
  locale?: "tr" | "en";
  className?: string;
};

export function IntelligenceStatusStrip({ run, locale = "tr", className }: IntelligenceStatusStripProps) {
  if (!run) {
    return (
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 rounded-card border border-border/70 bg-surface/80 px-4 py-3",
          className,
        )}
      >
        <StatusRibbon variant="pending" label="Analiz bekleniyor" size="sm" />
        <span className="text-xs text-muted">Henüz tamamlanmış güvenlik analizi yok</span>
      </div>
    );
  }

  const final = run.finalResult;
  const recallRibbon = recallToRibbon(run.recalls);
  const items: { key: string; node: ReactNode }[] = [];

  if (final) {
    items.push({
      key: "risk",
      node: <StatusRibbon variant={riskToRibbon(final.overallRisk)} label={riskLabel(final.overallRisk, locale)} size="sm" />,
    });
    items.push({
      key: "decision",
      node: (
        <StatusRibbon variant={decisionToRibbon(final.decision)} label={decisionLabel(final.decision, locale)} size="sm" />
      ),
    });
    items.push({
      key: "confidence",
      node: (
        <span className="rounded-full bg-secondary px-2.5 py-1 text-[11px] font-medium text-text-secondary">
          {Math.round(final.confidence * 100)}% güven
        </span>
      ),
    });
  } else if (run.status === "RUNNING" || run.status === "PENDING") {
    items.push({
      key: "status",
      node: <StatusRibbon variant="pending" label={statusLabel(run.status, locale)} size="sm" />,
    });
  }

  if (recallRibbon) {
    items.push({ key: "recall", node: <StatusRibbon variant={recallRibbon} size="sm" /> });
  }

  const analyzedAt = formatShortDate(run.completedAt ?? run.createdAt);
  if (analyzedAt) {
    items.push({
      key: "time",
      node: <span className="text-xs text-muted">Son analiz: {analyzedAt}</span>,
    });
  }

  if (items.length === 0) return null;

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 rounded-card border border-border/70 bg-gradient-to-r from-surface to-secondary/40 px-4 py-3",
        className,
      )}
      role="status"
      aria-label="Product intelligence snapshot"
    >
      {items.map((item) => (
        <span key={item.key}>{item.node}</span>
      ))}
    </div>
  );
}

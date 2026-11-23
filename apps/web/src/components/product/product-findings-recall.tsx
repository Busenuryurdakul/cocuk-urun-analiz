"use client";

import { useMemo, useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { FilterPill } from "@/components/ui/filter-pill";
import { MiyunaInsightCard } from "@/components/miyuna/insight-card";
import { SignalTimeline } from "@/components/miyuna/signal-timeline";
import { StatusRibbon } from "@/components/miyuna/status-ribbon";
import type { AnalysisRun } from "@/lib/analysis-types";
import {
  findingRationale,
  findingTypeLabel,
  isQualityFinding,
  localizePolicyText,
  riskLabel,
} from "@/lib/analysis-labels";
import type { Locale } from "@/lib/i18n/locale-config";
import {
  buildProductTimeline,
  severityToRibbon,
} from "@/lib/product-intelligence";
import type { Product } from "@/lib/product";
import { cn } from "@/lib/cn";

type ProductFindingsRecallProps = {
  product: Product;
  run?: AnalysisRun | null;
  locale?: Locale;
  lastSyncedAt?: string | null;
};

export function ProductIntelligenceSummary({
  run,
  locale = "tr",
}: {
  run?: AnalysisRun | null;
  locale?: Locale;
}) {
  const final = run?.finalResult;
  if (!final?.summary) return null;

  const highlights = [
    final.recommendation && !final.summary.includes(final.recommendation)
      ? localizePolicyText(final.recommendation, locale)
      : null,
    ...(final.limitations ?? []).slice(0, 2).map((l) => localizePolicyText(l, locale)),
  ].filter(Boolean) as string[];

  return (
    <MiyunaInsightCard
      title="Miyuna özeti"
      summary={localizePolicyText(final.summary, locale)}
      confidence={Math.round(final.confidence * 100)}
      footer={
        highlights.length > 0 ? (
          <ul className="space-y-1 text-left text-xs text-text-secondary">
            {highlights.map((h) => (
              <li key={h.slice(0, 40)} className="flex gap-2">
                <span className="text-primary">•</span>
                <span>{h}</span>
              </li>
            ))}
          </ul>
        ) : undefined
      }
    />
  );
}

export function ProductKeyFindings({
  run,
  locale = "tr",
}: {
  run?: AnalysisRun | null;
  locale?: Locale;
}) {
  const [filter, setFilter] = useState<"all" | "critical" | "high">("all");
  const findings = useMemo(() => run?.safetyFindings ?? [], [run?.safetyFindings]);
  const isEvidenceOnly = findings.length > 0 && findings.every((f) => isQualityFinding(f.type));

  const filtered = useMemo(() => {
    if (filter === "all") return findings;
    return findings.filter((f) =>
      filter === "critical" ? f.severity === "CRITICAL" : f.severity === "HIGH",
    );
  }, [filter, findings]);

  if (findings.length === 0) return null;

  return (
    <Card padding="md">
      <CardHeader>
        <CardTitle>{isEvidenceOnly ? "Kaynak durumu" : "Temel bulgular"}</CardTitle>
      </CardHeader>
      <div className="mb-4 flex flex-wrap gap-2">
        <FilterPill active={filter === "all"} onClick={() => setFilter("all")}>
          Tümü
        </FilterPill>
        <FilterPill active={filter === "critical"} onClick={() => setFilter("critical")}>
          Kritik
        </FilterPill>
        <FilterPill active={filter === "high"} onClick={() => setFilter("high")}>
          Yüksek
        </FilterPill>
      </div>
      <ul className="space-y-2">
        {filtered.map((finding) => {
          const rationale = findingRationale(finding.type, finding.rationale, locale);
          const isHigh = finding.severity === "CRITICAL" || finding.severity === "HIGH";
          return (
            <li
              key={finding.id}
              className={cn(
                "rounded-xl border px-4 py-3 transition-all duration-normal",
                isHigh ? "border-risk/25 bg-risk/[0.04]" : "border-border/70 bg-secondary/30",
              )}
            >
              <div className="flex flex-wrap items-center gap-2">
                <StatusRibbon
                  variant={severityToRibbon(finding.severity)}
                  label={riskLabel(finding.severity, locale)}
                  size="sm"
                />
                <span className="text-xs text-muted">{findingTypeLabel(finding.type, locale)}</span>
              </div>
              {rationale && <p className="mt-2 text-sm text-text-secondary">{rationale}</p>}
            </li>
          );
        })}
      </ul>
    </Card>
  );
}

export function ProductRecallBlock({ run }: { run?: AnalysisRun | null }) {
  const matched = run?.recalls?.filter((r) => r.matched) ?? [];

  return (
    <Card padding="md">
      <CardTitle className="text-base">Recall eşleşmeleri</CardTitle>
      {matched.length === 0 ? (
        <p className="mt-3 text-sm text-text-secondary">Aktif geri çağırma eşleşmesi bulunmadı.</p>
      ) : (
        <ul className="mt-4 space-y-2">
          {matched.map((match) => (
            <li key={`${match.source}-${match.sourceRecordId}`} className="rounded-xl border border-border/70 bg-surface px-4 py-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium text-ink">{match.source}</span>
                {match.requiresReview ? (
                  <StatusRibbon variant="review" size="sm" label="İnceleme gerekli" />
                ) : (
                  <StatusRibbon variant="recall" size="sm" />
                )}
              </div>
              {match.reference && <p className="mt-1 text-sm text-muted">{match.reference}</p>}
              <p className="mt-1 text-xs text-muted">
                Yöntem: {match.method} · {Math.round(match.confidence * 100)}% eşleşme güveni
              </p>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

export function ProductSignalBlock({
  product,
  run,
  lastSyncedAt,
}: ProductFindingsRecallProps) {
  const events = buildProductTimeline(product, run ?? undefined, lastSyncedAt, 8);
  if (events.length === 0) return null;

  return (
    <Card padding="md">
      <CardTitle className="text-base">Sinyal zaman çizelgesi</CardTitle>
      <div className="mt-4">
        <SignalTimeline events={events} />
      </div>
    </Card>
  );
}

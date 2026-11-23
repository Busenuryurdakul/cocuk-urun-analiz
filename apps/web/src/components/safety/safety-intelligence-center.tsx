"use client";

import Link from "next/link";
import { MetricCard } from "@/components/miyuna/metric-card";
import { PriorityAlertCard } from "@/components/miyuna/priority-alert-card";
import { SignalTimeline } from "@/components/miyuna/signal-timeline";
import { Card } from "@/components/ui/card";
import type { AnalysisRun } from "@/lib/analysis-types";
import {
  buildPriorityAlerts,
  buildSignalTimeline,
  computeSafetyOverview,
} from "@/lib/product-intelligence";
import type { Product } from "@/lib/product";
import { productTitle } from "@/lib/product";

type SafetyIntelligenceCenterProps = {
  orgId: string;
  products: Product[];
  analysisRuns: AnalysisRun[];
};

function SeverityBars({ runs }: { runs: AnalysisRun[] }) {
  const counts = { critical: 0, high: 0, medium: 0, low: 0 };
  for (const run of runs) {
    for (const f of run.safetyFindings ?? []) {
      if (f.severity === "CRITICAL") counts.critical += 1;
      else if (f.severity === "HIGH") counts.high += 1;
      else if (f.severity === "MEDIUM") counts.medium += 1;
      else counts.low += 1;
    }
  }
  const total = counts.critical + counts.high + counts.medium + counts.low;
  if (total === 0) {
    return <p className="text-sm text-muted">Güvenlik bulgusu kaydı yok.</p>;
  }

  const rows = [
    { label: "Kritik", count: counts.critical, tone: "bg-risk" },
    { label: "Yüksek", count: counts.high, tone: "bg-risk/70" },
    { label: "Orta", count: counts.medium, tone: "bg-caution" },
    { label: "Düşük", count: counts.low, tone: "bg-primary/40" },
  ];

  return (
    <div className="space-y-3">
      {rows.map((row) => (
        <div key={row.label} className="space-y-1">
          <div className="flex justify-between text-sm">
            <span className="text-text-secondary">{row.label}</span>
            <span className="font-medium">{row.count}</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-secondary">
            <div
              className={`h-full rounded-full ${row.tone}`}
              style={{ width: `${Math.round((row.count / total) * 100)}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

export function SafetyIntelligenceCenter({ orgId, products, analysisRuns }: SafetyIntelligenceCenterProps) {
  const stats = computeSafetyOverview(products, analysisRuns);
  const alerts = buildPriorityAlerts(products, analysisRuns, orgId, 6);
  const timeline = buildSignalTimeline(products, analysisRuns, [], 8);

  const sourceGroups = new Map<string, number>();
  for (const run of analysisRuns) {
    for (const r of run.recalls ?? []) {
      if (r.matched) sourceGroups.set(r.source, (sourceGroups.get(r.source) ?? 0) + 1);
    }
    for (const f of run.safetyFindings ?? []) {
      sourceGroups.set(f.type, (sourceGroups.get(f.type) ?? 0) + 1);
    }
  }

  const impacted = analysisRuns
    .filter(
      (r) =>
        r.recalls?.some((x) => x.matched) ||
        r.safetyFindings?.some((f) => f.severity === "CRITICAL" || f.severity === "HIGH") ||
        r.finalResult?.decision === "REVIEW_REQUIRED" ||
        r.finalResult?.decision === "BLOCK",
    )
    .slice(0, 8);

  const hasAnyData =
    stats.recallMatches > 0 ||
    stats.criticalFindings > 0 ||
    stats.reviewRequired > 0 ||
    alerts.length > 0;

  if (!hasAnyData && analysisRuns.length === 0) {
    return null;
  }

  return (
    <section className="space-y-6">
      <div>
        <h2 className="section-title text-xl">Safety Intelligence</h2>
        <p className="mt-1 text-sm text-muted">Recall, bulgu ve inceleme sinyallerinin org geneli görünümü</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.recallMatches > 0 && (
          <MetricCard label="Aktif recall eşleşmesi" value={stats.recallMatches} context="Analiz kayıtları" />
        )}
        {stats.criticalFindings > 0 && (
          <MetricCard label="Kritik/yüksek bulgu" value={stats.criticalFindings} context="Safety findings" />
        )}
        {stats.reviewRequired > 0 && (
          <MetricCard label="İnceleme gereken" value={stats.reviewRequired} context="Karar veya recall" />
        )}
        {stats.recentSignals > 0 && (
          <MetricCard label="Son güvenlik sinyali" value={stats.recentSignals} context="Tamamlanan analiz + bulgu" />
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-12">
        <div className="space-y-4 lg:col-span-7">
          {alerts.length > 0 ? (
            alerts.map((alert) => (
              <PriorityAlertCard
                key={alert.id}
                severity={alert.severity}
                title={alert.title}
                affectedProduct={alert.affectedProduct}
                source={alert.source}
                timestamp={alert.timestamp}
                actionLabel="Ürünü aç"
                onAction={() => {
                  window.location.href = alert.href;
                }}
              />
            ))
          ) : (
            <Card variant="tinted" className="text-sm text-muted">
              Öncelikli güvenlik uyarısı yok.
            </Card>
          )}
        </div>

        <div className="space-y-4 lg:col-span-5">
          <Card padding="md">
            <h3 className="card-title text-base">Bulgu dağılımı</h3>
            <div className="mt-4">
              <SeverityBars runs={analysisRuns} />
            </div>
          </Card>

          {sourceGroups.size > 0 && (
            <Card padding="md">
              <h3 className="card-title text-base">Kaynak sinyalleri</h3>
              <ul className="mt-3 space-y-1.5">
                {Array.from(sourceGroups.entries())
                  .sort((a, b) => b[1] - a[1])
                  .slice(0, 6)
                  .map(([source, count]) => (
                    <li key={source} className="flex justify-between text-sm">
                      <span className="text-text-secondary">{source}</span>
                      <span className="font-medium text-ink">{count}</span>
                    </li>
                  ))}
              </ul>
            </Card>
          )}
        </div>
      </div>

      {impacted.length > 0 && (
        <Card padding="md">
          <h3 className="card-title text-base">Etkilenen ürünler</h3>
          <ul className="mt-3 divide-y divide-border/60">
            {impacted.map((run) => {
              const product = products.find((p) => p.id === run.productId);
              return (
                <li key={run.id} className="flex flex-wrap items-center justify-between gap-2 py-3">
                  <Link
                    href={`/org/${orgId}/products/${run.productId}`}
                    className="text-sm font-medium text-primary hover:text-primary-hover"
                  >
                    {product ? productTitle(product) : run.productId}
                  </Link>
                  <span className="text-xs text-muted">
                    {run.finalResult?.decision ?? run.status}
                  </span>
                </li>
              );
            })}
          </ul>
        </Card>
      )}

      {timeline.length > 0 && (
        <Card padding="md">
          <h3 className="card-title text-base">Güvenlik zaman çizelgesi</h3>
          <div className="mt-4">
            <SignalTimeline events={timeline} maxItems={6} />
          </div>
        </Card>
      )}
    </section>
  );
}

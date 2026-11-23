"use client";

import Link from "next/link";
import { MiyunaInsightCard } from "@/components/miyuna/insight-card";
import { IntelligenceScoreRing } from "@/components/miyuna/intelligence-score-ring";
import { MetricCard } from "@/components/miyuna/metric-card";
import { PriorityAlertCard } from "@/components/miyuna/priority-alert-card";
import { SignalTimeline } from "@/components/miyuna/signal-timeline";
import { StatusRibbon } from "@/components/miyuna/status-ribbon";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { PackageIcon } from "@/components/ui/icons";
import {
  buildPriorityAlerts,
  buildSignalTimeline,
  computeDashboardStats,
  enrichProduct,
  formatShortDate,
  latestRunByProduct,
  parseProductImageUrl,
  pickInsightSummary,
  type DashboardStats,
} from "@/lib/product-intelligence";
import type { AnalysisRun, MarketplaceImportRun } from "@/lib/analysis-types";
import { productTitle, type Product } from "@/lib/product";
import { SafetyIntelligenceCenter } from "@/components/safety/safety-intelligence-center";
import { cn } from "@/lib/cn";

type WorkspaceDashboardProps = {
  orgId: string;
  orgName?: string;
  products: Product[];
  analysisRuns: AnalysisRun[];
  importRuns: MarketplaceImportRun[];
};

function RiskBar({
  label,
  count,
  total,
  tone,
}: {
  label: string;
  count: number;
  total: number;
  tone: string;
}) {
  const pct = total > 0 ? Math.round((count / total) * 100) : 0;
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="text-text-secondary">{label}</span>
        <span className="font-medium text-ink">{count}</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full transition-all duration-slow motion-reduce:transition-none", tone)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function RecentAnalysisRow({
  product,
  run,
  orgId,
}: {
  product: Product;
  run: AnalysisRun;
  orgId: string;
}) {
  const enriched = enrichProduct(product, run);
  const imageUrl = parseProductImageUrl(product);

  return (
    <Link
      href={`/org/${orgId}/products/${product.id}`}
      className="group flex items-center gap-4 rounded-xl border border-border/60 bg-surface px-4 py-3 transition-all duration-normal hover:-translate-y-0.5 hover:border-primary/20 hover:shadow-card-hover motion-reduce:transform-none"
    >
      <div className="relative h-12 w-12 shrink-0 overflow-hidden rounded-lg bg-secondary">
        {imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={imageUrl} alt="" className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-muted">
            <PackageIcon size={18} />
          </div>
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-ink">{productTitle(product)}</p>
        <p className="mt-0.5 truncate text-xs text-muted">
          {enriched.decisionLabel ?? enriched.statusLabel ?? "Analiz kaydı"}
          {enriched.confidence !== undefined ? ` · ${Math.round(enriched.confidence)}% güven` : ""}
        </p>
      </div>
      <div className="hidden shrink-0 flex-col items-end gap-1 sm:flex">
        <StatusRibbon variant={enriched.riskRibbon} size="sm" label={enriched.riskLabel} />
        {enriched.recallRibbon && <StatusRibbon variant={enriched.recallRibbon} size="sm" />}
      </div>
      <time className="shrink-0 text-[11px] text-muted">
        {formatShortDate(run.completedAt ?? run.createdAt)}
      </time>
    </Link>
  );
}

function HeroSummary({ stats, orgName }: { stats: DashboardStats; orgName?: string }) {
  const parts: string[] = [];
  parts.push(`${stats.totalProducts} ürün katalogda`);
  if (stats.analyzedProducts > 0) {
    parts.push(`${stats.analyzedProducts} analiz tamamlandı`);
  }
  if (stats.runningAnalyses > 0) {
    parts.push(`${stats.runningAnalyses} analiz devam ediyor`);
  }
  if (stats.highRiskCount > 0) {
    parts.push(`${stats.highRiskCount} yüksek risk`);
  }

  return (
    <div className="relative overflow-hidden rounded-card-lg border border-border/70 bg-gradient-to-br from-surface via-base to-secondary p-6 sm:p-8 lg:col-span-8">
      <div className="pointer-events-none absolute -right-16 -top-16 h-48 w-48 rounded-full bg-primary/5 blur-3xl" aria-hidden />
      <p className="kicker">Ürün güvenliği görünümü</p>
      <h2 className="mt-3 font-heading text-section-title text-ink">
        {orgName ? `${orgName} intelligence özeti` : "Intelligence özeti"}
      </h2>
      <p className="mt-3 max-w-xl text-sm leading-relaxed text-text-secondary">{parts.join(" · ")}</p>
      <div className="mt-6 flex flex-wrap gap-2">
        {stats.highRiskCount > 0 && (
          <StatusRibbon variant="risk" size="sm" label={`${stats.highRiskCount} yüksek risk`} />
        )}
        {stats.recallMatchCount > 0 && (
          <StatusRibbon variant="recall" size="sm" label={`${stats.recallMatchCount} recall`} />
        )}
        {stats.reviewRequiredCount > 0 && (
          <StatusRibbon variant="review" size="sm" label={`${stats.reviewRequiredCount} inceleme`} />
        )}
        {stats.runningAnalyses > 0 && (
          <StatusRibbon variant="pending" size="sm" label={`${stats.runningAnalyses} devam eden`} />
        )}
      </div>
    </div>
  );
}

export function WorkspaceDashboard({
  orgId,
  orgName,
  products,
  analysisRuns,
  importRuns,
}: WorkspaceDashboardProps) {
  const stats = computeDashboardStats(products, analysisRuns);
  const runMap = latestRunByProduct(analysisRuns);
  const alerts = buildPriorityAlerts(products, analysisRuns, orgId, 4);
  const timeline = buildSignalTimeline(products, analysisRuns, importRuns, 6);
  const insight = pickInsightSummary(analysisRuns);

  const recentRuns = [...analysisRuns]
    .sort((a, b) => (b.completedAt ?? b.createdAt).localeCompare(a.completedAt ?? a.createdAt))
    .slice(0, 6);

  const ringBreakdown =
    stats.analyzedProducts > 0
      ? [
          { label: "Düşük", value: stats.riskDistribution.safe / stats.analyzedProducts, color: "#46B486" },
          { label: "Orta", value: stats.riskDistribution.caution / stats.analyzedProducts, color: "#E7AA3C" },
          { label: "Yüksek", value: stats.riskDistribution.risk / stats.analyzedProducts, color: "#DC686D" },
          { label: "İnceleme", value: stats.riskDistribution.review / stats.analyzedProducts, color: "#7567E8" },
        ]
      : undefined;

  const ringSeverity =
    stats.riskDistribution.risk > 0 ? "risk" : stats.riskDistribution.caution > 0 ? "caution" : "safe";

  return (
    <div className="space-y-8">
      {/* Intelligence hero — bento row */}
      <section className="grid-12">
        <HeroSummary stats={stats} orgName={orgName} />
        <div className="flex items-center justify-center rounded-card-lg border border-border/70 bg-surface p-6 shadow-card lg:col-span-4">
          <IntelligenceScoreRing
            score={stats.coveragePercent}
            maxScore={100}
            label="Analiz kapsamı"
            severity={ringSeverity}
            breakdown={ringBreakdown}
            size="md"
          />
        </div>
      </section>

      {/* KPI row */}
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Toplam ürün" value={stats.totalProducts} context="Katalogdaki canonical ürünler" />
        {stats.analyzedProducts > 0 && (
          <MetricCard
            label="Analiz tamamlanan"
            value={stats.analyzedProducts}
            context={`${stats.coveragePercent}% kapsam`}
          />
        )}
        {stats.highRiskCount > 0 && (
          <MetricCard label="Yüksek riskli" value={stats.highRiskCount} context="Tamamlanan analizler" />
        )}
        {stats.recallMatchCount > 0 && (
          <MetricCard label="Recall eşleşmesi" value={stats.recallMatchCount} context="Analiz kayıtları" />
        )}
        {stats.reviewRequiredCount > 0 && (
          <MetricCard label="İnceleme bekleyen" value={stats.reviewRequiredCount} context="Karar veya recall" />
        )}
        {stats.missingFieldProducts > 0 && (
          <MetricCard
            label="Eksik alanlı ürün"
            value={stats.missingFieldProducts}
            context="Kaynak verisi eksik"
          />
        )}
      </section>

      {/* Main bento grid */}
      <section className="grid gap-6 lg:grid-cols-12">
        <div className="space-y-4 lg:col-span-8">
          <div className="flex items-end justify-between gap-3">
            <div>
              <h3 className="section-title text-xl">Son analizler</h3>
              <p className="mt-1 text-sm text-muted">En güncel agent analiz kayıtları</p>
            </div>
            <Link href={`/org/${orgId}/products`} className="text-sm font-medium text-primary hover:text-primary-hover">
              Tüm ürünler →
            </Link>
          </div>
          {recentRuns.length === 0 ? (
            <Card variant="tinted" className="text-center">
              <p className="text-sm text-muted">Henüz analiz kaydı yok.</p>
              <Link href={`/org/${orgId}/products`} className="btn-secondary mt-4 inline-flex">
                Ürün kataloğuna git
              </Link>
            </Card>
          ) : (
            <div className="space-y-2">
              {recentRuns.map((run) => {
                const product = products.find((p) => p.id === run.productId);
                if (!product) return null;
                return <RecentAnalysisRow key={run.id} product={product} run={run} orgId={orgId} />;
              })}
            </div>
          )}
        </div>

        <div className="space-y-6 lg:col-span-4">
          <Card padding="md">
            <h3 className="card-title">Risk dağılımı</h3>
            <p className="mt-1 text-xs text-muted">Tamamlanan analizlerde</p>
            {stats.analyzedProducts === 0 ? (
              <p className="mt-6 text-sm text-muted">Dağılım için tamamlanmış analiz gerekli.</p>
            ) : (
              <div className="mt-5 space-y-4">
                <RiskBar label="Güvenli" count={stats.riskDistribution.safe} total={stats.analyzedProducts} tone="bg-safe" />
                <RiskBar label="Dikkat" count={stats.riskDistribution.caution} total={stats.analyzedProducts} tone="bg-caution" />
                <RiskBar label="Yüksek risk" count={stats.riskDistribution.risk} total={stats.analyzedProducts} tone="bg-risk" />
                <RiskBar label="İnceleme" count={stats.riskDistribution.review} total={stats.analyzedProducts} tone="bg-primary" />
              </div>
            )}
          </Card>

          {alerts.length > 0 && (
            <div className="space-y-3">
              <h3 className="card-title">Öncelikli uyarılar</h3>
              {alerts.map((alert) => (
                <PriorityAlertCard
                  key={alert.id}
                  severity={alert.severity}
                  title={alert.title}
                  affectedProduct={alert.affectedProduct}
                  source={alert.source}
                  timestamp={alert.timestamp}
                  actionLabel="Detay"
                  onAction={() => {
                    window.location.href = alert.href;
                  }}
                />
              ))}
            </div>
          )}
        </div>
      </section>

      {/* Insight + timeline */}
      <section className="grid gap-6 lg:grid-cols-2">
        {insight ? (
          <MiyunaInsightCard
            title={insight.title}
            summary={insight.summary}
            confidence={insight.confidence !== undefined ? Math.round(insight.confidence * 100) : undefined}
            actionLabel="İlgili ürünü aç"
            onAction={() => {
              window.location.href = `/org/${orgId}/products/${insight.productId}`;
            }}
          />
        ) : (
          <Card variant="tinted">
            <h3 className="card-title">Miyuna intelligence</h3>
            <p className="mt-2 text-sm text-text-secondary">
              Tamamlanmış analiz sonucu olduğunda burada özet görünür. Şu an gösterilecek AI özeti yok.
            </p>
          </Card>
        )}

        <Card padding="md">
          <h3 className="card-title">Sinyal zaman çizelgesi</h3>
          <p className="mt-1 text-xs text-muted">Import, analiz ve güvenlik olayları</p>
          <div className="mt-5">
            <SignalTimeline events={timeline} />
          </div>
        </Card>
      </section>

      <SafetyIntelligenceCenter orgId={orgId} products={products} analysisRuns={analysisRuns} />

      {/* Quick catalog peek */}
      {products.length > 0 && (
        <section>
          <div className="mb-4 flex items-center justify-between">
            <h3 className="section-title text-xl">Katalog özeti</h3>
            <Link href={`/org/${orgId}/products`}>
              <Button variant="secondary" size="sm">
                Product Intelligence Library
              </Button>
            </Link>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {products.slice(0, 3).map((product) => {
              const enriched = enrichProduct(product, runMap.get(product.id));
              return (
                <Link
                  key={product.id}
                  href={`/org/${orgId}/products/${product.id}`}
                  className="rounded-card border border-border/70 bg-surface p-4 transition-all duration-normal hover:-translate-y-0.5 hover:shadow-card-hover motion-reduce:transform-none"
                >
                  <p className="truncate text-sm font-medium">{productTitle(product)}</p>
                  <div className="mt-2 flex flex-wrap gap-1.5">
                    <StatusRibbon variant={enriched.riskRibbon} size="sm" />
                    {enriched.recallRibbon && <StatusRibbon variant={enriched.recallRibbon} size="sm" />}
                  </div>
                </Link>
              );
            })}
          </div>
        </section>
      )}
    </div>
  );
}

export function WorkspaceDashboardSkeleton() {
  return (
    <div className="space-y-8" aria-busy="true">
      <div className="grid-12">
        <div className="h-48 animate-pulse rounded-card-lg bg-secondary lg:col-span-8" />
        <div className="h-48 animate-pulse rounded-card-lg bg-secondary lg:col-span-4" />
      </div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-card bg-secondary" />
        ))}
      </div>
      <div className="grid gap-6 lg:grid-cols-12">
        <div className="h-80 animate-pulse rounded-card bg-secondary lg:col-span-8" />
        <div className="h-80 animate-pulse rounded-card bg-secondary lg:col-span-4" />
      </div>
    </div>
  );
}

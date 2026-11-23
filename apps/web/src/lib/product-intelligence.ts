import type { StatusRibbonVariant } from "@/components/miyuna/status-ribbon";
import type { SignalTimelineEvent } from "@/components/miyuna/signal-timeline";
import { decisionLabel, riskLabel, statusLabel } from "@/lib/analysis-labels";
import type { AnalysisRun, MarketplaceImportRun, RecallMatch, SafetyFinding } from "@/lib/analysis-types";
import {
  fieldValue,
  missingFieldCount,
  productTitle,
  type Product,
} from "@/lib/product";

export function parseProductImageUrl(product: Product): string | undefined {
  const raw = fieldValue(product.imageRefs);
  if (!raw) return undefined;

  try {
    const parsed = JSON.parse(raw) as unknown;
    if (Array.isArray(parsed)) {
      const first = parsed.find((item) => typeof item === "string" && item.startsWith("http"));
      if (typeof first === "string") return first;
    }
  } catch {
    /* plain text / delimited */
  }

  const token = raw.split(/[\s,;|\n]+/).find((part) => part.startsWith("http"));
  return token || undefined;
}

export function riskToRibbon(risk: string): StatusRibbonVariant {
  switch (risk) {
    case "LOW":
      return "safe";
    case "MEDIUM":
      return "caution";
    case "HIGH":
      return "risk";
    case "CRITICAL":
      return "critical";
    default:
      return "review";
  }
}

export function decisionToRibbon(decision: string): StatusRibbonVariant {
  switch (decision) {
    case "ALLOW":
      return "safe";
    case "ALLOW_WITH_WARNING":
      return "caution";
    case "REVIEW_REQUIRED":
      return "review";
    case "BLOCK":
      return "risk";
    default:
      return "pending";
  }
}

export function recallToRibbon(recalls?: RecallMatch[]): StatusRibbonVariant | undefined {
  const matched = recalls?.filter((r) => r.matched) ?? [];
  if (matched.length === 0) return undefined;
  if (matched.some((r) => r.requiresReview)) return "review";
  return "recall";
}

export function latestRunByProduct(runs: AnalysisRun[]): Map<string, AnalysisRun> {
  const map = new Map<string, AnalysisRun>();
  for (const run of [...runs].sort((a, b) => b.createdAt.localeCompare(a.createdAt))) {
    if (!map.has(run.productId)) map.set(run.productId, run);
  }
  return map;
}

export type RiskBucket = "safe" | "caution" | "risk" | "review";

export type DashboardStats = {
  totalProducts: number;
  analyzedProducts: number;
  highRiskCount: number;
  recallMatchCount: number;
  reviewRequiredCount: number;
  missingFieldProducts: number;
  runningAnalyses: number;
  riskDistribution: Record<RiskBucket, number>;
  coveragePercent: number;
};

export function computeDashboardStats(products: Product[], runs: AnalysisRun[]): DashboardStats {
  const byProduct = latestRunByProduct(runs);
  const riskDistribution: Record<RiskBucket, number> = {
    safe: 0,
    caution: 0,
    risk: 0,
    review: 0,
  };

  let analyzedProducts = 0;
  let highRiskCount = 0;
  let recallMatchCount = 0;
  let reviewRequiredCount = 0;
  let runningAnalyses = 0;

  for (const product of products) {
    const run = byProduct.get(product.id);
    if (!run) continue;

    if (run.status === "RUNNING" || run.status === "PENDING") runningAnalyses += 1;

    if (run.status === "COMPLETED" && run.finalResult) {
      analyzedProducts += 1;
      const risk = run.finalResult.overallRisk;
      if (risk === "LOW") riskDistribution.safe += 1;
      else if (risk === "MEDIUM") riskDistribution.caution += 1;
      else if (risk === "HIGH" || risk === "CRITICAL") {
        riskDistribution.risk += 1;
        highRiskCount += 1;
      } else {
        riskDistribution.review += 1;
      }

      if (run.finalResult.decision === "REVIEW_REQUIRED") reviewRequiredCount += 1;
    } else if (run.status === "COMPLETED") {
      analyzedProducts += 1;
      riskDistribution.review += 1;
    }

    if (run.recalls?.some((r) => r.matched)) recallMatchCount += 1;
    if (run.recalls?.some((r) => r.matched && r.requiresReview)) reviewRequiredCount += 1;
  }

  const totalProducts = products.length;
  const missingFieldProducts = products.filter((p) => missingFieldCount(p) > 0).length;
  const coveragePercent =
    totalProducts > 0 ? Math.round((analyzedProducts / totalProducts) * 100) : 0;

  return {
    totalProducts,
    analyzedProducts,
    highRiskCount,
    recallMatchCount,
    reviewRequiredCount,
    missingFieldProducts,
    runningAnalyses,
    riskDistribution,
    coveragePercent,
  };
}

export type EnrichedProduct = {
  product: Product;
  run?: AnalysisRun;
  riskRibbon: StatusRibbonVariant;
  recallRibbon?: StatusRibbonVariant;
  confidence?: number;
  decisionLabel?: string;
  riskLabel?: string;
  statusLabel?: string;
  lastAnalysis?: string;
};

export function enrichProduct(product: Product, run?: AnalysisRun): EnrichedProduct {
  const final = run?.finalResult;
  return {
    product,
    run,
    riskRibbon: final ? riskToRibbon(final.overallRisk) : run?.status === "RUNNING" ? "pending" : "pending",
    recallRibbon: recallToRibbon(run?.recalls),
    confidence: final?.confidence,
    decisionLabel: final ? decisionLabel(final.decision) : run ? statusLabel(run.status) : undefined,
    riskLabel: final ? riskLabel(final.overallRisk) : undefined,
    statusLabel: run ? statusLabel(run.status) : undefined,
    lastAnalysis: run?.completedAt ?? run?.createdAt,
  };
}

export function formatShortDate(iso?: string | null): string | undefined {
  if (!iso) return undefined;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleDateString("tr-TR", { day: "numeric", month: "short", year: "numeric" });
}

export type PriorityAlert = {
  id: string;
  severity: StatusRibbonVariant;
  title: string;
  affectedProduct: string;
  source?: string;
  timestamp?: string;
  href: string;
};

function severityRank(variant: StatusRibbonVariant): number {
  switch (variant) {
    case "critical":
    case "recall":
      return 0;
    case "risk":
      return 1;
    case "caution":
      return 2;
    case "review":
      return 3;
    default:
      return 4;
  }
}

function findingSeverityVariant(finding: SafetyFinding): StatusRibbonVariant {
  if (finding.severity === "CRITICAL") return "critical";
  if (finding.severity === "HIGH") return "risk";
  if (finding.severity === "MEDIUM") return "caution";
  return "review";
}

export function buildPriorityAlerts(
  products: Product[],
  runs: AnalysisRun[],
  orgId: string,
  limit = 5,
): PriorityAlert[] {
  const productMap = new Map(products.map((p) => [p.id, p]));
  const alerts: PriorityAlert[] = [];

  for (const run of runs) {
    const product = productMap.get(run.productId);
    const productName = product ? productTitle(product) : run.productId;

    for (const finding of run.safetyFindings ?? []) {
      if (finding.severity !== "CRITICAL" && finding.severity !== "HIGH" && finding.severity !== "MEDIUM") {
        continue;
      }
      alerts.push({
        id: `finding-${finding.id}`,
        severity: findingSeverityVariant(finding),
        title: finding.rationale.slice(0, 120) || finding.type,
        affectedProduct: productName,
        source: finding.type,
        timestamp: formatShortDate(run.completedAt ?? run.createdAt),
        href: `/org/${orgId}/products/${run.productId}`,
      });
    }

    for (const recall of run.recalls ?? []) {
      if (!recall.matched) continue;
      alerts.push({
        id: `recall-${run.id}-${recall.sourceRecordId}`,
        severity: recall.requiresReview ? "review" : "recall",
        title: recall.requiresReview ? "Recall eşleşmesi — inceleme gerekli" : "Recall eşleşmesi",
        affectedProduct: productName,
        source: recall.source,
        timestamp: formatShortDate(run.completedAt ?? run.createdAt),
        href: `/org/${orgId}/products/${run.productId}`,
      });
    }

    if (run.finalResult?.decision === "BLOCK") {
      alerts.push({
        id: `block-${run.id}`,
        severity: "critical",
        title: run.finalResult.summary.slice(0, 120) || "Politika engeli",
        affectedProduct: productName,
        source: decisionLabel(run.finalResult.decision),
        timestamp: formatShortDate(run.completedAt ?? run.createdAt),
        href: `/org/${orgId}/products/${run.productId}`,
      });
    }
  }

  return alerts
    .sort((a, b) => severityRank(a.severity) - severityRank(b.severity))
    .slice(0, limit);
}

export function buildSignalTimeline(
  products: Product[],
  runs: AnalysisRun[],
  imports: MarketplaceImportRun[],
  limit = 8,
): SignalTimelineEvent[] {
  const productMap = new Map(products.map((p) => [p.id, p]));
  const events: SignalTimelineEvent[] = [];

  for (const imp of imports) {
    events.push({
      id: `import-${imp.id}`,
      type: "source_fetch",
      title: `Marketplace import — ${imp.source}`,
      description: `${imp.recordsAccepted} kayıt kabul edildi${imp.recordsRejected ? `, ${imp.recordsRejected} reddedildi` : ""}`,
      timestamp: formatShortDate(imp.createdAt) ?? imp.createdAt,
      status: imp.status === "FAILED" ? "risk" : imp.status === "SUCCEEDED" ? "safe" : "pending",
      source: imp.source,
    });
  }

  for (const run of runs) {
    const product = productMap.get(run.productId);
    const name = product ? productTitle(product) : "Ürün";

    if (run.status === "COMPLETED" && run.finalResult) {
      events.push({
        id: `analysis-${run.id}`,
        type: "analysis",
        title: `Analiz tamamlandı — ${name}`,
        description: run.finalResult.summary.slice(0, 140),
        timestamp: formatShortDate(run.completedAt ?? run.createdAt) ?? run.createdAt,
        status: decisionToRibbon(run.finalResult.decision),
      });
    } else if (run.status === "RUNNING") {
      events.push({
        id: `running-${run.id}`,
        type: "analysis",
        title: `Analiz devam ediyor — ${name}`,
        description: run.currentPhase ? `Aşama: ${run.currentPhase}` : undefined,
        timestamp: formatShortDate(run.createdAt) ?? run.createdAt,
        status: "pending",
      });
    } else if (run.status === "FAILED" || run.status === "REJECTED") {
      events.push({
        id: `failed-${run.id}`,
        type: "safety_finding",
        title: `Analiz tamamlanamadı — ${name}`,
        description: run.terminalReason ?? run.terminalError ?? statusLabel(run.status),
        timestamp: formatShortDate(run.createdAt) ?? run.createdAt,
        status: "risk",
      });
    }

    for (const recall of run.recalls ?? []) {
      if (!recall.matched) continue;
      events.push({
        id: `timeline-recall-${run.id}-${recall.sourceRecordId}`,
        type: "recall",
        title: `Recall sinyali — ${name}`,
        description: recall.reference ?? recall.source,
        timestamp: formatShortDate(run.completedAt ?? run.createdAt) ?? run.createdAt,
        status: recall.requiresReview ? "review" : "recall",
        source: recall.source,
      });
    }

    for (const finding of run.safetyFindings ?? []) {
      if (finding.severity !== "CRITICAL" && finding.severity !== "HIGH") continue;
      events.push({
        id: `timeline-finding-${finding.id}`,
        type: "safety_finding",
        title: `Güvenlik bulgusu — ${name}`,
        description: finding.rationale.slice(0, 140),
        timestamp: formatShortDate(run.completedAt ?? run.createdAt) ?? run.createdAt,
        status: findingSeverityVariant(finding),
        source: finding.type,
      });
    }
  }

  return events
    .sort((a, b) => (b.timestamp ?? "").localeCompare(a.timestamp ?? ""))
    .slice(0, limit);
}

export function pickInsightSummary(
  runs: AnalysisRun[],
): { title: string; summary: string; confidence?: number; productId: string } | null {
  const completed = runs
    .filter((r) => r.status === "COMPLETED" && r.finalResult?.summary)
    .sort((a, b) => (b.completedAt ?? b.createdAt).localeCompare(a.completedAt ?? a.createdAt));

  const blocked = completed.find((r) => r.finalResult?.decision === "BLOCK");
  if (blocked?.finalResult) {
    return {
      title: "Politika engeli tespit edildi",
      summary: blocked.finalResult.summary,
      confidence: blocked.finalResult.confidence,
      productId: blocked.productId,
    };
  }

  const review = completed.find((r) => r.finalResult?.decision === "REVIEW_REQUIRED");
  if (review?.finalResult) {
    return {
      title: "İnceleme gerektiren analiz",
      summary: review.finalResult.summary,
      confidence: review.finalResult.confidence,
      productId: review.productId,
    };
  }

  const highRisk = completed.find(
    (r) => r.finalResult && (r.finalResult.overallRisk === "HIGH" || r.finalResult.overallRisk === "CRITICAL"),
  );
  if (highRisk?.finalResult) {
    return {
      title: "Yüksek risk sinyali",
      summary: highRisk.finalResult.summary,
      confidence: highRisk.finalResult.confidence,
      productId: highRisk.productId,
    };
  }

  const latest = completed[0];
  if (latest?.finalResult) {
    return {
      title: "Son analiz özeti",
      summary: latest.finalResult.summary,
      confidence: latest.finalResult.confidence,
      productId: latest.productId,
    };
  }

  return null;
}

export function productSourceLabel(product: Product): string | undefined {
  return product.name.source ?? product.brand.source ?? product.category.source ?? undefined;
}

export function severityToRibbon(severity: string): StatusRibbonVariant {
  if (severity === "CRITICAL") return "critical";
  if (severity === "HIGH") return "risk";
  if (severity === "MEDIUM") return "caution";
  return "review";
}

export function buildProductTimeline(
  product: Product,
  run?: AnalysisRun,
  syncAt?: string | null,
  limit = 10,
): SignalTimelineEvent[] {
  const events: SignalTimelineEvent[] = [];
  const name = productTitle(product);

  if (product.createdAt) {
    events.push({
      id: `created-${product.id}`,
      type: "info",
      title: "Ürün kataloğa eklendi",
      description: name,
      timestamp: formatShortDate(product.createdAt) ?? product.createdAt,
      status: "safe",
    });
  }

  if (syncAt) {
    events.push({
      id: `sync-${product.id}-${syncAt}`,
      type: "source_fetch",
      title: "Canlı veri senkronu",
      description: productSourceLabel(product) ?? "Marketplace kaynağı",
      timestamp: formatShortDate(syncAt) ?? syncAt,
      status: "safe",
      source: productSourceLabel(product),
    });
  }

  if (run) {
    events.push(
      ...buildSignalTimeline([product], [run], [], limit).map((e) => ({
        ...e,
        id: `product-${e.id}`,
      })),
    );
  }

  return events
    .sort((a, b) => (b.timestamp ?? "").localeCompare(a.timestamp ?? ""))
    .slice(0, limit);
}

export function collectEvidenceSources(run?: AnalysisRun, product?: Product): string[] {
  const sources = new Set<string>();
  if (product) {
    const src = productSourceLabel(product);
    if (src) sources.add(src);
  }
  for (const recall of run?.recalls ?? []) {
    if (recall.matched && recall.source) sources.add(recall.source);
  }
  return Array.from(sources).sort((a, b) => a.localeCompare(b, "tr"));
}

export type SafetyOverviewStats = {
  recallMatches: number;
  criticalFindings: number;
  reviewRequired: number;
  recentSignals: number;
};

export function computeSafetyOverview(products: Product[], runs: AnalysisRun[]): SafetyOverviewStats {
  let recallMatches = 0;
  let criticalFindings = 0;
  let reviewRequired = 0;
  let recentSignals = 0;

  for (const run of runs) {
    if (run.recalls?.some((r) => r.matched)) recallMatches += 1;
    if (run.finalResult?.decision === "REVIEW_REQUIRED") reviewRequired += 1;
    for (const f of run.safetyFindings ?? []) {
      if (f.severity === "CRITICAL" || f.severity === "HIGH") {
        criticalFindings += 1;
        recentSignals += 1;
      }
    }
    if (run.status === "COMPLETED" && run.finalResult) recentSignals += 1;
  }

  void products;
  return { recallMatches, criticalFindings, reviewRequired, recentSignals };
}

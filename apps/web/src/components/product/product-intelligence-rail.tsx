import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { IntelligenceScoreRing } from "@/components/miyuna/intelligence-score-ring";
import { StatusRibbon } from "@/components/miyuna/status-ribbon";
import type { AnalysisRun } from "@/lib/analysis-types";
import {
  decisionLabel,
  riskLabel,
} from "@/lib/analysis-labels";
import {
  collectEvidenceSources,
  decisionToRibbon,
  recallToRibbon,
  riskToRibbon,
} from "@/lib/product-intelligence";
import type { Product } from "@/lib/product";

type ProductIntelligenceRailProps = {
  product: Product;
  run?: AnalysisRun | null;
  locale?: "tr" | "en";
};

export function ProductIntelligenceRail({ product, run, locale = "tr" }: ProductIntelligenceRailProps) {
  const final = run?.finalResult;
  const recallRibbon = recallToRibbon(run?.recalls);
  const sources = collectEvidenceSources(run ?? undefined, product);
  const matchedRecalls = run?.recalls?.filter((r) => r.matched) ?? [];

  return (
    <div className="space-y-4">
      {final && (
        <Card padding="md" className="text-center">
          <CardHeader className="mb-2">
            <CardTitle className="text-base">Analiz güveni</CardTitle>
          </CardHeader>
          <CardContent>
            <IntelligenceScoreRing
              score={Math.round(final.confidence * 100)}
              maxScore={100}
              label="Güven"
              severity={
                final.overallRisk === "HIGH" || final.overallRisk === "CRITICAL"
                  ? "risk"
                  : final.overallRisk === "MEDIUM"
                    ? "caution"
                    : "safe"
              }
              size="sm"
            />
          </CardContent>
        </Card>
      )}

      {final && (
        <Card padding="md">
          <CardTitle className="text-base">Mevcut risk</CardTitle>
          <div className="mt-4 space-y-3">
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm text-muted">Genel risk</span>
              <StatusRibbon variant={riskToRibbon(final.overallRisk)} label={riskLabel(final.overallRisk, locale)} size="sm" />
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm text-muted">Karar</span>
              <StatusRibbon variant={decisionToRibbon(final.decision)} label={decisionLabel(final.decision, locale)} size="sm" />
            </div>
            {final.decision === "REVIEW_REQUIRED" && (
              <p className="rounded-xl bg-info/10 px-3 py-2 text-xs text-info">İnsan incelemesi gerekli</p>
            )}
          </div>
        </Card>
      )}

      <Card padding="md">
        <CardTitle className="text-base">Recall durumu</CardTitle>
        {matchedRecalls.length === 0 ? (
          <p className="mt-3 text-sm text-text-secondary">Aktif geri çağırma eşleşmesi bulunmadı.</p>
        ) : (
          <div className="mt-3 space-y-2">
            {recallRibbon && <StatusRibbon variant={recallRibbon} size="sm" />}
            {matchedRecalls.map((r) => (
              <div key={`${r.source}-${r.sourceRecordId}`} className="rounded-xl bg-secondary/60 px-3 py-2 text-xs">
                <p className="font-medium text-ink">{r.source}</p>
                {r.reference && <p className="mt-0.5 text-muted">{r.reference}</p>}
                {r.requiresReview && (
                  <StatusRibbon variant="review" size="sm" label="İnceleme gerekli" className="mt-2" />
                )}
              </div>
            ))}
          </div>
        )}
      </Card>

      {sources.length > 0 && (
        <Card padding="md">
          <CardTitle className="text-base">Veri kaynakları</CardTitle>
          <ul className="mt-3 space-y-1.5">
            {sources.map((src) => (
              <li key={src} className="rounded-lg bg-secondary/50 px-3 py-1.5 text-xs font-medium text-text-secondary">
                {src}
              </li>
            ))}
          </ul>
        </Card>
      )}
    </div>
  );
}

"use client";

import Link from "next/link";
import { modelDisplayName, type RunLlmSummary } from "@/lib/llm-events";

type ModelRouteStripProps = {
  orgId: string;
  defaultModelKey: string;
  fallbackModelKey: string;
  modelNames: Record<string, string>;
  runSummary?: RunLlmSummary | null;
};

export function ModelRouteStrip({
  orgId,
  defaultModelKey,
  fallbackModelKey,
  modelNames,
  runSummary,
}: ModelRouteStripProps) {
  const fast = modelDisplayName(defaultModelKey, modelNames);
  const heavy = modelDisplayName(fallbackModelKey, modelNames);

  return (
    <div className="rounded-2xl border border-sand bg-cream/70 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="kicker">Model rotası</p>
          <p className="mt-2 text-sm leading-relaxed text-muted">
            Analizler önce <span className="font-semibold text-ink">{fast}</span> ile başlar. Uzun veya karmaşık
            adımlarda otomatik olarak <span className="font-semibold text-ink">{heavy}</span> modeline yükseltilir.
          </p>
        </div>
        <Link href={`/org/${orgId}/llm`} className="btn-ghost !px-3 !py-1.5 text-xs">
          Model ayarları
        </Link>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-2 text-xs">
        <span className="badge-forest">{fast}</span>
        <span className="text-muted">→</span>
        <span className="badge-clay">{heavy}</span>
        {runSummary?.escalated && <span className="badge-clay">Bu koşuda yükseltme yapıldı</span>}
        {runSummary?.fallbackUsed && !runSummary.escalated && (
          <span className="badge-muted">Yedek model kullanıldı</span>
        )}
      </div>

      {runSummary && runSummary.llmSteps > 0 && (
        <dl className="mt-4 grid gap-2 border-t border-sand pt-3 text-xs sm:grid-cols-3">
          <div>
            <dt className="text-muted">LLM adımı</dt>
            <dd className="font-semibold">{runSummary.llmSteps}</dd>
          </div>
          <div>
            <dt className="text-muted">Girdi token</dt>
            <dd className="font-semibold">{runSummary.totalInputTokens.toLocaleString()}</dd>
          </div>
          <div>
            <dt className="text-muted">Çıktı token</dt>
            <dd className="font-semibold">{runSummary.totalOutputTokens.toLocaleString()}</dd>
          </div>
        </dl>
      )}
    </div>
  );
}

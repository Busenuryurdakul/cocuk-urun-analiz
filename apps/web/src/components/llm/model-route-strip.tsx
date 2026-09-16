"use client";

import Link from "next/link";
import { useLocale } from "@/lib/i18n/locale-provider";
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
  const { t } = useLocale();
  const fastKey = defaultModelKey.trim() || "careful_analyst";
  const heavyKey = fallbackModelKey.trim() || "result_analyst";
  const fast = modelDisplayName(fastKey, modelNames);
  const heavy = modelDisplayName(heavyKey, modelNames);

  return (
    <div className="rounded-2xl border border-sand bg-cream/70 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="kicker">{t("modelRoute.kicker")}</p>
          <p className="mt-2 text-sm leading-relaxed text-muted">
            {t("modelRoute.intro", { fast, heavy })}
          </p>
        </div>
        <Link href={`/org/${orgId}/llm`} className="btn-ghost !px-3 !py-1.5 text-xs">
          {t("modelRoute.settings")}
        </Link>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-2 text-xs">
        <span className="badge-forest">{fast}</span>
        <span className="text-muted">→</span>
        <span className="badge-clay">{heavy}</span>
        {runSummary?.escalated && <span className="badge-clay">{t("modelRoute.runEscalated")}</span>}
        {runSummary?.fallbackUsed && !runSummary.escalated && (
          <span className="badge-muted">{t("modelRoute.runFallback")}</span>
        )}
      </div>

      {runSummary && runSummary.llmSteps > 0 && (
        <dl className="mt-4 grid gap-2 border-t border-sand pt-3 text-xs sm:grid-cols-3">
          <div>
            <dt className="text-muted">{t("modelRoute.llmStep")}</dt>
            <dd className="font-semibold">{runSummary.llmSteps}</dd>
          </div>
          <div>
            <dt className="text-muted">{t("modelRoute.inputTokens")}</dt>
            <dd className="font-semibold">{runSummary.totalInputTokens.toLocaleString()}</dd>
          </div>
          <div>
            <dt className="text-muted">{t("modelRoute.outputTokens")}</dt>
            <dd className="font-semibold">{runSummary.totalOutputTokens.toLocaleString()}</dd>
          </div>
        </dl>
      )}
    </div>
  );
}

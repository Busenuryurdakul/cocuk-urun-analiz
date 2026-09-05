"use client";

import Link from "next/link";

export type UsageDashboardData = {
  summary: {
    callCount: number;
    inputTokens: number;
    outputTokens: number;
    totalTokens: number;
    estimatedCostUsd: number;
    fallbackCount: number;
  };
  byModel: Array<{
    modelKey: string;
    displayName: string;
    callCount: number;
    inputTokens: number;
    outputTokens: number;
    totalTokens: number;
    estimatedCostUsd: number;
  }>;
  recentCalls: Array<{
    id: string;
    modelKey: string;
    personaKey: string;
    routingReason: string;
    fallbackUsed: boolean;
    inputTokens: number;
    outputTokens: number;
    latencyMs: number;
    status: string;
    createdAt: string;
  }>;
};

type UsagePanelProps = {
  orgId: string;
  usage: UsageDashboardData;
  title?: string;
  compact?: boolean;
  periodLabel?: string;
};

function isEscalationReason(reason: string): boolean {
  const lowered = reason.toLowerCase();
  return lowered.includes("escalat") || lowered.includes("deep_analysis") || lowered.includes("quality");
}

export function UsagePanel({
  orgId,
  usage,
  title = "Model kullanımı",
  compact = false,
  periodLabel = "bu ay",
}: UsagePanelProps) {
  const escalationCalls = usage.recentCalls.filter((call) => isEscalationReason(call.routingReason)).length;

  return (
    <section className={compact ? "space-y-4" : "card space-y-5"}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="kicker">LLM kullanımı</p>
          <h2 className="mt-1 font-display text-2xl">{title}</h2>
          <p className="mt-1 text-sm text-muted">Cursor benzeri özet — {periodLabel}</p>
        </div>
        {!compact && (
          <Link href={`/org/${orgId}/llm`} className="btn-secondary !px-3 !py-1.5 text-xs">
            Detaylı panel
          </Link>
        )}
      </div>

      <div className={`grid gap-3 ${compact ? "sm:grid-cols-2" : "sm:grid-cols-2 lg:grid-cols-4"}`}>
        <UsageStat label="Toplam token" value={usage.summary.totalTokens.toLocaleString()} />
        <UsageStat label="Çağrı" value={String(usage.summary.callCount)} />
        <UsageStat
          label="Yükseltme / fallback"
          value={String(Math.max(usage.summary.fallbackCount, escalationCalls))}
        />
        <UsageStat label="Tahmini maliyet" value={`$${usage.summary.estimatedCostUsd.toFixed(4)}`} />
      </div>

      {!compact && (
        <div className="grid gap-5 lg:grid-cols-2">
          <div>
            <h3 className="mb-3 text-sm font-semibold">Modele göre</h3>
            <ul className="space-y-2">
              {usage.byModel.map((row) => (
                <li
                  key={row.modelKey}
                  className="flex items-center justify-between rounded-2xl border border-sand bg-cream/50 px-4 py-3 text-sm"
                >
                  <div>
                    <p className="font-medium">{row.displayName || row.modelKey}</p>
                    <p className="text-xs text-muted">{row.callCount} çağrı</p>
                  </div>
                  <p className="font-semibold text-forest">{row.totalTokens.toLocaleString()} tok</p>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h3 className="mb-3 text-sm font-semibold">Son çağrılar</h3>
            <ul className="space-y-2">
              {usage.recentCalls.map((call) => (
                <li key={call.id} className="rounded-2xl border border-sand bg-cream/50 px-4 py-3 text-xs">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-semibold">{call.modelKey}</span>
                      {call.fallbackUsed && <span className="badge-muted">Fallback</span>}
                      {isEscalationReason(call.routingReason) && <span className="badge-clay">Yükseltme</span>}
                    </div>
                    <span>{call.inputTokens + call.outputTokens} tok</span>
                  </div>
                  <p className="mt-1 text-muted">{call.routingReason}</p>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </section>
  );
}

function UsageStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-sand bg-cream/60 px-4 py-3">
      <p className="text-xs text-muted">{label}</p>
      <p className="mt-1 font-display text-2xl text-forest">{value}</p>
    </div>
  );
}

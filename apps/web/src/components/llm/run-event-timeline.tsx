"use client";

import Link from "next/link";
import {
  isLlmPhase,
  metadataMap,
  phaseLabel,
  type AgentRunEventLike,
} from "@/lib/llm-events";

type RunEventTimelineProps = {
  orgId: string;
  events: AgentRunEventLike[];
};

function phaseBadgeClass(phase: string): string {
  if (phase === "LLM_ESCALATED") return "badge-clay";
  if (phase === "LLM_FALLBACK_USED") return "badge-muted";
  if (phase === "LLM_FAILED" || phase === "RUN_FAILED") return "badge-clay";
  if (isLlmPhase(phase)) return "badge-forest";
  return "badge-muted";
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return timestamp;
  return date.toLocaleString("tr-TR", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function EventDetails({ phase, meta, orgId }: { phase: string; meta: Record<string, string>; orgId: string }) {
  if (phase === "LLM_ESCALATED") {
    return (
      <p className="mt-2 text-xs text-muted">
        {meta.fromModel && <>Kaynak model: <span className="font-medium text-ink">{meta.fromModel}</span>. </>}
        {meta.reason && <>Neden: {meta.reason}. </>}
        {meta.taskType && <>Görev: {meta.taskType}</>}
      </p>
    );
  }

  if (phase === "LLM_COMPLETED" || phase === "LLM_REQUESTED") {
    const rows = [
      meta.selectedModel && { label: "Model", value: meta.selectedModel },
      meta.taskType && { label: "Görev", value: meta.taskType },
      meta.routingReason && { label: "Yönlendirme", value: meta.routingReason },
      meta.inputTokens && { label: "Girdi", value: `${meta.inputTokens} tok` },
      meta.outputTokens && { label: "Çıktı", value: `${meta.outputTokens} tok` },
    ].filter(Boolean) as { label: string; value: string }[];

    if (rows.length === 0) return null;

    return (
      <dl className="mt-2 grid gap-1 text-xs sm:grid-cols-2">
        {rows.map((row) => (
          <div key={row.label} className="flex gap-2">
            <dt className="text-muted">{row.label}</dt>
            <dd className="font-medium text-ink">{row.value}</dd>
          </div>
        ))}
        {meta.llmCallId && (
          <div className="sm:col-span-2">
            <Link href={`/org/${orgId}/llm?call=${meta.llmCallId}`} className="link-quiet text-xs">
              Çağrı kaydını gör
            </Link>
          </div>
        )}
      </dl>
    );
  }

  if (phase === "LLM_FALLBACK_USED") {
    return (
      <p className="mt-2 text-xs text-muted">
        Yedek sağlayıcı: {meta.selectedProvider ?? meta.selectedModel ?? "bilinmiyor"}
      </p>
    );
  }

  if (phase === "LLM_FAILED" && meta.error) {
    return <p className="mt-2 text-xs text-red-700">{meta.error}</p>;
  }

  return null;
}

export function RunEventTimeline({ orgId, events }: RunEventTimelineProps) {
  if (events.length === 0) {
    return null;
  }

  return (
    <ol className="space-y-3 border-t border-sand pt-4">
      {events.map((ev) => {
        const meta = metadataMap(ev.metadata);
        return (
          <li key={`${ev.sequence}-${ev.phase}`} className="rounded-2xl border border-sand/80 bg-cream/60 px-4 py-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="badge-forest">#{ev.sequence}</span>
              <span className={`${phaseBadgeClass(ev.phase)} !text-[10px]`}>{phaseLabel(ev.phase)}</span>
              <span className="text-xs text-muted">{ev.status}</span>
              {ev.toolName && <span className="badge-muted !text-[10px]">{ev.toolName}</span>}
              {meta.escalationUsed === "true" && <span className="badge-clay !text-[10px]">Yükseltme</span>}
              {meta.fallbackUsed === "true" && <span className="badge-muted !text-[10px]">Fallback</span>}
            </div>
            <time className="mt-1 block text-[11px] text-muted">{formatTime(ev.timestamp)}</time>
            <EventDetails phase={ev.phase} meta={meta} orgId={orgId} />
          </li>
        );
      })}
    </ol>
  );
}

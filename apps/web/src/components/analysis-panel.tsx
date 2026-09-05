"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { graphqlRequest } from "@/lib/graphql";

type AnalysisRun = {
  id: string;
  status: string;
  currentPhase?: string | null;
  terminalReason?: string | null;
  terminalError?: string | null;
  traceId: string;
  clientRequestId: string;
  createdAt: string;
  completedAt?: string | null;
};

type AgentRunEvent = {
  id: string;
  sequence: number;
  phase: string;
  status: string;
  toolName?: string | null;
  timestamp: string;
  metadata: { key: string; value: string }[];
};

const TERMINAL = new Set(["COMPLETED", "FAILED", "REJECTED"]);

function backoffMs(attempt: number) {
  return Math.min(1000 * 2 ** attempt, 10000);
}

export function AnalysisPanel({
  orgId,
  productId,
  canStart,
  canCancel,
}: {
  orgId: string;
  productId: string;
  canStart: boolean;
  canCancel: boolean;
}) {
  const [run, setRun] = useState<AnalysisRun | null>(null);
  const [events, setEvents] = useState<AgentRunEvent[]>([]);
  const [afterSequence, setAfterSequence] = useState(0);
  const [view, setView] = useState<"idle" | "loading" | "polling" | "error" | "unauthorized">("idle");
  const [actionError, setActionError] = useState("");
  const [starting, setStarting] = useState(false);
  const clientRequestRef = useRef<string>("");
  const pollAttempt = useRef(0);

  const loadEvents = useCallback(
    async (runId: string, cursor: number) => {
      const data = await graphqlRequest<{ agentRunEvents: AgentRunEvent[] }>(
        `query($orgId: ID!, $runId: ID!, $after: Int!) {
          agentRunEvents(organizationId: $orgId, analysisRunId: $runId, afterSequence: $after, limit: 100) {
            id sequence phase status toolName timestamp metadata { key value }
          }
        }`,
        { orgId, runId, after: cursor },
      );
      const batch = data.agentRunEvents ?? [];
      if (batch.length === 0) return cursor;
      setEvents((prev) => {
        const merged = [...prev];
        for (const ev of batch) {
          if (!merged.some((x) => x.sequence === ev.sequence)) merged.push(ev);
        }
        merged.sort((a, b) => a.sequence - b.sequence);
        return merged;
      });
      return batch[batch.length - 1]?.sequence ?? cursor;
    },
    [orgId],
  );

  const refreshRun = useCallback(
    async (runId: string) => {
      const data = await graphqlRequest<{ agentRun: AnalysisRun }>(
        `query($orgId: ID!, $runId: ID!) {
          agentRun(organizationId: $orgId, analysisRunId: $runId) {
            id status currentPhase terminalReason terminalError traceId clientRequestId createdAt completedAt
          }
        }`,
        { orgId, runId },
      );
      setRun(data.agentRun);
      return data.agentRun;
    },
    [orgId],
  );

  useEffect(() => {
    if (!run || TERMINAL.has(run.status)) return;
    setView("polling");
    let cancelled = false;
    const tick = async () => {
      try {
        const latest = await refreshRun(run.id);
        const next = await loadEvents(run.id, afterSequence);
        if (next > afterSequence) {
          setAfterSequence(next);
          pollAttempt.current = 0;
        }
        if (latest && TERMINAL.has(latest.status)) {
          setView("idle");
          return;
        }
        if (!cancelled) setTimeout(tick, backoffMs(pollAttempt.current++));
      } catch (err) {
        if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
          setView("unauthorized");
        } else {
          setView("error");
        }
      }
    };
    void tick();
    return () => {
      cancelled = true;
    };
  }, [run, afterSequence, loadEvents, refreshRun]);

  async function startAnalysis() {
    if (!canStart || starting) return;
    setStarting(true);
    setActionError("");
    setEvents([]);
    setAfterSequence(0);
    clientRequestRef.current = crypto.randomUUID();
    try {
      const data = await graphqlRequest<{ startAgentRun: AnalysisRun }>(
        `mutation($input: StartAgentRunInput!) {
          startAgentRun(input: $input) {
            id status currentPhase traceId clientRequestId createdAt
          }
        }`,
        {
          input: {
            organizationId: orgId,
            productId,
            clientRequestId: clientRequestRef.current,
          },
        },
      );
      setRun(data.startAgentRun);
      setView("polling");
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Analiz başlatılamadı");
    } finally {
      setStarting(false);
    }
  }

  async function cancelAnalysis() {
    if (!run || !canCancel) return;
    setActionError("");
    try {
      const data = await graphqlRequest<{ cancelAgentRun: AnalysisRun }>(
        `mutation($input: CancelAgentRunInput!) {
          cancelAgentRun(input: $input) { id status currentPhase terminalReason }
        }`,
        { input: { organizationId: orgId, analysisRunId: run.id } },
      );
      setRun(data.cancelAgentRun);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "İptal başarısız");
    }
  }

  return (
    <section className="card space-y-4">
      <div>
        <h2 className="font-display text-2xl">Ürün analizi</h2>
        <p className="mt-1 text-sm text-muted">
          Import durumundan ayrıdır — bu bölüm canonical ürün üzerindeki Agent analiz koşusunu gösterir.
        </p>
      </div>

      <div className="flex flex-wrap gap-3">
        {canStart && (
          <button type="button" className="btn-primary" disabled={starting || (run != null && !TERMINAL.has(run.status))} onClick={() => void startAnalysis()}>
            {starting ? "Başlatılıyor…" : "Analizi Başlat"}
          </button>
        )}
        {run && canCancel && !TERMINAL.has(run.status) && (
          <button type="button" className="btn-secondary" onClick={() => void cancelAnalysis()}>
            Analizi İptal Et
          </button>
        )}
      </div>

      {actionError && <p className="alert-error">{actionError}</p>}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && (
        <AsyncView
          state="retry"
          retry={
            run ? (
              <button type="button" className="btn-secondary" onClick={() => void refreshRun(run.id)}>
                Tekrar dene
              </button>
            ) : undefined
          }
        />
      )}

      {run && (
        <dl className="grid gap-2 rounded-2xl bg-cream p-4 text-sm">
          <div className="flex justify-between gap-4">
            <dt className="text-muted">Durum</dt>
            <dd className="font-semibold">{run.status}</dd>
          </div>
          {run.currentPhase && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Faz</dt>
              <dd>{run.currentPhase}</dd>
            </div>
          )}
          {run.terminalReason && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Sonuç</dt>
              <dd>{run.terminalReason}</dd>
            </div>
          )}
        </dl>
      )}

      {view === "polling" && run && !TERMINAL.has(run.status) && <AsyncView state="loading" />}

      {events.length === 0 ? (
        run ? <AsyncView state="empty" empty={<p className="text-sm text-muted">Henüz analiz olayı yok.</p>} /> : null
      ) : (
        <ol className="space-y-2 border-t border-sand pt-4">
          {events.map((ev) => (
            <li key={ev.id} className="rounded-xl bg-cream px-3 py-2 text-xs">
              <div className="flex flex-wrap items-center gap-2">
                <span className="badge-forest">#{ev.sequence}</span>
                <span className="font-semibold">{ev.phase}</span>
                <span className="text-muted">{ev.status}</span>
                {ev.toolName && <span className="badge-muted">{ev.toolName}</span>}
              </div>
              <time className="mt-1 block text-muted">{ev.timestamp}</time>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { ModelRouteStrip } from "@/components/llm/model-route-strip";
import { RunEventTimeline } from "@/components/llm/run-event-timeline";
import { graphqlRequest } from "@/lib/graphql";
import { summarizeRunLlm } from "@/lib/llm-events";

type AnalysisRun = {
  id: string;
  status: string;
  currentPhase?: string | null;
  terminalReason?: string | null;
  terminalError?: string | null;
  traceId: string;
  clientRequestId: string;
  productId: string;
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

type OrgLlmContext = {
  defaultModelKey: string;
  fallbackModelKey: string;
  modelNames: Record<string, string>;
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
  const [llmContext, setLlmContext] = useState<OrgLlmContext>({
    defaultModelKey: "careful_analyst",
    fallbackModelKey: "result_analyst",
    modelNames: {},
  });
  const [view, setView] = useState<"idle" | "loading" | "polling" | "error" | "unauthorized">("idle");
  const [actionError, setActionError] = useState("");
  const [starting, setStarting] = useState(false);
  const clientRequestRef = useRef<string>("");
  const pollAttempt = useRef(0);

  const runSummary = useMemo(() => summarizeRunLlm(events), [events]);

  const loadLlmContext = useCallback(async () => {
    try {
      const data = await graphqlRequest<{
        llmOrgSettings: { defaultModelKey: string; fallbackModelKey: string } | null;
        llmModels: Array<{ modelKey: string; displayName: string }>;
      }>(
        `query($id: ID!) {
          llmOrgSettings(organizationId: $id) { defaultModelKey fallbackModelKey }
          llmModels(organizationId: $id) { modelKey displayName }
        }`,
        { id: orgId },
      );
      const names = Object.fromEntries(data.llmModels.map((model) => [model.modelKey, model.displayName]));
      setLlmContext({
        defaultModelKey: data.llmOrgSettings?.defaultModelKey ?? "careful_analyst",
        fallbackModelKey: data.llmOrgSettings?.fallbackModelKey ?? "result_analyst",
        modelNames: names,
      });
    } catch {
      // Optional context — analysis can proceed without it.
    }
  }, [orgId]);

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
            id status currentPhase terminalReason terminalError traceId clientRequestId productId createdAt completedAt
          }
        }`,
        { orgId, runId },
      );
      setRun(data.agentRun);
      return data.agentRun;
    },
    [orgId],
  );

  const hydrateLatestRun = useCallback(async () => {
    try {
      const data = await graphqlRequest<{ analysisRuns: AnalysisRun[] }>(
        `query($orgId: ID!) {
          analysisRuns(organizationId: $orgId, limit: 20) {
            id status currentPhase terminalReason traceId clientRequestId productId createdAt completedAt
          }
        }`,
        { orgId },
      );
      const latest = data.analysisRuns.find((item) => item.productId === productId);
      if (!latest) return;
      setRun(latest);
      const next = await loadEvents(latest.id, 0);
      setAfterSequence(next);
    } catch {
      // Ignore — user can start a fresh run.
    }
  }, [loadEvents, orgId, productId]);

  useEffect(() => {
    void loadLlmContext();
    void hydrateLatestRun();
  }, [hydrateLatestRun, loadLlmContext]);

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
            id status currentPhase traceId clientRequestId productId createdAt
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
    <section className="card space-y-5">
      <div>
        <p className="kicker">Agent analizi</p>
        <h2 className="mt-1 font-display text-2xl">Ürün analizi</h2>
        <p className="mt-1 text-sm text-muted">
          Import durumundan ayrıdır — bu bölüm canonical ürün üzerindeki Agent koşusunu, model yükseltmelerini ve
          token kullanımını gösterir.
        </p>
      </div>

      <ModelRouteStrip
        orgId={orgId}
        defaultModelKey={llmContext.defaultModelKey}
        fallbackModelKey={llmContext.fallbackModelKey}
        modelNames={llmContext.modelNames}
        runSummary={run ? runSummary : null}
      />

      <div className="flex flex-wrap gap-3">
        {canStart && (
          <button
            type="button"
            className="btn-primary"
            disabled={starting || (run != null && !TERMINAL.has(run.status))}
            onClick={() => void startAnalysis()}
          >
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
          {runSummary.escalated && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Model yükseltme</dt>
              <dd className="font-semibold text-clay">Ağır modele geçildi</dd>
            </div>
          )}
        </dl>
      )}

      {view === "polling" && run && !TERMINAL.has(run.status) && <AsyncView state="loading" />}

      {events.length === 0 ? (
        run ? <AsyncView state="empty" empty={<p className="text-sm text-muted">Henüz analiz olayı yok.</p>} /> : null
      ) : (
        <RunEventTimeline orgId={orgId} events={events} />
      )}
    </section>
  );
}

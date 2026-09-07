"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { ModelRouteStrip } from "@/components/llm/model-route-strip";
import { RunEventTimeline } from "@/components/llm/run-event-timeline";
import { graphqlRequest } from "@/lib/graphql";
import { summarizeRunLlm } from "@/lib/llm-events";

type ReviewInsight = {
  topic: string;
  count: number;
  summary: string;
  kind: string;
};

type SafetyFinding = {
  id: string;
  type: string;
  severity: string;
  confidence: number;
  rationale: string;
};

type RecallMatch = {
  source: string;
  sourceRecordId: string;
  matched: boolean;
  confidence: number;
  method: string;
  reference?: string | null;
  requiresReview: boolean;
};

type EvidenceItem = {
  source: string;
  claim: string;
  supportStatus?: string | null;
  reference?: string | null;
};

type FinalAnalysisResult = {
  schemaVersion: string;
  summary: string;
  overallRisk: string;
  confidence: number;
  decision: string;
  recommendation: string;
  limitations: string[];
  hallucinationFlags: string[];
  createdAt: string;
};

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
  reviewInsights?: ReviewInsight[];
  safetyFindings?: SafetyFinding[];
  recalls?: RecallMatch[];
  evidence?: EvidenceItem[];
  finalResult?: FinalAnalysisResult | null;
};

const RUN_FIELDS = `
  id status currentPhase terminalReason terminalError traceId clientRequestId productId createdAt completedAt
  reviewInsights { topic count summary kind }
  safetyFindings { id type severity confidence rationale }
  recalls { source sourceRecordId matched confidence method reference requiresReview }
  evidence { source claim supportStatus reference }
  finalResult { schemaVersion summary overallRisk confidence decision recommendation limitations hallucinationFlags createdAt }
`;

function decisionLabel(decision: string) {
  switch (decision) {
    case "ALLOW":
      return "İzin verildi";
    case "ALLOW_WITH_WARNING":
      return "Uyarı ile izin";
    case "REVIEW_REQUIRED":
      return "İnceleme gerekli";
    case "BLOCK":
      return "Engellendi";
    default:
      return decision;
  }
}

function isHighSeverity(severity: string) {
  return severity === "CRITICAL" || severity === "HIGH";
}

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
          agentRun(organizationId: $orgId, analysisRunId: $runId) { ${RUN_FIELDS} }
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
      const detailed = await refreshRun(latest.id);
      setRun(detailed ?? latest);
      const next = await loadEvents(latest.id, 0);
      setAfterSequence(next);
    } catch {
      // Ignore — user can start a fresh run.
    }
  }, [loadEvents, orgId, productId, refreshRun]);

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

      {run?.finalResult && (
        <div className="space-y-4 rounded-2xl bg-cream p-4 text-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">Politika sonucu</p>
          <dl className="grid gap-2">
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Karar</dt>
              <dd className={`font-semibold ${run.finalResult.decision === "BLOCK" ? "text-clay" : ""}`}>
                {decisionLabel(run.finalResult.decision)}
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Genel risk</dt>
              <dd className={isHighSeverity(run.finalResult.overallRisk) ? "font-semibold text-clay" : "font-semibold"}>
                {run.finalResult.overallRisk}
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Güven</dt>
              <dd>{Math.round(run.finalResult.confidence * 100)}%</dd>
            </div>
          </dl>
          <p>{run.finalResult.summary}</p>
          {run.finalResult.recommendation && (
            <p className="text-muted">{run.finalResult.recommendation}</p>
          )}
        </div>
      )}

      {run?.safetyFindings && run.safetyFindings.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">Güvenlik bulguları</h3>
          <ul className="space-y-2 text-sm">
            {run.safetyFindings.map((finding) => (
              <li
                key={finding.id}
                className={`rounded-2xl bg-cream p-3 ${isHighSeverity(finding.severity) ? "text-clay" : ""}`}
              >
                <p className="font-semibold">
                  {finding.severity} · {finding.type}
                </p>
                <p>{finding.rationale}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

      {run?.recalls && run.recalls.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">Geri çağırma eşleşmeleri</h3>
          <ul className="space-y-2 text-sm">
            {run.recalls.map((match) => (
              <li key={`${match.source}-${match.sourceRecordId}`} className="rounded-2xl bg-cream p-3">
                <p className="font-semibold">
                  {match.source} {match.requiresReview ? "(inceleme gerekli)" : "(doğrulandı)"}
                </p>
                {match.reference && <p className="text-muted">{match.reference}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {run?.evidence && (
        <div className="space-y-2">
          <h3 className="font-semibold">Kanıt</h3>
          <p className="text-sm text-muted">{run.evidence.length} kaynak</p>
          <ul className="space-y-2 text-sm">
            {run.evidence.slice(0, 8).map((item, index) => (
              <li key={`${item.source}-${index}`} className="rounded-2xl bg-cream p-3">
                <p className="font-semibold">{item.source}</p>
                <p>{item.claim}</p>
                <p className="text-muted">{item.supportStatus ?? "DURUM YOK"}</p>
                {item.reference && <p className="text-muted">{item.reference}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {run?.reviewInsights && run.reviewInsights.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">Yorum içgörüleri</h3>
          <ul className="space-y-2 text-sm">
            {run.reviewInsights.map((insight, index) => (
              <li key={`${insight.kind}-${insight.topic}-${index}`} className="rounded-2xl bg-cream p-3">
                <p className="font-semibold">
                  {insight.kind} · {insight.topic} ({insight.count})
                </p>
                <p className="text-muted">{insight.summary}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

      {run?.finalResult?.limitations && run.finalResult.limitations.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">Sınırlamalar</h3>
          <ul className="list-disc space-y-1 pl-5 text-sm text-muted">
            {run.finalResult.limitations.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
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

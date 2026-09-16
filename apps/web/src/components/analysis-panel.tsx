"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { ModelRouteStrip } from "@/components/llm/model-route-strip";
import { RunEventTimeline } from "@/components/llm/run-event-timeline";
import { TechnicalDetails } from "@/components/technical-details";
import {
  decisionLabel,
  findingRationale,
  findingTypeLabel,
  insightKindLabel,
  insightTopicLabel,
  isQualityFinding,
  localizePolicyText,
  riskLabel,
  statusLabel,
  supportStatusLabel,
} from "@/lib/analysis-labels";
import {
  graphqlErrorCode,
  graphqlErrorMessage,
  graphqlRequest,
  isAuthError,
  isComplianceErrorCode,
} from "@/lib/graphql";
import type { Locale } from "@/lib/i18n/locale-config";
import { useLocale } from "@/lib/i18n/locale-provider";
import { phaseLabel, summarizeRunLlm } from "@/lib/llm-events";
import { humanizeAnalysisError } from "@/lib/ui-labels";

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

function isHighSeverity(severity: string) {
  return severity === "CRITICAL" || severity === "HIGH";
}

function PolicyCopy({
  summary,
  recommendation,
  locale,
}: {
  summary: string;
  recommendation: string;
  locale: Locale;
}) {
  const localizedSummary = localizePolicyText(summary, locale);
  const localizedRecommendation = localizePolicyText(recommendation, locale);
  const showRecommendation =
    Boolean(localizedRecommendation) && !localizedSummary.includes(localizedRecommendation);
  return (
    <>
      <p>{localizedSummary}</p>
      {showRecommendation ? <p className="text-muted">{localizedRecommendation}</p> : null}
    </>
  );
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
  const { locale, t } = useLocale();
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
  const [actionErrorTechnical, setActionErrorTechnical] = useState("");
  const [actionErrorCode, setActionErrorCode] = useState("");
  const [starting, setStarting] = useState(false);
  const clientRequestRef = useRef<string>("");
  const previousRunRef = useRef<AnalysisRun | null>(null);
  const pollAttempt = useRef(0);
  const afterSequenceRef = useRef(0);
  const [pollStartedAt, setPollStartedAt] = useState<number | null>(null);
  const [eventsRunId, setEventsRunId] = useState<string | null>(null);

  const activeEvents = useMemo(() => {
    if (!run || eventsRunId !== run.id) return [];
    return events;
  }, [events, eventsRunId, run]);

  const runSummary = useMemo(() => {
    if (activeEvents.length === 0) return null;
    return summarizeRunLlm(activeEvents);
  }, [activeEvents]);

  const [awaitingAgentWake, setAwaitingAgentWake] = useState(false);

  useEffect(() => {
    if (!run || run.status !== "PENDING" || activeEvents.length > 0 || pollStartedAt == null) {
      setAwaitingAgentWake(false);
      return;
    }
    const elapsed = Date.now() - pollStartedAt;
    if (elapsed >= 20_000) {
      setAwaitingAgentWake(true);
      return;
    }
    const timer = window.setTimeout(() => setAwaitingAgentWake(true), 20_000 - elapsed);
    return () => window.clearTimeout(timer);
  }, [activeEvents.length, pollStartedAt, run?.id, run?.status]);

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
        defaultModelKey: data.llmOrgSettings?.defaultModelKey?.trim() || "careful_analyst",
        fallbackModelKey: data.llmOrgSettings?.fallbackModelKey?.trim() || "result_analyst",
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
      setEvents([]);
      setEventsRunId(latest.id);
      afterSequenceRef.current = 0;
      const next = await loadEvents(latest.id, 0);
      setAfterSequence(next);
      afterSequenceRef.current = next;
    } catch {
      // Ignore — user can start a fresh run.
    }
  }, [loadEvents, orgId, productId, refreshRun]);

  useEffect(() => {
    void loadLlmContext();
    void hydrateLatestRun();
    void fetch("/api/cron/keep-warm", { cache: "no-store" }).catch(() => {
      // Best-effort: wake Render agent/API before the user clicks start.
    });
  }, [hydrateLatestRun, loadLlmContext]);

  useEffect(() => {
    if (!run) {
      return;
    }
    if (!TERMINAL.has(run.status)) {
      setActionError("");
      setActionErrorTechnical("");
      setActionErrorCode("");
      return;
    }
    if (run.status === "FAILED" || run.status === "REJECTED") {
      if (run.terminalReason === "ORCHESTRATOR_DISPATCH_FAILED") {
        setActionError(t("analysis.wakeFailed"));
        setActionErrorTechnical(run.terminalError?.trim() ?? "");
        setActionErrorCode(run.terminalReason);
      } else if (
        run.terminalReason === "COMPLIANCE_BLOCKED" ||
        run.terminalError?.toLowerCase().includes("blocked by compliance")
      ) {
        setActionError(t("analysis.complianceBlockedRun"));
        setActionErrorTechnical(run.terminalError?.trim() ?? "");
        setActionErrorCode("COMPLIANCE_BLOCKED");
      } else {
        const humanized = humanizeAnalysisError(
          run.terminalError?.trim() || run.terminalReason?.trim(),
          locale,
        );
        setActionError(humanized.message || t("analysis.incomplete"));
        setActionErrorTechnical(humanized.technical ?? run.terminalError?.trim() ?? "");
        setActionErrorCode(run.terminalReason ?? "");
      }
      setView("idle");
      return;
    }
    setView("idle");
  }, [locale, run?.id, run?.status, run?.terminalError, run?.terminalReason, t]);

  useEffect(() => {
    if (!run || TERMINAL.has(run.status)) return;
    const runId = run.id;
    setView("polling");
    setPollStartedAt(Date.now());
    afterSequenceRef.current = afterSequence;
    let cancelled = false;
    const tick = async () => {
      try {
        const latest = await refreshRun(runId);
        const next = await loadEvents(runId, afterSequenceRef.current);
        if (next > afterSequenceRef.current) {
          afterSequenceRef.current = next;
          setAfterSequence(next);
          setEventsRunId(runId);
          pollAttempt.current = 0;
        }
        if (latest && TERMINAL.has(latest.status)) {
          setView("idle");
          return;
        }
        if (!cancelled) setTimeout(tick, backoffMs(pollAttempt.current++));
      } catch (err) {
        if (isAuthError(err)) {
          setView("unauthorized");
          return;
        }
        if (!cancelled) setTimeout(tick, backoffMs(pollAttempt.current++));
      }
    };
    void tick();
    return () => {
      cancelled = true;
    };
  }, [run?.id, run?.status, loadEvents, refreshRun]);

  async function ensureDataProcessingConsent() {
    try {
      await graphqlRequest(
        `mutation($input: GrantConsentInput!) { grantConsent(input: $input) { id } }`,
        { input: { purpose: "DATA_PROCESSING", organizationId: orgId } },
      );
    } catch {
      // Consent may already exist; startAgentRun validates again.
    }
  }

  async function startAnalysis() {
    if (!canStart || starting) return;
    setStarting(true);
    setActionError("");
    setActionErrorTechnical("");
    setActionErrorCode("");
    setEvents([]);
    setEventsRunId(null);
    setAfterSequence(0);
    afterSequenceRef.current = 0;
    pollAttempt.current = 0;
    setPollStartedAt(null);
    setView("loading");
    previousRunRef.current = run;
    setRun(null);
    clientRequestRef.current = crypto.randomUUID();
    try {
      await ensureDataProcessingConsent();
      void fetch("/api/cron/keep-warm", { cache: "no-store" }).catch(() => {});
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
      setEventsRunId(data.startAgentRun.id);
      setView("polling");
    } catch (err) {
      if (isAuthError(err)) {
        setRun(previousRunRef.current);
        setView("unauthorized");
        setActionError("");
        setActionErrorTechnical("");
        setActionErrorCode("");
      } else {
        const code = graphqlErrorCode(err);
        const message = graphqlErrorMessage(err, t("analysis.startFailed"), locale);
        setActionError(message);
        setActionErrorTechnical(code && !message.includes(code) ? code : "");
        setActionErrorCode(code ?? "");
        setView("idle");
      }
    } finally {
      setStarting(false);
    }
  }

  async function cancelAnalysis() {
    if (!run || !canCancel) return;
    setActionError("");
    setActionErrorTechnical("");
    setActionErrorCode("");
    try {
      const data = await graphqlRequest<{ cancelAgentRun: AnalysisRun }>(
        `mutation($input: CancelAgentRunInput!) {
          cancelAgentRun(input: $input) { id status currentPhase terminalReason }
        }`,
        { input: { organizationId: orgId, analysisRunId: run.id } },
      );
      setRun(data.cancelAgentRun);
    } catch (err) {
      const code = graphqlErrorCode(err);
      setActionError(graphqlErrorMessage(err, t("analysis.cancelFailed"), locale));
      setActionErrorCode(code ?? "");
    }
  }

  return (
    <section className="card min-w-0 space-y-5 overflow-hidden">
      <div>
        <p className="kicker">{t("analysis.kicker")}</p>
        <h2 className="mt-1 font-display text-2xl">{t("analysis.title")}</h2>
        <p className="mt-1 text-sm text-muted">{t("analysis.intro")}</p>
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
            {starting ? t("analysis.starting") : t("analysis.start")}
          </button>
        )}
        {run && canCancel && !TERMINAL.has(run.status) && (
          <button type="button" className="btn-secondary" onClick={() => void cancelAnalysis()}>
            {t("analysis.cancel")}
          </button>
        )}
      </div>

      {actionError && (!run || TERMINAL.has(run.status)) && (
        <div className="alert-error space-y-2">
          <p>{actionError}</p>
          {actionErrorTechnical && (
            <TechnicalDetails label={t("analysis.technicalDetails")} value={actionErrorTechnical} />
          )}
          {(isComplianceErrorCode(actionErrorCode) ||
            actionError.toLowerCase().includes("consent") ||
            actionError.toLowerCase().includes("compliance") ||
            actionError.includes("onay") ||
            actionError.includes("Uyumluluk")) && (
            <p className="text-sm">
              <Link href={`/org/${orgId}/compliance`} className="font-semibold underline">
                {t("analysis.complianceLink")}
              </Link>
            </p>
          )}
        </div>
      )}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {run && (
        <dl className="grid gap-2 rounded-2xl bg-cream p-4 text-sm">
          <div className="flex justify-between gap-4">
            <dt className="text-muted">{t("analysis.status")}</dt>
            <dd className="font-semibold">{statusLabel(run.status, locale)}</dd>
          </div>
          {run.currentPhase && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">{t("analysis.phase")}</dt>
              <dd>{phaseLabel(run.currentPhase, locale)}</dd>
            </div>
          )}
          {TERMINAL.has(run.status) && run.terminalReason && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">{t("analysis.result")}</dt>
              <dd>{statusLabel(run.terminalReason, locale)}</dd>
            </div>
          )}
          {TERMINAL.has(run.status) && run.terminalError && (
            <div className="col-span-full space-y-1">
              <dt className="text-muted">{t("analysis.error")}</dt>
              <dd className="text-clay">
                {humanizeAnalysisError(run.terminalError, locale).message}
              </dd>
              <TechnicalDetails label={t("analysis.technicalDetails")} value={run.terminalError} />
            </div>
          )}
          {runSummary?.escalated && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">{t("analysis.modelEscalation")}</dt>
              <dd className="font-semibold text-clay">{t("analysis.modelEscalated")}</dd>
            </div>
          )}
        </dl>
      )}

      {run && TERMINAL.has(run.status) && run.finalResult && (
        <div className="space-y-4 rounded-2xl bg-cream p-4 text-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">{t("analysis.policyResult")}</p>
          <dl className="grid gap-2">
            <div className="flex justify-between gap-4">
              <dt className="text-muted">{t("analysis.decision")}</dt>
              <dd className={`font-semibold ${run.finalResult.decision === "BLOCK" ? "text-clay" : ""}`}>
                {decisionLabel(run.finalResult.decision, locale)}
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted">{t("analysis.overallRisk")}</dt>
              <dd className={isHighSeverity(run.finalResult.overallRisk) ? "font-semibold text-clay" : "font-semibold"}>
                {riskLabel(run.finalResult.overallRisk, locale)}
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-muted">{t("analysis.confidence")}</dt>
              <dd>{Math.round(run.finalResult.confidence * 100)}%</dd>
            </div>
          </dl>
          <PolicyCopy
            summary={run.finalResult.summary}
            recommendation={run.finalResult.recommendation}
            locale={locale}
          />
        </div>
      )}

      {run && TERMINAL.has(run.status) && run.safetyFindings && run.safetyFindings.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">
            {run.safetyFindings.every((finding) => isQualityFinding(finding.type))
              ? t("analysis.evidenceStatus")
              : t("analysis.safetyFindings")}
          </h3>
          <ul className="space-y-2 text-sm">
            {run.safetyFindings.map((finding) => {
              const rationale = findingRationale(finding.type, finding.rationale, locale);
              return (
                <li
                  key={finding.id}
                  className={`rounded-2xl bg-cream p-3 ${isHighSeverity(finding.severity) ? "text-clay" : ""}`}
                >
                  <p className="font-semibold">
                    {riskLabel(finding.severity, locale)} · {findingTypeLabel(finding.type, locale)}
                  </p>
                  {rationale ? <p>{rationale}</p> : null}
                </li>
              );
            })}
          </ul>
        </div>
      )}

      {run && TERMINAL.has(run.status) && run.recalls && run.recalls.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">{t("analysis.recalls")}</h3>
          <ul className="space-y-2 text-sm">
            {run.recalls.map((match) => (
              <li key={`${match.source}-${match.sourceRecordId}`} className="rounded-2xl bg-cream p-3">
                <p className="font-semibold">
                  {match.source}{" "}
                  {match.requiresReview ? t("analysis.recallReview") : t("analysis.recallConfirmed")}
                </p>
                {match.reference && <p className="text-muted">{match.reference}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {run && TERMINAL.has(run.status) && run.evidence && run.evidence.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">{t("analysis.evidence")}</h3>
          <p className="text-sm text-muted">{t("analysis.evidenceCount", { count: run.evidence.length })}</p>
          <ul className="space-y-2 text-sm">
            {run.evidence.slice(0, 8).map((item, index) => (
              <li key={`${item.source}-${index}`} className="rounded-2xl bg-cream p-3">
                <p className="font-semibold">{item.source}</p>
                <p>{item.claim}</p>
                <p className="text-muted">
                  {item.supportStatus ? supportStatusLabel(item.supportStatus, locale) : t("analysis.noSupportStatus")}
                </p>
                {item.reference && <p className="text-muted">{item.reference}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {run && TERMINAL.has(run.status) && run.reviewInsights && run.reviewInsights.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">{t("analysis.reviewInsights")}</h3>
          <ul className="space-y-2 text-sm">
            {run.reviewInsights.map((insight, index) => (
              <li key={`${insight.kind}-${insight.topic}-${index}`} className="rounded-2xl bg-cream p-3">
                <p className="font-semibold">
                  {insightKindLabel(insight.kind, locale)} · {insightTopicLabel(insight.topic, locale)} ({insight.count})
                </p>
                <p className="text-muted">{localizePolicyText(insight.summary, locale)}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

      {run && TERMINAL.has(run.status) && run.finalResult?.limitations && run.finalResult.limitations.length > 0 && (
        <div className="space-y-2">
          <h3 className="font-semibold">{t("analysis.limitations")}</h3>
          <ul className="list-disc space-y-1 pl-5 text-sm text-muted">
            {run.finalResult.limitations.map((item) => (
              <li key={item}>{localizePolicyText(item, locale)}</li>
            ))}
          </ul>
        </div>
      )}

      {view === "polling" && run && !TERMINAL.has(run.status) && (
        <AsyncView
          state="loading"
          loading={
            <div role="status" className="card text-center">
              <div className="mx-auto mb-3 h-8 w-8 animate-pulse rounded-full bg-forest-soft" />
              <p className="text-sm text-muted">
                {awaitingAgentWake ? t("analysis.wakeWait") : t("analysis.polling")}
              </p>
            </div>
          }
        />
      )}

      {activeEvents.length === 0 ? (
        run && view !== "polling" && view !== "loading" ? (
          <AsyncView state="empty" empty={<p className="text-sm text-muted">Henüz analiz olayı yok.</p>} />
        ) : null
      ) : (
        <RunEventTimeline orgId={orgId} events={activeEvents} />
      )}
    </section>
  );
}

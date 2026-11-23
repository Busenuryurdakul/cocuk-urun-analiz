/** Shared GraphQL shapes for analysis runs — read-only, no behavior change. */

export type ReviewInsight = {
  topic: string;
  count: number;
  summary: string;
  kind: string;
};

export type SafetyFinding = {
  id: string;
  type: string;
  severity: string;
  confidence: number;
  rationale: string;
};

export type RecallMatch = {
  source: string;
  sourceRecordId: string;
  matched: boolean;
  confidence: number;
  method: string;
  reference?: string | null;
  requiresReview: boolean;
};

export type FinalAnalysisResult = {
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

export type AnalysisRun = {
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
  finalResult?: FinalAnalysisResult | null;
};

export type MarketplaceImportRun = {
  id: string;
  source: string;
  sourceUrl?: string | null;
  status: string;
  recordsAccepted: number;
  recordsRejected: number;
  createdAt: string;
};

export const ANALYSIS_RUN_FIELDS = `
  id status currentPhase terminalReason terminalError traceId clientRequestId productId createdAt completedAt
  reviewInsights { topic count summary kind }
  safetyFindings { id type severity confidence rationale }
  recalls { source sourceRecordId matched confidence method reference requiresReview }
  finalResult { schemaVersion summary overallRisk confidence decision recommendation limitations hallucinationFlags createdAt }
`;

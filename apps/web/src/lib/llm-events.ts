export type EventMetadataEntry = { key: string; value: string };

export type AgentRunEventLike = {
  sequence: number;
  phase: string;
  status: string;
  toolName?: string | null;
  timestamp: string;
  metadata: EventMetadataEntry[];
};

export function metadataMap(metadata: EventMetadataEntry[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const entry of metadata) {
    out[entry.key] = entry.value;
  }
  return out;
}

export const PHASE_LABELS: Record<string, string> = {
  LLM_REQUESTED: "Model isteği",
  LLM_COMPLETED: "Model yanıtı",
  LLM_FAILED: "Model hatası",
  LLM_FALLBACK_USED: "Yedek modele geçildi",
  LLM_ESCALATED: "Ağır modele yükseltildi",
  RUN_STARTED: "Analiz başladı",
  RUN_COMPLETED: "Analiz tamamlandı",
  RUN_FAILED: "Analiz başarısız",
  RUN_CANCELLED: "Analiz iptal edildi",
  TOOL_REQUESTED: "Araç isteği",
  TOOL_COMPLETED: "Araç tamamlandı",
  TOOL_FAILED: "Araç hatası",
  OBSERVATION_CREATED: "Gözlem kaydı",
  PLAN_CREATED: "Plan oluşturuldu",
};

export function phaseLabel(phase: string): string {
  return PHASE_LABELS[phase] ?? phase.replaceAll("_", " ").toLowerCase();
}

export function isLlmPhase(phase: string): boolean {
  return phase.startsWith("LLM_");
}

export type RunLlmSummary = {
  escalated: boolean;
  fallbackUsed: boolean;
  primaryModel?: string;
  escalatedModel?: string;
  totalInputTokens: number;
  totalOutputTokens: number;
  llmSteps: number;
};

export function summarizeRunLlm(events: AgentRunEventLike[]): RunLlmSummary {
  let escalated = false;
  let fallbackUsed = false;
  let primaryModel: string | undefined;
  let escalatedModel: string | undefined;
  let totalInputTokens = 0;
  let totalOutputTokens = 0;
  let llmSteps = 0;

  for (const ev of events) {
    const meta = metadataMap(ev.metadata);
    if (ev.phase === "LLM_ESCALATED") {
      escalated = true;
      escalatedModel = meta.fromModel ?? meta.selectedModel ?? escalatedModel;
    }
    if (ev.phase === "LLM_FALLBACK_USED" || meta.fallbackUsed === "true") {
      fallbackUsed = true;
    }
    if (ev.phase === "LLM_COMPLETED") {
      llmSteps += 1;
      primaryModel = primaryModel ?? meta.selectedModel;
      if (meta.escalationUsed === "true") escalated = true;
      totalInputTokens += Number(meta.inputTokens ?? 0);
      totalOutputTokens += Number(meta.outputTokens ?? 0);
    }
  }

  return {
    escalated,
    fallbackUsed,
    primaryModel,
    escalatedModel,
    totalInputTokens,
    totalOutputTokens,
    llmSteps,
  };
}

export function modelDisplayName(modelKey: string, registry: Record<string, string>): string {
  return registry[modelKey] ?? modelKey;
}

export function monthStartIsoDate(): string {
  const now = new Date();
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1)).toISOString().slice(0, 10);
}

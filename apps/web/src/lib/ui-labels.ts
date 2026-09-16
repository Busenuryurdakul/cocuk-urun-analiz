import { riskLabel as analysisRiskLabel } from "@/lib/analysis-labels";
import { resolveLocaleMap } from "@/lib/i18n/helpers";
import type { Locale } from "@/lib/i18n/locale-config";
import { localeTag } from "@/lib/i18n/locale-config";

type LabelTable = Partial<Record<Locale, Record<string, string>>> & { en: Record<string, string>; tr: Record<string, string> };

const ROLE_LABELS: LabelTable = {
  tr: { OWNER: "Sahip", ADMIN: "Yönetici", ANALYST: "Analist", VIEWER: "Görüntüleyici" },
  en: { OWNER: "Owner", ADMIN: "Admin", ANALYST: "Analyst", VIEWER: "Viewer" },
};

const WORKSPACE_TYPE_LABELS: LabelTable = {
  tr: { PERSONAL: "Kişisel", ORGANIZATION: "Organizasyon" },
  en: { PERSONAL: "Personal", ORGANIZATION: "Organization" },
};

const ACTIVITY_LABELS: LabelTable = {
  tr: {
    LOGIN: "Giriş yapıldı",
    LOGOUT: "Çıkış yapıldı",
    REGISTER: "Kayıt olundu",
    MFA_ENABLED: "Çok aşamalı doğrulama açıldı",
    MFA_DISABLED: "Çok aşamalı doğrulama kapatıldı",
    DEVICE_REVOKED: "Cihaz oturumu sonlandırıldı",
    PASSWORD_CHANGED: "Şifre değiştirildi",
  },
  en: {
    LOGIN: "Signed in",
    LOGOUT: "Signed out",
    REGISTER: "Registered",
    MFA_ENABLED: "MFA enabled",
    MFA_DISABLED: "MFA disabled",
    DEVICE_REVOKED: "Device session ended",
    PASSWORD_CHANGED: "Password changed",
  },
};

const COMPLIANCE_PROFILE_LABELS: LabelTable = {
  tr: { KVKK: "KVKK", GDPR: "GDPR", BOTH: "KVKK ve GDPR" },
  en: { KVKK: "KVKK", GDPR: "GDPR", BOTH: "KVKK and GDPR" },
};

const POLICY_STATUS_LABELS: LabelTable = {
  tr: { PUBLISHED: "Yayımlandı", OFF: "Kapalı", DRAFT: "Taslak", ACTIVE: "Aktif" },
  en: { PUBLISHED: "Published", OFF: "Off", DRAFT: "Draft", ACTIVE: "Active" },
};

const LLM_STATUS_LABELS: LabelTable = {
  tr: {
    ACTIVE: "Aktif",
    INACTIVE: "Pasif",
    DRAFT: "Taslak",
    VALIDATED: "Doğrulandı",
    PUBLISHED: "Yayımlandı",
    HEALTHY: "Sağlıklı",
    UNHEALTHY: "Sağlıksız",
    UNKNOWN: "Bilinmiyor",
  },
  en: {
    ACTIVE: "Active",
    INACTIVE: "Inactive",
    DRAFT: "Draft",
    VALIDATED: "Validated",
    PUBLISHED: "Published",
    HEALTHY: "Healthy",
    UNHEALTHY: "Unhealthy",
    UNKNOWN: "Unknown",
  },
};

const EVENT_STATUS_LABELS: LabelTable = {
  tr: {
    COMPLETED: "Tamamlandı",
    FAILED: "Başarısız",
    RUNNING: "Çalışıyor",
    PENDING: "Bekliyor",
    SUCCESS: "Başarılı",
    REJECTED: "Reddedildi",
    CANCELLED: "İptal edildi",
  },
  en: {
    COMPLETED: "Completed",
    FAILED: "Failed",
    RUNNING: "Running",
    PENDING: "Pending",
    SUCCESS: "Success",
    REJECTED: "Rejected",
    CANCELLED: "Cancelled",
  },
};

const TOOL_LABELS: LabelTable = {
  tr: {
    POLICY_EVALUATOR: "Politika değerlendiricisi",
    PII_REDACTOR: "Kişisel veri maskeleme",
    REVIEW_ANALYZER: "Yorum analizcisi",
    RECALL_CHECKER: "Geri çağırma denetimi",
    EVIDENCE_COLLECTOR: "Kanıt toplayıcı",
    SAFETY_ANALYZER: "Güvenlik analizcisi",
  },
  en: {
    POLICY_EVALUATOR: "Policy evaluator",
    PII_REDACTOR: "PII redactor",
    REVIEW_ANALYZER: "Review analyzer",
    RECALL_CHECKER: "Recall checker",
    EVIDENCE_COLLECTOR: "Evidence collector",
    SAFETY_ANALYZER: "Safety analyzer",
  },
};

const TASK_TYPE_LABELS: LabelTable = {
  tr: {
    analysis: "Analiz",
    review: "İnceleme",
    reviewer: "İnceleme",
    deep_analysis: "Derin analiz",
    rating_prediction: "Puan tahmini",
    summarizer: "Özet",
    planner: "Planlama",
  },
  en: {
    analysis: "Analysis",
    review: "Review",
    reviewer: "Review",
    deep_analysis: "Deep analysis",
    rating_prediction: "Rating prediction",
    summarizer: "Summary",
    planner: "Planning",
  },
};

const MODEL_KEY_LABELS: LabelTable = {
  tr: {
    careful_analyst: "Dikkatli analist",
    result_analyst: "Sonuç analisti",
  },
  en: {
    careful_analyst: "Careful analyst",
    result_analyst: "Result analyst",
  },
};

const ROUTING_KEY_LABELS: LabelTable = {
  tr: {
    task: "Görev",
    taskType: "Görev",
    primary: "Birincil model",
    fallback: "Yedek model",
    selectedModel: "Seçilen model",
    fromModel: "Kaynak model",
    persona: "Davranış profili",
    policy: "Politika sürümü",
    evidence_required: "Kanıt gerekli",
    quality_escalation: "Kalite yükseltmesi",
    safety: "Güvenlik riski",
    reason: "Neden",
    routingReason: "Yönlendirme",
  },
  en: {
    task: "Task",
    taskType: "Task",
    primary: "Primary model",
    fallback: "Fallback model",
    selectedModel: "Selected model",
    fromModel: "Source model",
    persona: "Persona",
    policy: "Policy version",
    evidence_required: "Evidence required",
    quality_escalation: "Quality escalation",
    safety: "Safety risk",
    reason: "Reason",
    routingReason: "Routing",
  },
};

const ROUTING_POLICY_VALUE_LABELS: LabelTable = {
  tr: {
    "de-escalated": "Düşürülmüş model",
  },
  en: {
    "de-escalated": "De-escalated",
  },
};

const ROUTING_BOOLEAN_KEYS = new Set(["evidence_required", "quality_escalation"]);

const ROUTING_MODEL_VALUE_KEYS = new Set(["primary", "fallback", "selectedmodel", "frommodel", "persona"]);

export function roleLabel(role: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, ROLE_LABELS)[role] ?? role;
}

export function workspaceTypeLabel(type: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, WORKSPACE_TYPE_LABELS)[type] ?? type;
}

export function workspaceDisplayName(name: string, type: string, locale: Locale = "tr"): string {
  if (type === "PERSONAL" || /^personal workspace$/i.test(name.trim())) {
    return locale === "tr" ? "Kişisel çalışma alanı" : "Personal workspace";
  }
  return name;
}

export function activityLabel(action: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, ACTIVITY_LABELS)[action] ?? action.replaceAll("_", " ").toLowerCase();
}

export function complianceProfileLabel(profile: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, COMPLIANCE_PROFILE_LABELS)[profile] ?? profile;
}

export function policyStatusLabel(status: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, POLICY_STATUS_LABELS)[status] ?? status;
}

export function llmStatusLabel(status: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, LLM_STATUS_LABELS)[status] ?? status;
}

export function eventStatusLabel(status: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, EVENT_STATUS_LABELS)[status] ?? status;
}

export function toolLabel(toolName: string, locale: Locale = "tr"): string {
  return resolveLocaleMap(locale, TOOL_LABELS)[toolName] ?? toolName.replaceAll("_", " ");
}

export function modelKeyLabel(modelKey: string, locale: Locale = "tr"): string {
  const normalized = modelKey.trim().toLowerCase();
  return resolveLocaleMap(locale, MODEL_KEY_LABELS)[normalized] ?? modelKey.replaceAll("_", " ");
}

export function taskTypeLabel(taskType: string, locale: Locale = "tr"): string {
  const normalized = taskType.trim().toLowerCase();
  return resolveLocaleMap(locale, TASK_TYPE_LABELS)[normalized] ?? normalized.replaceAll("_", " ");
}

function routingValueLabel(key: string, value: string, locale: Locale): string {
  const normalizedKey = key.trim().toLowerCase();
  const trimmed = value.trim();
  const lowered = trimmed.toLowerCase();

  if (normalizedKey === "task" || normalizedKey === "tasktype") {
    return taskTypeLabel(trimmed, locale);
  }

  if (ROUTING_MODEL_VALUE_KEYS.has(normalizedKey)) {
    return modelKeyLabel(trimmed, locale);
  }

  if (normalizedKey === "policy") {
    return resolveLocaleMap(locale, ROUTING_POLICY_VALUE_LABELS)[lowered] ?? trimmed;
  }

  if (normalizedKey === "safety") {
    return analysisRiskLabel(trimmed.toUpperCase(), locale);
  }

  if (ROUTING_BOOLEAN_KEYS.has(normalizedKey)) {
    if (lowered === "true") return locale === "tr" ? "Evet" : "Yes";
    if (lowered === "false") return locale === "tr" ? "Hayır" : "No";
  }

  return trimmed;
}

function formatRoutingSegment(key: string, value: string, locale: Locale): string | null {
  const normalizedKey = key.trim().toLowerCase();
  const keyLabels = resolveLocaleMap(locale, ROUTING_KEY_LABELS);
  const label = keyLabels[normalizedKey] ?? keyLabels[key] ?? key;

  if (ROUTING_BOOLEAN_KEYS.has(normalizedKey)) {
    if (value.trim().toLowerCase() === "false") return null;
    if (value.trim().toLowerCase() === "true") return label;
  }

  return `${label}: ${routingValueLabel(key, value, locale)}`;
}

export function formatCostUsd(amount: number, locale: Locale = "tr"): string {
  return new Intl.NumberFormat(localeTag(locale), {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(amount);
}

export function formatNumber(value: number, locale: Locale = "tr"): string {
  return value.toLocaleString(localeTag(locale));
}

export function deviceDisplayLabel(
  platform: string,
  label: string,
): { primary: string; technical?: string } {
  const raw = (label.trim() || platform.trim()).trim();
  if (!raw) return { primary: "Bilinmeyen cihaz" };
  if (/^curl\//i.test(raw)) return { primary: "Komut satırı istemcisi", technical: raw };
  if (/^Web\s*[—-]\s*node/i.test(raw)) return { primary: "Web tarayıcısı", technical: raw };
  if (raw.length > 48 || /Mozilla\/|AppleWebKit\//i.test(raw)) {
    return { primary: "Web tarayıcısı", technical: raw };
  }
  return { primary: raw };
}

export function summarizeRoutingString(
  raw: string,
  locale: Locale = "tr",
): { summary: string; technical?: string } {
  const trimmed = raw.trim();
  if (!trimmed) return { summary: "" };

  if (trimmed.includes("=")) {
    const parts: Record<string, string> = {};
    for (const segment of trimmed.split(/[;,]/)) {
      const idx = segment.indexOf("=");
      if (idx <= 0) continue;
      parts[segment.slice(0, idx).trim()] = segment.slice(idx + 1).trim();
    }
    if (Object.keys(parts).length > 0) {
      const segments = Object.entries(parts)
        .map(([key, value]) => formatRoutingSegment(key, value, locale))
        .filter((segment): segment is string => Boolean(segment));
      return {
        summary: segments.join(" · "),
        technical: trimmed,
      };
    }
  }

  if (trimmed.length > 72) {
    return { summary: `${trimmed.slice(0, 68)}…`, technical: trimmed };
  }

  return { summary: trimmed };
}

function analysisFailureStepLabel(detail: string, locale: Locale): string {
  const lower = detail.toLowerCase();
  if (lower.includes("reviewer") || lower.includes("review")) {
    return locale === "tr" ? "İnceleme adımı" : "Review step";
  }
  return locale === "tr" ? "Model adımı" : "Model step";
}

export function humanizeAnalysisError(
  raw: string | null | undefined,
  locale: Locale = "tr",
): { message: string; technical?: string } {
  const detail = raw?.trim();
  if (!detail) {
    return { message: locale === "tr" ? "Analiz tamamlanamadı." : "Analysis did not complete." };
  }

  const lower = detail.toLowerCase();
  const step = analysisFailureStepLabel(detail, locale);

  if (lower.includes("blocked by compliance") || (lower.includes("403") && lower.includes("compliance"))) {
    return {
      message:
        locale === "tr"
          ? `Analiz tamamlanamadı. ${step} uyumluluk denetiminde engellendi.`
          : `Analysis did not complete. The ${step.toLowerCase()} was blocked during compliance checks.`,
      technical: detail,
    };
  }

  if (
    lower.includes("deadline exceeded") ||
    lower.includes("timed out") ||
    lower.includes("timeout exceeded") ||
    lower.includes("the read operation timed out")
  ) {
    return {
      message:
        locale === "tr"
          ? `Analiz tamamlanamadı. ${step} zaman aşımına uğradı; model sağlayıcısı geç yanıt verdi.`
          : `Analysis did not complete. The ${step.toLowerCase()} timed out waiting for the model provider.`,
      technical: detail,
    };
  }

  if (lower.includes("429") || lower.includes("rate limit") || lower.includes("rate limited")) {
    return {
      message:
        locale === "tr"
          ? `Analiz tamamlanamadı. ${step} model hız limitine takıldı; kısa süre sonra tekrar deneyin.`
          : `Analysis did not complete. The ${step.toLowerCase()} hit a model rate limit; retry shortly.`,
      technical: detail,
    };
  }

  if (
    lower.includes("500") ||
    lower.includes("502") ||
    lower.includes("503") ||
    lower.includes("engine_overloaded") ||
    lower.includes("provider overloaded")
  ) {
    return {
      message:
        locale === "tr"
          ? `Analiz tamamlanamadı. ${step} model sağlayıcısına ulaşılamadı.`
          : `Analysis did not complete. The ${step.toLowerCase()} could not reach the model provider.`,
      technical: detail,
    };
  }

  if (lower.includes("llm gateway failed") || lower.includes("gateway failed")) {
    return {
      message:
        locale === "tr"
          ? `Analiz tamamlanamadı. ${step} tamamlanamadı.`
          : `Analysis did not complete. The ${step.toLowerCase()} could not finish.`,
      technical: detail,
    };
  }

  if (lower.includes("orchestrator")) {
    return {
      message: locale === "tr" ? "Analiz servisi yanıt veremedi." : "The analysis service did not respond.",
      technical: detail,
    };
  }

  if (lower.includes("compliance")) {
    return {
      message:
        locale === "tr"
          ? "Analiz uyumluluk kuralları nedeniyle tamamlanamadı."
          : "Analysis could not complete due to compliance rules.",
      technical: detail,
    };
  }

  if (/^[A-Z0-9_]+$/.test(detail) && detail.length <= 40) {
    return { message: locale === "tr" ? "Analiz tamamlanamadı." : "Analysis did not complete.", technical: detail };
  }

  if (detail.length > 80 || /[=;]/.test(detail)) {
    return { message: locale === "tr" ? "Analiz tamamlanamadı." : "Analysis did not complete.", technical: detail };
  }

  return { message: detail };
}

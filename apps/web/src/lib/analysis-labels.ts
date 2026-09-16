import {
  DECISION_LABELS,
  FINDING_TYPE_EXPLANATIONS,
  FINDING_TYPE_LABELS,
  INSIGHT_KIND_LABELS,
  INSIGHT_TOPIC_LABELS,
  RISK_LABELS,
  STATUS_LABELS,
  SUPPORT_STATUS_LABELS,
} from "@/lib/analysis-label-data";
import { resolveLocaleMap } from "@/lib/i18n/helpers";
import type { Locale } from "@/lib/i18n/locale-config";
import { POLICY_PHRASES, phraseText } from "@/lib/policy-phrases";

function humanizeCode(value: string) {
  return value.replaceAll("_", " ").toLowerCase();
}

function labelFor(map: Record<string, string>, value: string) {
  return map[value] ?? humanizeCode(value);
}

export function statusLabel(status: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, STATUS_LABELS), status);
}

export function riskLabel(risk: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, RISK_LABELS), risk);
}

export function findingTypeLabel(type: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, FINDING_TYPE_LABELS), type);
}

export function isQualityFinding(type: string) {
  return type === "INSUFFICIENT_EVIDENCE";
}

export function findingRationale(type: string, rationale: string, locale: Locale = "tr") {
  const trimmed = rationale.trim();
  const typeLabels = resolveLocaleMap(locale, FINDING_TYPE_LABELS);
  if (!trimmed || trimmed === type || typeLabels[trimmed]) {
    return resolveLocaleMap(locale, FINDING_TYPE_EXPLANATIONS)[type] ?? "";
  }
  return localizePolicyText(trimmed, locale);
}

export function supportStatusLabel(status: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, SUPPORT_STATUS_LABELS), status);
}

export function insightKindLabel(kind: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, INSIGHT_KIND_LABELS), kind);
}

export function insightTopicLabel(topic: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, INSIGHT_TOPIC_LABELS), topic);
}

function normalizePhrase(text: string) {
  return text
    .trim()
    .toLowerCase()
    .replace(/[\u2018\u2019]/g, "'")
    .replace(/[\u201c\u201d]/g, '"')
    .replace(/[.,;:!?]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function localizePolicyText(text: string, locale: Locale = "tr") {
  const trimmed = text.trim();
  if (!trimmed) return trimmed;
  const normalized = normalizePhrase(trimmed);

  for (const phrase of POLICY_PHRASES) {
    const target = phraseText(phrase, locale);
    const variants = Object.values(phrase).filter(Boolean) as string[];
    if (variants.some((variant) => normalizePhrase(variant) === normalized)) {
      return target;
    }
  }

  let out = trimmed;
  for (const phrase of POLICY_PHRASES) {
    const target = phraseText(phrase, locale);
    const variants = Object.values(phrase).filter(Boolean) as string[];
    for (const variant of variants) {
      out = out.replaceAll(variant, target);
    }
  }
  return out;
}

export function decisionLabel(decision: string, locale: Locale = "tr") {
  return labelFor(resolveLocaleMap(locale, DECISION_LABELS), decision);
}

import type { Locale } from "./locale-config";
import { ar, de, es, fr, ja, pt, ru, zh } from "./messages/additional";
import { en } from "./messages/en";
import { tr, type MessageCatalog, type MessageKey } from "./messages/keys";

export type { MessageKey };

export const messages: Record<Locale, MessageCatalog> = {
  tr,
  en,
  de,
  fr,
  es,
  ar,
  zh,
  ja,
  ru,
  pt,
};

export function t(key: MessageKey, locale: Locale, vars?: Record<string, string | number>): string {
  let text = messages[locale]?.[key] ?? messages.en[key] ?? messages.tr[key] ?? key;
  if (vars) {
    for (const [name, value] of Object.entries(vars)) {
      text = text.replaceAll(`{${name}}`, String(value));
    }
  }
  return text;
}

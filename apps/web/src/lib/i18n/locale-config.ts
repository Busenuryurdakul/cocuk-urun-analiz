export const LOCALE_DEFINITIONS = [
  { code: "tr", api: "TR", label: "Turkish", native: "Türkçe", tag: "tr-TR", rtl: false },
  { code: "en", api: "EN", label: "English", native: "English", tag: "en-US", rtl: false },
  { code: "de", api: "DE", label: "German", native: "Deutsch", tag: "de-DE", rtl: false },
  { code: "fr", api: "FR", label: "French", native: "Français", tag: "fr-FR", rtl: false },
  { code: "es", api: "ES", label: "Spanish", native: "Español", tag: "es-ES", rtl: false },
  { code: "ar", api: "AR", label: "Arabic", native: "العربية", tag: "ar-SA", rtl: true },
  { code: "zh", api: "ZH", label: "Chinese", native: "中文", tag: "zh-CN", rtl: false },
  { code: "ja", api: "JA", label: "Japanese", native: "日本語", tag: "ja-JP", rtl: false },
  { code: "ru", api: "RU", label: "Russian", native: "Русский", tag: "ru-RU", rtl: false },
  { code: "pt", api: "PT", label: "Portuguese", native: "Português", tag: "pt-BR", rtl: false },
] as const;

export type Locale = (typeof LOCALE_DEFINITIONS)[number]["code"];
export type ApiLocale = (typeof LOCALE_DEFINITIONS)[number]["api"];

export const DEFAULT_LOCALE: Locale = "tr";
export const LOCALES = LOCALE_DEFINITIONS.map((item) => item.code) as Locale[];

const byCode = new Map(LOCALE_DEFINITIONS.map((item) => [item.code, item]));
const byApi = new Map(LOCALE_DEFINITIONS.map((item) => [item.api, item]));

export function localeDefinition(code: Locale) {
  return byCode.get(code)!;
}

export function localeFromApiCode(value?: string | null): Locale {
  if (!value) return DEFAULT_LOCALE;
  const match = byApi.get(value.trim().toUpperCase() as ApiLocale);
  return match?.code ?? DEFAULT_LOCALE;
}

export function localeToApiCode(locale: Locale): ApiLocale {
  return localeDefinition(locale).api;
}

export function normalizeLocale(value?: string | null): Locale {
  if (!value) return DEFAULT_LOCALE;
  const trimmed = value.trim();
  const lower = trimmed.toLowerCase();
  const direct = byCode.get(lower as Locale);
  if (direct) return direct.code;
  const apiMatch = byApi.get(trimmed.toUpperCase() as ApiLocale);
  if (apiMatch) return apiMatch.code;
  if (lower.startsWith("en")) return "en";
  if (lower.startsWith("de")) return "de";
  if (lower.startsWith("fr")) return "fr";
  if (lower.startsWith("es")) return "es";
  if (lower.startsWith("ar")) return "ar";
  if (lower.startsWith("zh")) return "zh";
  if (lower.startsWith("ja")) return "ja";
  if (lower.startsWith("ru")) return "ru";
  if (lower.startsWith("pt")) return "pt";
  return DEFAULT_LOCALE;
}

export function localeTag(locale: Locale): string {
  return localeDefinition(locale).tag;
}

export function localeIsRtl(locale: Locale): boolean {
  return localeDefinition(locale).rtl;
}

import {
  DEFAULT_LOCALE,
  LOCALE_DEFINITIONS,
  LOCALES,
  localeDefinition,
  localeFromApiCode,
  localeIsRtl,
  localeTag,
  localeToApiCode,
  normalizeLocale,
  type ApiLocale,
  type Locale,
} from "./locale-config";

export {
  DEFAULT_LOCALE,
  LOCALE_DEFINITIONS,
  LOCALES,
  localeDefinition,
  localeIsRtl,
  localeTag,
  type ApiLocale,
  type Locale,
};

export const localeFromApi = localeFromApiCode;
export const localeToApi = localeToApiCode;

export const LOCALE_STORAGE_KEY = "miyuna_locale";

export function getStoredLocale(): Locale {
  if (typeof window === "undefined") return DEFAULT_LOCALE;
  return normalizeLocale(window.localStorage.getItem(LOCALE_STORAGE_KEY));
}

export function setStoredLocale(locale: Locale): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(LOCALE_STORAGE_KEY, locale);
}

export function formatDateTime(value: string | Date, locale: Locale): string {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return date.toLocaleString(localeTag(locale), {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

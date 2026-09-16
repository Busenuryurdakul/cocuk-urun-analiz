import type { Locale } from "./locale-config";

export function resolveLocaleText<T extends string>(
  locale: Locale,
  table: Partial<Record<Locale, T>> & { en: T; tr: T },
): T {
  return table[locale] ?? table.en ?? table.tr;
}

export function resolveLocaleMap<T extends Record<string, string>>(
  locale: Locale,
  table: Partial<Record<Locale, T>> & { en: T; tr: T },
): T {
  return table[locale] ?? table.en ?? table.tr;
}

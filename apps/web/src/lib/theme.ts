export const THEME_STORAGE_KEY = "miyuna_theme";

export const COLOR_SCHEMES = ["light", "dark", "system"] as const;

export type ColorScheme = (typeof COLOR_SCHEMES)[number];
export type ResolvedColorScheme = "light" | "dark";
export type ApiColorScheme = "LIGHT" | "DARK" | "SYSTEM";

export const DEFAULT_COLOR_SCHEME: ColorScheme = "light";

const byValue = new Set<string>(COLOR_SCHEMES);

export function normalizeColorScheme(value?: string | null): ColorScheme {
  if (!value) return DEFAULT_COLOR_SCHEME;
  const lower = value.trim().toLowerCase();
  if (byValue.has(lower)) return lower as ColorScheme;
  return colorSchemeFromApi(value);
}

export function colorSchemeFromApi(value?: string | null): ColorScheme {
  switch (value?.trim().toUpperCase()) {
    case "DARK":
      return "dark";
    case "SYSTEM":
      return "system";
    default:
      return DEFAULT_COLOR_SCHEME;
  }
}

export function colorSchemeToApi(scheme: ColorScheme): ApiColorScheme {
  switch (scheme) {
    case "dark":
      return "DARK";
    case "system":
      return "SYSTEM";
    default:
      return "LIGHT";
  }
}

export function resolveColorScheme(scheme: ColorScheme): ResolvedColorScheme {
  if (scheme !== "system") return scheme;
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function applyColorScheme(scheme: ColorScheme): ResolvedColorScheme {
  const resolved = resolveColorScheme(scheme);
  if (typeof document === "undefined") return resolved;
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
  return resolved;
}

export function getStoredColorScheme(): ColorScheme {
  if (typeof window === "undefined") return DEFAULT_COLOR_SCHEME;
  return normalizeColorScheme(window.localStorage.getItem(THEME_STORAGE_KEY));
}

export function setStoredColorScheme(scheme: ColorScheme): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(THEME_STORAGE_KEY, scheme);
}

export const THEME_INIT_SCRIPT = `(function(){try{var k=${JSON.stringify(THEME_STORAGE_KEY)};var t=localStorage.getItem(k);if(t!=="light"&&t!=="dark"&&t!=="system")t="light";var r=t==="system"?(window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"):t;var e=document.documentElement;e.setAttribute("data-theme",r);e.style.colorScheme=r;}catch(err){}})();`;

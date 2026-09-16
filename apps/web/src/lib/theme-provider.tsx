"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { graphqlRequest } from "@/lib/graphql";
import {
  applyColorScheme,
  colorSchemeFromApi,
  colorSchemeToApi,
  getStoredColorScheme,
  setStoredColorScheme,
  type ColorScheme,
  type ResolvedColorScheme,
} from "@/lib/theme";

type ThemeOptions = { persist?: boolean; store?: boolean };

type ThemeContextValue = {
  colorScheme: ColorScheme;
  resolvedScheme: ResolvedColorScheme;
  setColorScheme: (scheme: ColorScheme, options?: ThemeOptions) => Promise<boolean>;
  ready: boolean;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [colorScheme, setColorSchemeState] = useState<ColorScheme>("light");
  const [resolvedScheme, setResolvedScheme] = useState<ResolvedColorScheme>("light");
  const [ready, setReady] = useState(false);
  const userTouched = useRef(false);

  const sync = useCallback((next: ColorScheme, options?: { store?: boolean }) => {
    setColorSchemeState(next);
    if (options?.store !== false) {
      setStoredColorScheme(next);
    }
    setResolvedScheme(applyColorScheme(next));
  }, []);

  useEffect(() => {
    const stored = getStoredColorScheme();
    sync(stored);
    setReady(true);

    graphqlRequest<{ me: { preferredColorScheme: string } | null }>(`{ me { preferredColorScheme } }`)
      .then((data) => {
        if (userTouched.current || !data.me?.preferredColorScheme) return;
        sync(colorSchemeFromApi(data.me.preferredColorScheme));
      })
      .catch(() => undefined);
  }, [sync]);

  useEffect(() => {
    if (colorScheme !== "system") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setResolvedScheme(applyColorScheme("system"));
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [colorScheme]);

  const setColorScheme = useCallback(
    async (next: ColorScheme, options?: ThemeOptions) => {
      userTouched.current = true;
      sync(next, { store: options?.store });
      if (options?.persist === false) return true;

      try {
        await graphqlRequest<{ updateUserPreferences: { preferredColorScheme: string } }>(
          `mutation($input: UpdateUserPreferencesInput!) {
            updateUserPreferences(input: $input) { preferredColorScheme }
          }`,
          { input: { preferredColorScheme: colorSchemeToApi(next) } },
        );
        return true;
      } catch {
        return false;
      }
    },
    [sync],
  );

  const value = useMemo<ThemeContextValue>(
    () => ({ colorScheme, resolvedScheme, setColorScheme, ready }),
    [colorScheme, ready, resolvedScheme, setColorScheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return ctx;
}

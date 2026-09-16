"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { graphqlRequest } from "@/lib/graphql";
import { authErrorMessage } from "./auth-errors";
import {
  DEFAULT_LOCALE,
  getStoredLocale,
  localeFromApi,
  localeIsRtl,
  localeToApi,
  setStoredLocale,
  type Locale,
} from "./locale";
import { t as translate, type MessageKey } from "./messages";

type LocaleContextValue = {
  locale: Locale;
  setLocale: (locale: Locale, options?: { persist?: boolean }) => Promise<boolean>;
  t: (key: MessageKey, vars?: Record<string, string | number>) => string;
  authMessage: (code: string, fallback?: string) => string;
  ready: boolean;
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const stored = getStoredLocale();
    setLocaleState(stored);
    document.documentElement.lang = stored;
    document.documentElement.dir = localeIsRtl(stored) ? "rtl" : "ltr";
    setReady(true);

    graphqlRequest<{ me: { preferredLocale: string } | null }>(`{ me { preferredLocale } }`)
      .then((data) => {
        if (!data.me?.preferredLocale) return;
        const serverLocale = localeFromApi(data.me.preferredLocale);
        setLocaleState(serverLocale);
        setStoredLocale(serverLocale);
        document.documentElement.lang = serverLocale;
        document.documentElement.dir = localeIsRtl(serverLocale) ? "rtl" : "ltr";
      })
      .catch(() => undefined);
  }, []);

  const setLocale = useCallback(async (next: Locale, options?: { persist?: boolean }) => {
    setLocaleState(next);
    setStoredLocale(next);
    document.documentElement.lang = next;
    document.documentElement.dir = localeIsRtl(next) ? "rtl" : "ltr";

    if (options?.persist === false) return true;

    try {
      await graphqlRequest<{ updateUserPreferences: { preferredLocale: string } }>(
        `mutation($input: UpdateUserPreferencesInput!) {
          updateUserPreferences(input: $input) { preferredLocale }
        }`,
        { input: { preferredLocale: localeToApi(next) } },
      );
      return true;
    } catch {
      // Guest or offline — local preference still applies.
      return false;
    }
  }, []);

  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      setLocale,
      ready,
      t: (key, vars) => translate(key, locale, vars),
      authMessage: (code, fallback) =>
        authErrorMessage(code, locale) ?? fallback ?? translate("auth.fallback", locale),
    }),
    [locale, ready, setLocale],
  );

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  const ctx = useContext(LocaleContext);
  if (!ctx) {
    throw new Error("useLocale must be used within LocaleProvider");
  }
  return ctx;
}

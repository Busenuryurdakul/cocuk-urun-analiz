"use client";

import { useEffect, useState } from "react";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlRequest } from "@/lib/graphql";
import { LOCALE_DEFINITIONS, localeFromApi, localeToApi, type Locale } from "@/lib/i18n/locale";
import { t as translateMessage } from "@/lib/i18n/messages";
import { useLocale } from "@/lib/i18n/locale-provider";
import { COLOR_SCHEMES, colorSchemeFromApi, colorSchemeToApi, type ColorScheme } from "@/lib/theme";
import { useTheme } from "@/lib/theme-provider";

export default function PreferencesPage() {
  const { locale, setLocale, t } = useLocale();
  const { colorScheme, setColorScheme } = useTheme();
  const [email, setEmail] = useState("");
  const [selected, setSelected] = useState<Locale>(locale);
  const [selectedTheme, setSelectedTheme] = useState<ColorScheme>(colorScheme);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await graphqlRequest<{
          me: { email: string; preferredLocale: string; preferredColorScheme: string };
        }>(`{ me { email preferredLocale preferredColorScheme } }`);
        if (cancelled) return;
        setEmail(data.me.email);
        const serverLocale = localeFromApi(data.me.preferredLocale);
        const serverTheme = colorSchemeFromApi(data.me.preferredColorScheme);
        setSelected(serverLocale);
        setSelectedTheme(serverTheme);
        void setLocale(serverLocale, { persist: false });
        void setColorScheme(serverTheme, { persist: false });
      } catch {
        // Guest, offline, or older API — local language/theme still apply.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [setColorScheme, setLocale]);

  async function save() {
    setSaving(true);
    setMessage("");
    setError("");
    try {
      await graphqlRequest<{
        updateUserPreferences: { preferredLocale: string; preferredColorScheme: string };
      }>(
        `mutation($input: UpdateUserPreferencesInput!) {
          updateUserPreferences(input: $input) { preferredLocale preferredColorScheme }
        }`,
        {
          input: {
            preferredLocale: localeToApi(selected),
            preferredColorScheme: colorSchemeToApi(selectedTheme),
          },
        },
      );
      await setLocale(selected, { persist: false });
      await setColorScheme(selectedTheme, { persist: false });
      setMessage(translateMessage("preferences.saved", selected));
    } catch {
      await setLocale(selected, { persist: false });
      await setColorScheme(selectedTheme, { persist: false });
      setError(translateMessage("preferences.saveFailed", selected));
    } finally {
      setSaving(false);
    }
  }

  function chooseTheme(next: ColorScheme) {
    setSelectedTheme(next);
    void setColorScheme(next, { persist: false, store: false });
  }

  const themeLabels: Record<ColorScheme, string> = {
    light: t("preferences.themeLight"),
    dark: t("preferences.themeDark"),
    system: t("preferences.themeSystem"),
  };

  return (
    <AppShell title={t("preferences.title")} description={t("preferences.description")} accountEmail={email}>
      <section className="card max-w-lg space-y-6">
        <div className="space-y-2">
          <label htmlFor="locale" className="text-sm font-semibold">
            {t("preferences.language")}
          </label>
          <select
            id="locale"
            className="input w-full"
            value={selected}
            onChange={(event) => setSelected(event.target.value as Locale)}
          >
            {LOCALE_DEFINITIONS.map((item) => (
              <option key={item.code} value={item.code}>
                {item.native} ({item.label})
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-3">
          <p className="text-sm font-semibold">{t("preferences.theme")}</p>
          <div className="grid grid-cols-3 gap-2">
            {COLOR_SCHEMES.map((value) => {
              const selectedCard = selectedTheme === value;
              return (
                <button
                  key={value}
                  type="button"
                  aria-pressed={selectedCard}
                  onClick={() => chooseTheme(value)}
                  className={`rounded-2xl border px-3 py-3 text-start transition ${
                    selectedCard
                      ? "border-forest bg-forest-soft shadow-sm ring-2 ring-forest/25"
                      : "border-sand bg-paper hover:border-forest/40"
                  }`}
                >
                  <span
                    className={`mb-2 flex h-10 overflow-hidden rounded-lg border ${
                      value === "dark" ? "border-zinc-700" : "border-sand"
                    }`}
                    aria-hidden
                  >
                    {value === "system" ? (
                      <>
                        <span className="w-1/2 bg-[#f8f5f0]" />
                        <span className="w-1/2 bg-[#1c1b18]" />
                      </>
                    ) : (
                      <span className={`w-full ${value === "dark" ? "bg-[#1c1b18]" : "bg-[#f8f5f0]"}`} />
                    )}
                  </span>
                  <span className="block text-sm font-semibold text-ink">{themeLabels[value]}</span>
                </button>
              );
            })}
          </div>
          <p className="text-xs leading-relaxed text-muted">{t("preferences.themeHint")}</p>
        </div>
        {message && <p className="text-sm text-forest">{message}</p>}
        {error && <p className="text-sm text-clay">{error}</p>}
        <button type="button" className="btn-primary" disabled={saving} onClick={() => void save()}>
          {saving ? t("preferences.saving") : t("preferences.save")}
        </button>
      </section>
    </AppShell>
  );
}

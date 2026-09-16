"use client";

import { LocaleProvider } from "@/lib/i18n/locale-provider";
import { ThemeProvider } from "@/lib/theme-provider";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <LocaleProvider>{children}</LocaleProvider>
    </ThemeProvider>
  );
}
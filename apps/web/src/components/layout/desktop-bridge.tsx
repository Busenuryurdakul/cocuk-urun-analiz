"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

function routeFromDeepLink(url: string): string | null {
  try {
    const parsed = new URL(url);
    const token = parsed.searchParams.get("token");
    const host = `${parsed.hostname}${parsed.pathname}`.replace(/\/+$/, "");
    if (host.includes("verify-email") && token) {
      return `/auth/verify-email?token=${encodeURIComponent(token)}`;
    }
    if (host.includes("login")) return "/auth/login";
    if (host.includes("workspace")) return "/workspace";
    return null;
  } catch {
    return null;
  }
}

export function DesktopBridge() {
  const router = useRouter();

  useEffect(() => {
    const api = window.miyunaDesktop;
    if (!api) return;
    document.documentElement.dataset.platform = "electron";
    if (!api.onDeepLink) return;
    const unsubscribe = api.onDeepLink((url: string) => {
      const next = routeFromDeepLink(url);
      if (next) router.push(next);
    });
    return () => {
      if (typeof unsubscribe === "function") unsubscribe();
    };
  }, [router]);

  return null;
}

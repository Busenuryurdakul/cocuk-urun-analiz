"use client";

import Link from "next/link";
import { ReactNode, useState } from "react";
import { buildDefaultNavItems, Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";
import { clearDesktopSession, graphqlRequest } from "@/lib/graphql";
import { useLocale } from "@/lib/i18n/locale-provider";
import { cn } from "@/lib/cn";

type AppShellProps = {
  title: string;
  kicker?: string;
  description?: string;
  orgId?: string;
  organizationLabel?: string;
  accountEmail?: string;
  actions?: ReactNode;
  restricted?: boolean;
  children: ReactNode;
};

export function AppShell({
  title,
  kicker,
  description,
  orgId,
  organizationLabel,
  accountEmail,
  actions,
  restricted = false,
  children,
}: AppShellProps) {
  const { t } = useLocale();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  async function logout() {
    await graphqlRequest(`mutation { logout }`).catch(() => undefined);
    await clearDesktopSession();
    window.location.href = "/auth/login";
  }

  const navItems = buildDefaultNavItems(t, orgId);

  return (
    <div className="flex min-h-screen">
      <div className="hidden lg:block">
        <div className="sticky top-0 h-screen">
          <Sidebar
            items={navItems}
            accountEmail={accountEmail}
            onLogout={() => void logout()}
            logoutLabel={t("nav.logout")}
            restricted={restricted}
            loginHref="/auth/login"
            loginLabel={t("nav.login")}
          />
        </div>
      </div>

      {mobileNavOpen && (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-ink/20 backdrop-blur-sm lg:hidden"
          aria-label="Close navigation menu"
          onClick={() => setMobileNavOpen(false)}
        />
      )}

      <div
        className={cn(
          "fixed inset-y-0 left-0 z-50 transform transition-transform duration-normal lg:hidden motion-reduce:transition-none",
          mobileNavOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <Sidebar
          items={navItems}
          accountEmail={accountEmail}
          onLogout={() => void logout()}
          logoutLabel={t("nav.logout")}
          restricted={restricted}
          loginHref="/auth/login"
          loginLabel={t("nav.login")}
          className="h-full shadow-lift"
        />
      </div>

      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar
          showMenuButton
          onMenuClick={() => setMobileNavOpen((v) => !v)}
          organizationLabel={organizationLabel}
          accountEmail={accountEmail}
        />

        <main className="flex-1">
          <div className="page-container py-6 sm:py-8">
            <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
              <div className="max-w-2xl space-y-2">
                {kicker && <p className="kicker">{kicker}</p>}
                <h1 className="page-title">{title}</h1>
                {description && (
                  <p className="text-sm leading-relaxed text-text-secondary">{description}</p>
                )}
              </div>
              {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
            </header>

            {children}

            <div className="mt-8 flex items-center justify-between border-t border-border/60 pt-4 lg:hidden">
              {accountEmail && !restricted && <p className="text-xs text-muted">{accountEmail}</p>}
              {!restricted ? (
                <button type="button" onClick={() => void logout()} className="link-quiet">
                  {t("nav.logout")}
                </button>
              ) : (
                <Link href="/auth/login" className="link-quiet">
                  {t("nav.login")}
                </Link>
              )}
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

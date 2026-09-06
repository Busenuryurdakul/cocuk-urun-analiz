"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ReactNode } from "react";
import { Logo } from "@/components/brand/logo";
import { clearDesktopSession, graphqlRequest } from "@/lib/graphql";

type AppShellProps = {
  title: string;
  kicker?: string;
  description?: string;
  orgId?: string;
  accountEmail?: string;
  actions?: ReactNode;
  children: ReactNode;
};

export function AppShell({
  title,
  kicker,
  description,
  orgId,
  accountEmail,
  actions,
  children,
}: AppShellProps) {
  const pathname = usePathname();

  async function logout() {
    await graphqlRequest(`mutation { logout }`).catch(() => undefined);
    await clearDesktopSession();
    window.location.href = "/auth/login";
  }

  const items = [
    { href: "/workspace", label: "Çalışma alanı", match: (p: string) => p === "/workspace" },
    { href: "/settings/security", label: "Güvenlik", match: (p: string) => p.startsWith("/settings") },
    ...(orgId
      ? [
          {
            href: `/org/${orgId}/products`,
            label: "Ürünler",
            match: (p: string) => p.includes("/products"),
          },
          {
            href: `/org/${orgId}/members`,
            label: "Üyeler",
            match: (p: string) => p.includes("/members"),
          },
          {
            href: `/org/${orgId}/compliance`,
            label: "Uyumluluk",
            match: (p: string) => p.includes("/compliance"),
          },
          {
            href: `/org/${orgId}/llm`,
            label: "LLM",
            match: (p: string) => p.includes("/llm"),
          },
        ]
      : []),
  ];

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[260px_1fr]">
      <aside className="bg-forest-deep px-5 py-6 text-paper lg:sticky lg:top-0 lg:flex lg:h-screen lg:flex-col">
        <Logo href="/workspace" tone="paper" size="sm" />
        <nav className="mt-8 space-y-1">
          {items.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={item.match(pathname) ? "nav-item-active" : "nav-item"}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-current opacity-70" />
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="mt-auto hidden pt-8 lg:block">
          {accountEmail && <p className="truncate px-3 text-xs text-paper/50">{accountEmail}</p>}
          <button type="button" onClick={() => void logout()} className="nav-item mt-2 w-full text-left">
            Çıkış
          </button>
        </div>
      </aside>

      <div className="px-5 py-8 sm:px-8 lg:px-12">
        <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
          <div className="max-w-2xl space-y-2">
            {kicker && <p className="kicker">{kicker}</p>}
            <h1 className="display text-3xl sm:text-4xl">{title}</h1>
            {description && <p className="text-sm leading-relaxed text-muted">{description}</p>}
          </div>
          {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
        </header>
        {children}
        <div className="mt-8 flex items-center justify-between lg:hidden">
          {accountEmail && <p className="text-xs text-muted">{accountEmail}</p>}
          <button type="button" onClick={() => void logout()} className="link-quiet">
            Çıkış
          </button>
        </div>
      </div>
    </div>
  );
}

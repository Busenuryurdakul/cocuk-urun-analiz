"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ReactNode } from "react";
import { Logo } from "@/components/brand/logo";
import { cn } from "@/lib/cn";
import {
  BrainIcon,
  ComplianceIcon,
  GridIcon,
  LogOutIcon,
  PackageIcon,
  SettingsIcon,
  UsersIcon,
} from "@/components/ui/icons";
import type { MessageKey } from "@/lib/i18n/messages/keys";

export type SidebarNavItem = {
  href: string;
  label: string;
  icon: ReactNode;
  match: (pathname: string) => boolean;
};

type SidebarProps = {
  items: SidebarNavItem[];
  accountEmail?: string;
  onLogout?: () => void;
  logoutLabel?: string;
  restricted?: boolean;
  loginHref?: string;
  loginLabel?: string;
  className?: string;
};

export function Sidebar({
  items,
  accountEmail,
  onLogout,
  logoutLabel = "Log out",
  restricted = false,
  loginHref = "/auth/login",
  loginLabel = "Log in",
  className,
}: SidebarProps) {
  const pathname = usePathname();

  return (
    <aside
      className={cn(
        "flex h-full w-[240px] shrink-0 flex-col border-r border-border/60 bg-base px-3 py-5",
        className,
      )}
    >
      <Logo href="/workspace" tone="forest" size="sm" />

      <nav className="mt-8 flex-1 space-y-0.5" aria-label="Main navigation">
        {!restricted &&
          items.map((item) => {
            const active = item.match(pathname);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={active ? "nav-item-active" : "nav-item"}
                aria-current={active ? "page" : undefined}
              >
                <span className={cn("shrink-0", active ? "text-primary" : "text-muted")}>
                  {item.icon}
                </span>
                <span className="truncate">{item.label}</span>
              </Link>
            );
          })}
        {restricted && (
          <Link href={loginHref} className="nav-item">
            {loginLabel}
          </Link>
        )}
      </nav>

      <div className="mt-auto space-y-1 border-t border-border/60 pt-4">
        {accountEmail && !restricted && (
          <p className="truncate px-3 py-1 text-[11px] text-muted">{accountEmail}</p>
        )}
        {!restricted && onLogout && (
          <button type="button" onClick={onLogout} className="nav-item w-full text-left">
            <LogOutIcon size={16} className="shrink-0 text-muted" />
            {logoutLabel}
          </button>
        )}
      </div>
    </aside>
  );
}

export function buildDefaultNavItems(
  t: (key: MessageKey) => string,
  orgId?: string,
): SidebarNavItem[] {
  return [
    {
      href: "/workspace",
      label: t("nav.workspace"),
      icon: <GridIcon size={16} />,
      match: (p) => p === "/workspace",
    },
    {
      href: "/settings/preferences",
      label: t("nav.preferences"),
      icon: <SettingsIcon size={16} />,
      match: (p) => p.startsWith("/settings/preferences"),
    },
    {
      href: "/settings/security",
      label: t("nav.security"),
      icon: <SettingsIcon size={16} />,
      match: (p) => p.startsWith("/settings/security"),
    },
    ...(orgId
      ? [
          {
            href: `/org/${orgId}/products`,
            label: t("nav.products"),
            icon: <PackageIcon size={16} />,
            match: (p: string) => p.includes("/products"),
          },
          {
            href: `/org/${orgId}/members`,
            label: t("nav.members"),
            icon: <UsersIcon size={16} />,
            match: (p: string) => p.includes("/members"),
          },
          {
            href: `/org/${orgId}/compliance`,
            label: t("nav.compliance"),
            icon: <ComplianceIcon size={16} />,
            match: (p: string) => p.includes("/compliance"),
          },
          {
            href: `/org/${orgId}/llm`,
            label: t("nav.llm"),
            icon: <BrainIcon size={16} />,
            match: (p: string) => p.includes("/llm"),
          },
        ]
      : []),
  ];
}

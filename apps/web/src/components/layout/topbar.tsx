"use client";

import { ReactNode, useState } from "react";
import { cn } from "@/lib/cn";
import { BellIcon, MenuIcon, SearchIcon, UserIcon } from "@/components/ui/icons";

type TopbarProps = {
  onMenuClick?: () => void;
  showMenuButton?: boolean;
  searchPlaceholder?: string;
  organizationLabel?: string;
  accountEmail?: string;
  notificationCount?: number;
  actions?: ReactNode;
  className?: string;
};

export function Topbar({
  onMenuClick,
  showMenuButton = false,
  searchPlaceholder = "Search products, recalls…",
  organizationLabel,
  accountEmail,
  notificationCount = 0,
  actions,
  className,
}: TopbarProps) {
  const [searchFocused, setSearchFocused] = useState(false);

  return (
    <header
      className={cn(
        "sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-border/60 bg-base/80 px-4 backdrop-blur-md sm:gap-4 sm:px-6",
        className,
      )}
    >
      {showMenuButton && (
        <button
          type="button"
          onClick={onMenuClick}
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-text-secondary transition hover:bg-secondary lg:hidden"
          aria-label="Open navigation menu"
        >
          <MenuIcon size={18} />
        </button>
      )}

      <div
        className={cn(
          "relative hidden min-w-0 flex-1 sm:block sm:max-w-sm",
          searchFocused && "sm:max-w-md",
        )}
      >
        <SearchIcon
          size={15}
          className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-muted"
        />
        <input
          type="search"
          placeholder={searchPlaceholder}
          className="w-full rounded-xl border border-border bg-surface py-2 pl-10 pr-4 text-sm text-ink outline-none transition duration-normal placeholder:text-muted focus:border-primary/30 focus:shadow-focus"
          onFocus={() => setSearchFocused(true)}
          onBlur={() => setSearchFocused(false)}
          aria-label={searchPlaceholder}
        />
      </div>

      <div className="ml-auto flex items-center gap-2 sm:gap-3">
        {organizationLabel && (
          <span className="hidden rounded-full border border-border bg-surface px-3 py-1.5 text-xs font-medium text-text-secondary md:inline-flex">
            {organizationLabel}
          </span>
        )}

        {actions}

        <button
          type="button"
          className="relative flex h-9 w-9 items-center justify-center rounded-xl text-text-secondary transition hover:bg-secondary"
          aria-label="Notifications"
        >
          <BellIcon size={17} />
          {notificationCount > 0 && (
            <span className="absolute right-1.5 top-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[9px] font-semibold text-on-brand">
              {notificationCount > 9 ? "9+" : notificationCount}
            </span>
          )}
        </button>

        <div
          className="flex h-9 items-center gap-2 rounded-xl border border-border bg-surface px-2.5"
          title={accountEmail}
        >
          <span className="flex h-6 w-6 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <UserIcon size={14} />
          </span>
          {accountEmail && (
            <span className="hidden max-w-[120px] truncate text-xs font-medium text-ink lg:inline">
              {accountEmail.split("@")[0]}
            </span>
          )}
        </div>
      </div>
    </header>
  );
}

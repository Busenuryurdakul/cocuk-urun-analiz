"use client";

import Link from "next/link";
import { ReactNode } from "react";
import { useLocale } from "@/lib/i18n/locale-provider";

type AsyncViewState = "loading" | "error" | "empty" | "unauthorized" | "retry" | "success";

type AsyncViewProps = {
  state: AsyncViewState;
  loading?: ReactNode;
  error?: ReactNode;
  empty?: ReactNode;
  unauthorized?: ReactNode;
  retry?: ReactNode;
  children?: ReactNode;
};

/** Phase 1 UI shell — FINAL_MASTER_PROMPT.md §3 state coverage */
export function AsyncView({
  state,
  loading,
  error,
  empty,
  unauthorized,
  retry,
  children,
}: AsyncViewProps) {
  switch (state) {
    case "loading":
      return <>{loading ?? <DefaultLoading />}</>;
    case "error":
      return <>{error ?? <DefaultError />}</>;
    case "empty":
      return <>{empty ?? <DefaultEmpty />}</>;
    case "unauthorized":
      return <>{unauthorized ?? <DefaultUnauthorized />}</>;
    case "retry":
      return <>{retry ?? <DefaultRetry />}</>;
    default:
      return <>{children}</>;
  }
}

function DefaultLoading() {
  const { t } = useLocale();
  return (
    <div role="status" className="card text-center">
      <div className="mx-auto mb-3 h-8 w-8 animate-pulse rounded-full bg-forest-soft" />
      <p className="text-sm text-muted">{t("async.loading")}</p>
    </div>
  );
}

function DefaultError() {
  const { t } = useLocale();
  return (
    <div role="alert" className="alert-error text-center">
      <p>{t("async.error")}</p>
    </div>
  );
}

function DefaultEmpty() {
  const { t } = useLocale();
  return (
    <div className="card text-center">
      <p className="font-display text-xl text-ink">{t("async.empty.title")}</p>
      <p className="mt-1 text-sm text-muted">{t("async.empty.desc")}</p>
    </div>
  );
}

function DefaultUnauthorized() {
  const { t } = useLocale();
  return (
    <div className="alert-warn space-y-3 text-center">
      <p>{t("async.unauthorized")}</p>
      <Link href="/auth/login" className="btn-primary inline-flex">
        {t("nav.login")}
      </Link>
    </div>
  );
}

function DefaultRetry() {
  const { t } = useLocale();
  return (
    <div className="card text-center">
      <p className="mb-4 text-sm text-muted">{t("async.retry")}</p>
      <button type="button" className="btn-primary">
        {t("async.retryButton")}
      </button>
    </div>
  );
}

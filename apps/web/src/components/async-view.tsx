import { ReactNode } from "react";

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
  return (
    <div role="status" className="card text-center">
      <div className="mx-auto mb-3 h-8 w-8 animate-pulse rounded-full bg-forest-soft" />
      <p className="text-sm text-muted">Yükleniyor…</p>
    </div>
  );
}

function DefaultError() {
  return (
    <div role="alert" className="alert-error text-center">
      <p>Bir hata oluştu.</p>
    </div>
  );
}

function DefaultEmpty() {
  return (
    <div className="card text-center">
      <p className="font-display text-xl text-ink">Henüz bir şey yok</p>
      <p className="mt-1 text-sm text-muted">Gösterilecek veri bulunamadı.</p>
    </div>
  );
}

function DefaultUnauthorized() {
  return (
    <div className="alert-warn text-center">
      <p>Bu içeriği görüntüleme yetkiniz yok.</p>
    </div>
  );
}

function DefaultRetry() {
  return (
    <div className="card text-center">
      <p className="mb-4 text-sm text-muted">İstek tamamlanamadı.</p>
      <button type="button" className="btn-primary">
        Tekrar dene
      </button>
    </div>
  );
}

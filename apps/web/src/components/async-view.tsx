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
    <div role="status" className="rounded-lg border border-slate-200 bg-white p-8 text-center">
      <p className="text-sm text-slate-600">Yükleniyor…</p>
    </div>
  );
}

function DefaultError() {
  return (
    <div role="alert" className="rounded-lg border border-red-200 bg-red-50 p-8 text-center">
      <p className="text-sm text-red-800">Bir hata oluştu.</p>
    </div>
  );
}

function DefaultEmpty() {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-8 text-center">
      <p className="text-sm text-slate-600">Gösterilecek veri yok.</p>
    </div>
  );
}

function DefaultUnauthorized() {
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-8 text-center">
      <p className="text-sm text-amber-900">Bu içeriği görüntüleme yetkiniz yok.</p>
    </div>
  );
}

function DefaultRetry() {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-8 text-center">
      <p className="mb-3 text-sm text-slate-600">İstek tamamlanamadı.</p>
      <button
        type="button"
        className="rounded-md bg-miyuna-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700"
      >
        Tekrar dene
      </button>
    </div>
  );
}

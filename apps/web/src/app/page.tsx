import Link from "next/link";
import { AsyncView } from "@/components/async-view";

export default function HomePage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-4xl flex-col gap-8 px-6 py-16">
      <header className="space-y-2">
        <p className="text-sm font-medium uppercase tracking-wide text-miyuna-600">
          Miyuna
        </p>
        <h1 className="text-3xl font-semibold text-slate-900">
          Çocuk ürünleri için agent destekli analiz
        </h1>
        <p className="text-lg text-slate-600">
          Skoru değil, skorun kanıtını göster.
        </p>
      </header>

      <section className="grid gap-4 md:grid-cols-2">
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="mb-2 font-medium text-slate-900">Phase 2 — Auth & Tenancy</h2>
          <p className="mb-4 text-sm text-slate-600">
            Kayıt, e-posta doğrulama, MFA, cihaz doğrulama, multi-tenant workspace ve
            tenant isolation.
          </p>
          <div className="flex flex-wrap gap-3 text-sm">
            <Link className="text-miyuna-600 underline" href="/auth/register">
              Kayıt ol
            </Link>
            <Link className="text-miyuna-600 underline" href="/auth/login">
              Giriş yap
            </Link>
            <Link className="text-miyuna-600 underline" href="/workspace">
              Workspace
            </Link>
          </div>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="mb-2 font-medium text-slate-900">UI state örneği</h2>
          <AsyncView state="loading" />
        </div>
      </section>

      <footer className="text-xs text-slate-500">
        Decision-support / risk-assessment system — resmi sertifikasyon iddiası yoktur.
      </footer>
    </main>
  );
}

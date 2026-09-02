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
          <h2 className="mb-2 font-medium text-slate-900">Phase 1 — Foundation</h2>
          <p className="text-sm text-slate-600">
            Monorepo iskeleti, dev altyapısı (MongoDB, Redis, MinIO, MailHog), Go
            GraphQL API stub, internal Python agent stub ve Next.js web shell.
          </p>
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

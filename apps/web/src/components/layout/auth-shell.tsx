import { ReactNode } from "react";
import { Logo } from "@/components/brand/logo";

type AuthShellProps = {
  title: string;
  subtitle?: string;
  children: ReactNode;
};

export function AuthShell({ title, subtitle, children }: AuthShellProps) {
  return (
    <main className="grid min-h-screen lg:grid-cols-[1.05fr_1fr]">
      <aside className="relative hidden overflow-hidden bg-forest-deep px-12 py-12 text-paper lg:flex lg:flex-col">
        <div
          className="pointer-events-none absolute -left-20 top-16 h-72 w-72 rounded-full bg-forest-mid/30 blur-3xl"
          aria-hidden
        />
        <div
          className="pointer-events-none absolute bottom-10 right-0 h-80 w-80 rounded-full bg-clay/20 blur-3xl"
          aria-hidden
        />
        <Logo href="/" tone="paper" />
        <div className="relative mt-auto max-w-md space-y-5 pb-8">
          <p className="font-display text-4xl leading-tight text-paper">
            Sadece skoru değil,
            <br />
            arkasındaki kanıtı gör.
          </p>
          <p className="text-sm leading-relaxed text-paper/85">
            Çocuk ürünlerinde her iddia bir kaynağa bağlanır. Eksik alan uydurulmaz; resmi
            sertifikasyon iddiası yoktur.
          </p>
        </div>
      </aside>

      <section className="flex items-center justify-center px-6 py-16">
        <div className="w-full max-w-md space-y-8">
          <div className="lg:hidden">
            <Logo href="/" />
          </div>
          <header className="space-y-2">
            <p className="kicker">Miyuna</p>
            <h1 className="display text-3xl">{title}</h1>
            {subtitle && <p className="text-sm leading-relaxed text-muted">{subtitle}</p>}
          </header>
          {children}
        </div>
      </section>
    </main>
  );
}

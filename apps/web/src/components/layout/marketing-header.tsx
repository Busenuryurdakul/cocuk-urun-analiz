import Link from "next/link";
import { Logo } from "@/components/brand/logo";

export function MarketingHeader() {
  return (
    <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-6 py-6">
      <Logo />
      <nav className="flex items-center gap-2">
        <Link href="/auth/login" className="btn-ghost hidden sm:inline-flex">
          Giriş yap
        </Link>
        <Link href="/auth/register" className="btn-primary">
          Ücretsiz başla
        </Link>
      </nav>
    </header>
  );
}

export function MarketingFooter() {
  return (
    <footer className="mx-auto mt-8 w-full max-w-6xl border-t border-sand px-6 py-10">
      <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
        <Logo size="sm" />
        <p className="max-w-lg text-xs leading-relaxed text-muted">
          Miyuna bir karar destek / risk değerlendirme sistemidir. Resmi güvenlik sertifikasyonu,
          hukuki uygunluk garantisi veya regülasyon onayı iddiasında bulunmaz.
        </p>
      </div>
    </footer>
  );
}

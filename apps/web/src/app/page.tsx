import Link from "next/link";
import type { ReactNode } from "react";
import { HomeProductPreview } from "@/components/home-product-preview";
import { MarketingFooter, MarketingHeader } from "@/components/layout/marketing-header";

const CATEGORIES = [
  { label: "Oyuncak", d: "M5 10h14v10H5V10Zm3-4 4-3 4 3M12 10v10", tone: "bg-[#e6f0ec] border-[#c8ddd4] text-forest-deep" },
  { label: "Bakım", d: "M10 3h4l1 3H9l1-3Zm-1 3h6v15a2 2 0 0 1-2 2h-2a2 2 0 0 1-2-2V6Z", tone: "bg-[#eef1f8] border-[#d2d8ea] text-[#3d4568]" },
  { label: "Giyim", d: "M8 7 12 5l4 2 3 4-3 1v8H8v-8L5 11l3-4Zm2 1.5c.7.8 3.3.8 4 0", tone: "bg-[#faf0e8] border-[#ecd9c8] text-clay-deep" },
  { label: "Beslenme", d: "M9 5c1.8 0 3 1.7 3 4.2V21H6V9.2C6 6.7 7.2 5 9 5Zm6 4h2v12h-2m2-14c1 0 1.8.7 1.8 1.8 0 1.4-1.8 2.6-1.8 2.6", tone: "bg-[#f5f0e6] border-[#e6dcc8] text-[#6b5a3a]" },
];

export default function HomePage() {
  return (
    <div className="min-h-screen">
      <MarketingHeader />

      <main className="mx-auto max-w-6xl px-6">
        <section className="grid items-center gap-12 pb-8 pt-6 lg:grid-cols-[1.1fr_0.9fr] lg:pt-10">
          <div className="space-y-7">
            <p className="kicker">Çocuk ürünleri · kanıta dayalı karar desteği</p>

            <div className="relative max-w-xl">
              <EvidencePattern />
              <h1 className="display relative text-5xl leading-[1.08] sm:text-6xl">
                Bir skora değil,
                <span className="relative mt-1 block text-forest">
                  kanıtına güven.
                  <HandDrawnUnderline />
                </span>
              </h1>
            </div>

            <p className="max-w-xl text-lg leading-relaxed text-muted">
              Miyuna; satış kanalı yorumlarını, doğrudan kullanıcı deneyimlerini ve iki LLM&apos;in
              analizini tek raporda birleştirir. Her bulguyu kaynağıyla gösterir, eksik bilgiyi
              açıkça işaretler.
            </p>

            <div className="flex flex-wrap gap-3">
              <Link
                href="/auth/register"
                className="btn-accent shadow-[0_12px_28px_-8px_rgba(196,92,38,0.45)] hover:shadow-[0_16px_32px_-8px_rgba(196,92,38,0.5)]"
              >
                İlk analizini oluştur
                <span aria-hidden>→</span>
              </Link>
              <a href="#ornek-rapor" className="btn-secondary">
                Örnek raporu incele
              </a>
            </div>

            <TrustStrip />
            <CategoryStrip />
          </div>

          <HomeProductPreview />
        </section>

        <section className="mt-10 grid gap-4 sm:grid-cols-3">
          {[
            ["Kanıt önce", "Ciddi bir iddia, bağlı kanıt olmadan kesin dil kullanamaz."],
            ["Eksik alan görünür", "Kaynakta olmayan bilgi uydurulmaz — bilgi eksik olarak işaretlenir."],
            ["Sertifika değil", "Karar destek sistemidir; resmi onay veya güvenlik garantisi vermez."],
          ].map(([title, body]) => (
            <article key={title} className="card">
              <h2 className="font-display text-xl text-ink">{title}</h2>
              <p className="mt-2 text-sm leading-relaxed text-muted">{body}</p>
            </article>
          ))}
        </section>

        <section className="mt-16 grid gap-10 lg:grid-cols-2">
          <div className="space-y-4">
            <p className="kicker">Nasıl çalışır</p>
            <h2 className="display text-3xl">Üründen kanıtlı rapora dört adım</h2>
            <ol className="space-y-4 pt-2">
              {[
                ["Ürünü ekle", "Ürün bilgilerini elle, dosyayla veya izin verilen ürün bağlantısıyla ekle."],
                ["Kaynakları ayır", "Doğrudan kullanıcı deneyimleri ile satış kanalı yorumları ayrı değerlendirilir."],
                ["Ajan analiz etsin", "İki LLM, otomatik görev dağılımıyla analiz ve denetim süreçlerini yürütür."],
                ["Raporu incele", "Sinyaller, eksik bilgiler ve kaynak bağlantıları tek raporda sunulur."],
              ].map(([title, body], i) => (
                <li key={title} className="flex gap-4">
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-forest text-xs font-semibold text-paper">
                    {i + 1}
                  </span>
                  <div>
                    <p className="font-semibold text-ink">{title}</p>
                    <p className="text-sm text-muted">{body}</p>
                  </div>
                </li>
              ))}
            </ol>
          </div>

          <PlatformPanel />
        </section>
      </main>

      <MarketingFooter />
    </div>
  );
}

function PlatformPanel() {
  return (
    <div className="relative overflow-hidden rounded-2.5xl border border-sand/90 bg-paper p-6 shadow-lift sm:p-7">
      <div
        className="pointer-events-none absolute -right-8 -top-8 h-32 w-32 rounded-full bg-clay/10 blur-2xl"
        aria-hidden
      />
      <div className="absolute right-5 top-5 flex h-9 w-9 items-center justify-center rounded-full border border-sand bg-cream text-forest">
        <SyncIcon />
        <span className="sr-only">Senkronize çalışma alanı</span>
      </div>

      <div className="relative max-w-md space-y-3 pr-12">
        <p className="kicker">Platform</p>
        <h2 className="font-display text-2xl leading-tight sm:text-3xl">
          <span className="text-forest">Web&apos;de</span> başla,{" "}
          <span className="text-clay">masaüstünde</span> devam et.
        </h2>
        <p className="text-sm leading-relaxed text-muted">
          Tek çalışma alanı, ortak kanıt modeli ve tüm cihazlarda kesintisiz analiz deneyimi.
        </p>
      </div>

      <div className="relative mt-7 flex flex-col items-stretch gap-3 sm:flex-row sm:items-center">
        <PlatformSurfaceCard
          badge="Tarayıcıdan erişim"
          title="Web"
          subtitle="Tarayıcıdan anında eriş"
          description="Güvenli kayıt, doğrulama ve ürün analizi."
          tone="web"
          icon={<BrowserIcon />}
        />
        <PlatformConnector />
        <PlatformSurfaceCard
          badge="Windows + macOS"
          title="Masaüstü"
          subtitle="Kayıtlı cihaz güvencesi"
          description="Windows ve macOS için güvenli Electron uygulaması."
          tone="desktop"
          icon={<DesktopIcon />}
        />
      </div>
    </div>
  );
}

type PlatformSurfaceCardProps = {
  badge: string;
  title: string;
  subtitle: string;
  description: string;
  tone: "web" | "desktop";
  icon: ReactNode;
};

function PlatformSurfaceCard({ badge, title, subtitle, description, tone, icon }: PlatformSurfaceCardProps) {
  const surface =
    tone === "web"
      ? "border-forest/15 bg-forest-soft shadow-[0_14px_36px_-18px_rgba(30,77,69,0.45)]"
      : "border-clay/20 bg-clay-soft shadow-[0_14px_36px_-18px_rgba(196,92,38,0.4)]";

  return (
    <article className={`relative flex-1 rounded-2xl border p-5 transition hover:-translate-y-0.5 ${surface}`}>
      <span
        className={`absolute right-3 top-3 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
          tone === "web" ? "bg-paper/80 text-forest-deep" : "bg-paper/80 text-clay-deep"
        }`}
      >
        {badge}
      </span>
      <div
        className={`mb-4 flex h-14 w-14 items-center justify-center rounded-2xl ${
          tone === "web" ? "bg-paper/70 text-forest" : "bg-paper/70 text-clay"
        }`}
        aria-hidden
      >
        {icon}
      </div>
      <h3 className="font-display text-xl text-ink">{title}</h3>
      <p className="mt-1 text-sm font-semibold text-ink">{subtitle}</p>
      <p className="mt-2 text-sm leading-relaxed text-muted">{description}</p>
    </article>
  );
}

function PlatformConnector() {
  return (
    <div className="flex shrink-0 items-center justify-center py-1 sm:w-10 sm:flex-col sm:py-0" aria-hidden>
      <span className="hidden h-px flex-1 bg-gradient-to-r from-forest/25 via-sand to-clay/25 sm:block sm:h-auto sm:w-px sm:flex-none sm:bg-gradient-to-b" />
      <span className="mx-2 flex h-8 w-8 items-center justify-center rounded-full border border-sand bg-paper text-forest shadow-sm">
        <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none">
          <path d="M7 12h10M13 8l4 4-4 4M11 16l-4-4 4-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </span>
      <span className="hidden h-px flex-1 bg-gradient-to-r from-forest/25 via-sand to-clay/25 sm:block sm:h-auto sm:w-px sm:flex-none sm:bg-gradient-to-b" />
    </div>
  );
}

function SyncIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" aria-hidden>
      <path
        d="M18 4v4h-4M6 20v-4h4M19.5 8.5A7 7 0 0 0 7 7.5M4.5 15.5A7 7 0 0 0 17 16.5"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function BrowserIcon() {
  return (
    <svg viewBox="0 0 32 32" className="h-8 w-8" fill="none" aria-hidden>
      <rect x="4" y="6" width="24" height="20" rx="3" stroke="currentColor" strokeWidth="1.8" />
      <path d="M4 11h24" stroke="currentColor" strokeWidth="1.8" />
      <circle cx="8" cy="8.5" r="0.9" fill="currentColor" />
      <circle cx="11" cy="8.5" r="0.9" fill="currentColor" />
      <path d="M10 18h12M10 22h8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}

function DesktopIcon() {
  return (
    <svg viewBox="0 0 32 32" className="h-8 w-8" fill="none" aria-hidden>
      <rect x="5" y="7" width="22" height="15" rx="2.5" stroke="currentColor" strokeWidth="1.8" />
      <path d="M13 26h6M16 22v4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
      <path d="M9 26h14" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
      <rect x="9" y="11" width="14" height="8" rx="1.5" stroke="currentColor" strokeWidth="1.4" />
    </svg>
  );
}

function CategoryStrip() {
  return (
    <ul className="flex flex-wrap gap-2.5 pt-1">
      {CATEGORIES.map((item) => (
        <li
          key={item.label}
          className={`inline-flex items-center gap-2.5 rounded-full border px-4 py-2 text-sm font-medium ${item.tone}`}
        >
          <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" aria-hidden>
            <path
              d={item.d}
              stroke="currentColor"
              strokeWidth="1.7"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          {item.label}
        </li>
      ))}
    </ul>
  );
}

function TrustStrip() {
  const items = [
    { label: "Kaynak bağlantılı bulgular", icon: <LinkTrustIcon /> },
    { label: "Eksik bilgiler görünür", icon: <MissingTrustIcon /> },
    { label: "KVKK/GDPR kontrollü", icon: <ShieldTrustIcon /> },
    { label: "Web ve masaüstü", icon: <DevicesTrustIcon /> },
  ];

  return (
    <div className="rounded-2xl border border-sand/80 bg-paper/80 p-3 shadow-sm backdrop-blur-sm">
      <ul className="grid gap-2 sm:grid-cols-2">
        {items.map((item) => (
          <li
            key={item.label}
            className="flex items-center gap-2.5 rounded-xl bg-cream/70 px-3 py-2.5 text-sm text-ink"
          >
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-paper text-forest shadow-sm">
              {item.icon}
            </span>
            <span className="font-medium leading-snug">{item.label}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function EvidencePattern() {
  return (
    <svg
      className="pointer-events-none absolute -left-6 -top-4 h-40 w-56 text-forest/[0.02] sm:h-48 sm:w-72"
      viewBox="0 0 280 180"
      fill="none"
      aria-hidden
    >
      <circle cx="40" cy="40" r="4" fill="currentColor" />
      <circle cx="120" cy="24" r="3" fill="currentColor" />
      <circle cx="210" cy="56" r="4" fill="currentColor" />
      <circle cx="160" cy="120" r="3" fill="currentColor" />
      <circle cx="70" cy="130" r="3" fill="currentColor" />
      <path d="M40 40L120 24M120 24L210 56M40 40L70 130M210 56L160 120M70 130L160 120" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  );
}

function HandDrawnUnderline() {
  return (
    <svg
      className="absolute -bottom-1 left-0 h-3 w-[105%] text-clay"
      viewBox="0 0 220 12"
      fill="none"
      preserveAspectRatio="none"
      aria-hidden
    >
      <path
        d="M2 8C38 2 82 10 118 6C154 2 182 9 218 5"
        stroke="currentColor"
        strokeWidth="3"
        strokeLinecap="round"
      />
    </svg>
  );
}

function LinkTrustIcon() {
  return (
    <svg viewBox="0 0 20 20" className="h-4 w-4" fill="none" aria-hidden>
      <path d="M8 12 12 8m-2-1 3.5-3.5a2.5 2.5 0 1 1 3.5 3.5L11 12" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      <path d="M7 13 5 15a2 2 0 1 0 2.8 2.8l2-2" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}

function MissingTrustIcon() {
  return (
    <svg viewBox="0 0 20 20" className="h-4 w-4" fill="none" aria-hidden>
      <circle cx="10" cy="10" r="6.5" stroke="currentColor" strokeWidth="1.6" />
      <circle cx="10" cy="10" r="2" fill="currentColor" />
    </svg>
  );
}

function ShieldTrustIcon() {
  return (
    <svg viewBox="0 0 20 20" className="h-4 w-4" fill="none" aria-hidden>
      <path d="M10 3 15 5.5V10c0 3-2.2 5.4-5 6.5-2.8-1.1-5-3.5-5-6.5V5.5L10 3Z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
    </svg>
  );
}

function DevicesTrustIcon() {
  return (
    <svg viewBox="0 0 20 20" className="h-4 w-4" fill="none" aria-hidden>
      <rect x="3" y="4" width="10" height="7" rx="1.2" stroke="currentColor" strokeWidth="1.6" />
      <rect x="11" y="7" width="6" height="9" rx="1.2" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  );
}


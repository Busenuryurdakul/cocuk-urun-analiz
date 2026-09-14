import Link from "next/link";

export const HOW_TO_STEPS = [
  {
    title: "Hesap aç",
    body: "Ücretsiz kayıt ol, e-postandaki doğrulama bağlantısını onayla ve giriş yap. İlk girişte e-posta kodu veya cihaz onayı istenebilir.",
  },
  {
    title: "Organizasyon oluştur",
    body: "Çalışma alanında bir organizasyon aç. KVKK, GDPR veya her ikisini seç — uyumluluk katmanı kapatılmaz.",
  },
  {
    title: "Ürün ekle",
    body: "Ürünler sayfasından kaydı oluştur. Ad zorunludur; marka, kategori ve satış bağlantısı isteğe bağlıdır. Bilmediğin alanı boş bırak.",
  },
  {
    title: "Kaynakları tamamla",
    body: "Ürün sayfasında doğrudan kullanıcı deneyimini paylaş. Satış kanalı yorumları ayrı tutulur; ikisi karıştırılmaz.",
  },
  {
    title: "Analizi başlat",
    body: "Ürün sayfasında “Analizi Başlat”a bas. İki LLM raporu üretir; her bulgu kaynağa bağlanır, eksik bilgi uydurulmaz.",
  },
] as const;

export function HomeHowTo() {
  return (
    <section id="nasil-kullanilir" className="mt-16 scroll-mt-24">
      <div className="max-w-2xl space-y-3">
        <p className="kicker">Nasıl kullanılır</p>
        <h2 className="display text-3xl sm:text-4xl">İlk kayıttan kanıtlı rapora</h2>
        <p className="text-base leading-relaxed text-muted">
          Miyuna; çocuk ürününü kaydedip kaynaklarını toplayarak kanıtlı bir risk değerlendirmesi üretmeni sağlar.
          Aşağıdaki beş adım, sitede göreceğin sırayla aynıdır.
        </p>
      </div>

      <ol className="mt-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        {HOW_TO_STEPS.map((step, index) => (
          <li key={step.title} className="flex h-full flex-col rounded-2.5xl border border-sand/90 bg-paper p-5 shadow-card">
            <span className="flex h-9 w-9 items-center justify-center rounded-full bg-forest text-sm font-semibold text-paper">
              {index + 1}
            </span>
            <h3 className="mt-4 font-display text-xl text-ink">{step.title}</h3>
            <p className="mt-2 flex-1 text-sm leading-relaxed text-muted">{step.body}</p>
          </li>
        ))}
      </ol>

      <div className="mt-6 grid gap-4 lg:grid-cols-[1.4fr_0.6fr]">
        <aside className="rounded-2.5xl border border-forest/15 bg-forest-soft/70 p-5 sm:p-6">
          <p className="kicker">Bilmen gerekenler</p>
          <ul className="mt-4 space-y-3 text-sm leading-relaxed text-forest-deep">
            <li className="flex gap-3">
              <TipIcon />
              Kaynakta olmayan alan boş kalır; sistem tahminle doldurmaz.
            </li>
            <li className="flex gap-3">
              <TipIcon />
              Rapor bir karar destek çıktısıdır — resmi sertifika veya güvenlik garantisi değildir.
            </li>
            <li className="flex gap-3">
              <TipIcon />
              Aynı çalışma alanına web tarayıcısından veya masaüstü uygulamasından devam edebilirsin.
            </li>
          </ul>
        </aside>

        <div className="flex flex-col justify-between gap-4 rounded-2.5xl border border-clay/20 bg-clay-soft/60 p-5 sm:p-6">
          <div>
            <p className="kicker">Hazırsan</p>
            <p className="mt-2 font-display text-2xl leading-tight text-ink">İlk ürününü kaydet.</p>
            <p className="mt-2 text-sm leading-relaxed text-muted">
              Hesap açıldıktan sonra organizasyon ve ürün adımları çalışma alanında sırayla görünür.
            </p>
          </div>
          <Link href="/auth/register" className="btn-accent w-full sm:w-auto">
            Ücretsiz başla
            <span aria-hidden>→</span>
          </Link>
        </div>
      </div>
    </section>
  );
}

export function HomeHowToCompact() {
  return (
    <aside className="rounded-2.5xl border border-sand/90 bg-cream/70 p-5">
      <p className="kicker">Nasıl kullanılır</p>
      <h2 className="mt-2 font-display text-xl text-ink">Siteyi beş adımda kullan</h2>
      <ol className="mt-4 space-y-2.5">
        {HOW_TO_STEPS.map((step, index) => (
          <li key={step.title} className="flex items-baseline gap-3 text-sm">
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-forest text-[10px] font-semibold text-paper">
              {index + 1}
            </span>
            <span className="font-semibold text-ink">{step.title}</span>
          </li>
        ))}
      </ol>
      <p className="mt-3 text-xs leading-relaxed text-muted">
        Kayıt, organizasyon, ürün, kaynaklar ve kanıtlı rapor — her adım çalışma alanında sırayla görünür.
      </p>
      <Link href="/#nasil-kullanilir" className="link-quiet mt-3 inline-flex">
        Detaylı rehberi oku
      </Link>
    </aside>
  );
}

function TipIcon() {
  return (
    <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-paper text-forest shadow-sm" aria-hidden>
      <svg viewBox="0 0 16 16" className="h-3 w-3" fill="none">
        <path d="M4 8.2 6.6 11 12 5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </span>
  );
}

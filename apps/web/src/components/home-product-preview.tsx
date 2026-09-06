"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { graphqlRequest } from "@/lib/graphql";
import { PRODUCT_FIELD_SELECTION, fieldValue, formatPrice, missingFieldCount, productSubtitle, productTitle, type Product } from "@/lib/product";

type Workspace = { organizationId: string; name: string; type: string };

export function HomeProductPreview() {
  const [products, setProducts] = useState<Product[]>([]);
  const [orgId, setOrgId] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    graphqlRequest<{ myWorkspaces: Workspace[] }>(`{ myWorkspaces { organizationId name type } }`)
      .then(async (data) => {
        const org = data.myWorkspaces.find((ws) => ws.type === "ORGANIZATION") ?? data.myWorkspaces[0];
        if (!org) {
          return;
        }
        const result = await graphqlRequest<{ products: Product[] }>(
          `query($id: ID!) { products(organizationId: $id, limit: 24) { ${PRODUCT_FIELD_SELECTION} } }`,
          { id: org.organizationId },
        );
        if (cancelled) {
          return;
        }
        setOrgId(org.organizationId);
        setProducts(result.products);
        setSelectedId(result.products[0]?.id ?? null);
      })
      .catch(() => {
        /* public homepage: fallback sample when session/products are unavailable */
      })
      .finally(() => {
        if (!cancelled) {
          setReady(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const product = useMemo(
    () => products.find((item) => item.id === selectedId) ?? products[0] ?? null,
    [products, selectedId],
  );
  const categories = useMemo(() => {
    const seen = new Set<string>();
    const items: { id: string; label: string }[] = [];
    for (const item of products) {
      const label = fieldValue(item.category) ?? "Diğer";
      if (seen.has(label)) {
        continue;
      }
      seen.add(label);
      items.push({ id: item.id, label });
    }
    return items.slice(0, 6);
  }, [products]);

  if (!ready) {
    return <SampleReportCard />;
  }

  if (!product || !orgId) {
    return <SampleReportCard />;
  }

  return (
    <aside id="ornek-rapor" className="relative scroll-mt-24">
      <div className="absolute -inset-8 -z-10 rounded-[2.2rem] bg-clay/10 blur-2xl" aria-hidden />
      <article className="card shadow-lift">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="kicker">Veritabanındaki ürün</p>
            <h2 className="mt-2 font-display text-2xl leading-tight">{productTitle(product)}</h2>
            <p className="mt-1 text-sm text-muted">{productSubtitle(product) || "Kaynak alanları eksik olabilir"}</p>
          </div>
          <span className={product.brand.missing ? "badge-clay" : "badge-forest"}>
            {product.brand.missing ? "Marka eksik" : "Kanıtlı alanlar"}
          </span>
        </div>

        {categories.length > 1 ? (
          <ul className="mt-5 flex flex-wrap gap-2">
            {categories.map((item) => (
              <li key={item.label}>
                <button
                  type="button"
                  onClick={() => setSelectedId(item.id)}
                  className={`rounded-full border px-3 py-1 text-xs font-medium ${
                    selectedId === item.id ? "border-forest bg-forest-soft text-forest-deep" : "border-sand bg-paper text-muted"
                  }`}
                >
                  {item.label}
                </button>
              </li>
            ))}
          </ul>
        ) : null}

        <dl className="mt-6 grid grid-cols-3 gap-3 text-center">
          <MiniStat value={formatPrice(product) ?? "—"} label="Fiyat" />
          <MiniStat value={fieldValue(product.rating) ?? "—"} label="Puan" />
          <MiniStat value={String(missingFieldCount(product))} label="Eksik alan" />
        </dl>

        <ul className="mt-6 space-y-3 text-sm">
          <Finding missing={product.materials.missing} text={fieldValue(product.materials) ? `Malzeme: ${fieldValue(product.materials)}` : "Malzeme belgesi kaynakta yok — bilgi eksik."} />
          <Finding missing={product.targetAge.missing} text={fieldValue(product.targetAge) ? `Hedef yaş: ${fieldValue(product.targetAge)}` : "Hedef yaş kaynağında yok."} />
          <Finding missing={product.safetyWarnings.missing} text={fieldValue(product.safetyWarnings) ? `Uyarı: ${fieldValue(product.safetyWarnings)}` : "Sertifikasyon iddiası üretilmez."} />
        </ul>

        <Link href={`/org/${orgId}/products/${product.id}`} className="btn-secondary mt-6 w-full">
          Ürün kaydını aç
        </Link>
      </article>
    </aside>
  );
}

function MiniStat({ value, label }: { value: string; label: string }) {
  return (
    <div className="rounded-2xl bg-cream px-2 py-3">
      <dt className="font-display text-xl text-forest">{value}</dt>
      <dd className="text-[11px] uppercase tracking-wide text-muted">{label}</dd>
    </div>
  );
}

function Finding({ missing, text }: { missing: boolean; text: string }) {
  return (
    <li className="flex gap-3">
      <span className={`mt-1 h-2 w-2 shrink-0 rounded-full ${missing ? "bg-amber-500" : "bg-forest-mid"}`} />
      <span>{text}</span>
    </li>
  );
}

function SampleReportCard() {
  return (
    <aside id="ornek-rapor" className="relative scroll-mt-24">
      <div className="absolute -inset-8 -z-10 rounded-[2.2rem] bg-clay/10 blur-2xl" aria-hidden />
      <article className="card shadow-lift">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="kicker">Örnek rapor</p>
            <h2 className="mt-2 font-display text-2xl leading-tight">Ahşap aktivite küpü</h2>
            <p className="mt-1 text-sm text-muted">0–3 yaş · marka alanı eksik</p>
          </div>
          <span className="badge-clay">Kanıtlı</span>
        </div>
        <p className="mt-6 text-sm text-muted">
          Giriş yaptığınızda bu kart, organizasyonunuzdaki gerçek ürün kaydını ve eksik alanları gösterir.
        </p>
        <Link href="/auth/login" className="btn-secondary mt-6 w-full">
          Ürünlerimi gör
        </Link>
      </article>
    </aside>
  );
}

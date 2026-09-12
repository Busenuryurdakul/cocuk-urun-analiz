"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AnalysisPanel } from "@/components/analysis-panel";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { ProductMetaGrid, ProductSummaryStats } from "@/components/product-fields";
import { graphqlRequest } from "@/lib/graphql";
import { PRODUCT_FIELD_SELECTION, fieldValue, productSubtitle, productTitle, type Product } from "@/lib/product";

type UserExperience = {
  id: string;
  usageStatus: string;
  satisfactionLevel: string;
  rating?: number | null;
  issueType?: string | null;
  narrative: string;
  moderationStatus: string;
  createdAt: string;
};

type MarketplaceReview = {
  id: string;
  source: string;
  rating?: number | null;
  reviewText: string;
  reviewDate?: string | null;
  language: string;
  datasetEligibility: string;
};

type DatasetSummary = {
  totalRecords: number;
  ugcCount: number;
  marketplaceCount: number;
  eligibilityDistribution: { eligibility: string; count: number }[];
};

const SATISFACTION_LABELS: Record<string, string> = {
  VERY_SATISFIED: "Çok memnunum",
  SATISFIED: "Memnunum",
  NEUTRAL: "Nötr",
  UNSATISFIED: "Memnun değilim",
  VERY_UNSATISFIED: "Hiç memnun değilim",
};

const ISSUE_LABELS: Record<string, string> = {
  DURABILITY: "Dayanıklılık",
  BREAKAGE: "Kırılma",
  AGE_SIZE_MISMATCH: "Yaş/beden uyumsuzluğu",
  MATERIAL: "Malzeme",
  ODOR: "Koku",
  PACKAGING: "Ambalaj",
  USABILITY: "Kullanılabilirlik",
  QUALITY: "Kalite",
  SAFETY_RELATED_OBSERVATION: "Güvenlik gözlemi",
  OTHER: "Diğer",
};

export default function ProductDetailPage() {
  const params = useParams<{ id: string; productId: string }>();
  const orgId = params.id;
  const productId = params.productId;

  const [product, setProduct] = useState<Product | null>(null);
  const [experiences, setExperiences] = useState<UserExperience[]>([]);
  const [reviews, setReviews] = useState<MarketplaceReview[]>([]);
  const [summary, setSummary] = useState<DatasetSummary | null>(null);
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized">("loading");

  const [usageStatus, setUsageStatus] = useState("USED");
  const [satisfaction, setSatisfaction] = useState("SATISFIED");
  const [rating, setRating] = useState("");
  const [issueType, setIssueType] = useState("");
  const [narrative, setNarrative] = useState("");
  const [consentAck, setConsentAck] = useState(false);
  const [formState, setFormState] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [formError, setFormError] = useState("");

  const load = useCallback(async () => {
    setView("loading");
    try {
      const data = await graphqlRequest<{
        product: Product;
        userExperiences: UserExperience[];
        productMarketplaceReviews: MarketplaceReview[];
        datasetEligibilitySummary: DatasetSummary;
      }>(
        `query($orgId: ID!, $productId: ID!) {
          product(organizationId: $orgId, productId: $productId) {
            ${PRODUCT_FIELD_SELECTION}
          }
          userExperiences(organizationId: $orgId, productId: $productId) {
            id usageStatus satisfactionLevel rating issueType narrative moderationStatus createdAt
          }
          productMarketplaceReviews(organizationId: $orgId, productId: $productId) {
            id source rating reviewText reviewDate language datasetEligibility
          }
          datasetEligibilitySummary(organizationId: $orgId) {
            totalRecords ugcCount marketplaceCount
            eligibilityDistribution { eligibility count }
          }
        }`,
        { orgId, productId },
      );
      setProduct(data.product);
      setExperiences(data.userExperiences);
      setReviews(data.productMarketplaceReviews);
      setSummary(data.datasetEligibilitySummary);
      setView("success");
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setView("unauthorized");
      } else {
        setView("error");
      }
    }
  }, [orgId, productId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function grantConsentAndSubmit(e: FormEvent) {
    e.preventDefault();
    if (!consentAck) {
      setFormError("Devam etmek için gizlilik bildirimini onaylamalısınız.");
      setFormState("error");
      return;
    }
    setFormState("loading");
    setFormError("");
    try {
      await graphqlRequest(
        `mutation($input: GrantConsentInput!) { grantConsent(input: $input) { id } }`,
        { input: { purpose: "DATA_PROCESSING", organizationId: orgId } },
      );
    } catch {
      // Consent may already exist; create will validate.
    }
    try {
      await graphqlRequest(
        `mutation($input: CreateUserExperienceInput!) {
          createUserExperience(input: $input) { id moderationStatus }
        }`,
        {
          input: {
            organizationId: orgId,
            productId,
            usageStatus,
            satisfactionLevel: satisfaction,
            rating: rating ? Number(rating) : null,
            issueType: issueType || null,
            narrative,
          },
        },
      );
      setNarrative("");
      setRating("");
      setIssueType("");
      setConsentAck(false);
      setFormState("success");
      await load();
    } catch (err) {
      setFormState("error");
      setFormError(err instanceof Error ? err.message : "Gönderim başarısız");
    }
  }

  return (
    <AppShell
      title={product ? productTitle(product) : "Ürün detayı"}
      kicker="Ürün"
      orgId={orgId}
      restricted={view === "unauthorized"}
      description={product ? productSubtitle(product) || "Kanıt, deneyim ve marketplace kaynakları." : "Kanıt, deneyim ve marketplace kaynakları."}
    >
      {view === "loading" && <AsyncView state="loading" />}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && (
        <AsyncView
          state="retry"
          retry={
            <button type="button" onClick={() => void load()} className="btn-secondary">
              Tekrar dene
            </button>
          }
        />
      )}

      {view === "success" && product && (
        <div className="space-y-6">
          <section className="card space-y-4">
            <div>
              <p className="kicker">Veritabanı kaydı</p>
              <h2 className="mt-2 font-display text-2xl">{productTitle(product)}</h2>
              {fieldValue(product.description) ? (
                <p className="mt-2 max-w-2xl text-sm leading-relaxed text-muted">{fieldValue(product.description)}</p>
              ) : (
                <p className="mt-2 text-sm text-muted">Açıklama kaynağında yok — üretilmez.</p>
              )}
            </div>
            <ProductSummaryStats product={product} />
            <ProductMetaGrid product={product} />
          </section>

          <AnalysisPanel orgId={orgId} productId={productId} canStart canCancel />

          <section className="card space-y-4">
            <div>
              <h2 className="font-display text-2xl">Miyuna deneyimleri</h2>
              <p className="mt-1 text-sm text-muted">Doğrudan kullanıcı deneyimleri — satış kanalı yorumlarından ayrı tutulur.</p>
            </div>

            <form onSubmit={grantConsentAndSubmit} className="space-y-4 border-t border-sand pt-4">
              <h3 className="font-semibold">Deneyimini paylaş</h3>

              <fieldset className="space-y-2">
                <legend className="text-sm font-medium">Bu ürünü kullandın mı?</legend>
                <label className="flex items-center gap-2 text-sm">
                  <input type="radio" name="usage" value="USING" checked={usageStatus === "USING"} onChange={() => setUsageStatus("USING")} />
                  Kullanıyorum
                </label>
                <label className="flex items-center gap-2 text-sm">
                  <input type="radio" name="usage" value="USED" checked={usageStatus === "USED"} onChange={() => setUsageStatus("USED")} />
                  Kullandım
                </label>
              </fieldset>

              <label className="label">
                Deneyimin nasıldı?
                <select className="input" value={satisfaction} onChange={(e) => setSatisfaction(e.target.value)}>
                  {Object.entries(SATISFACTION_LABELS).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </select>
              </label>

              <label className="label">
                Puanın (1–5, isteğe bağlı)
                <input type="number" min={1} max={5} className="input" value={rating} onChange={(e) => setRating(e.target.value)} />
              </label>

              <label className="label">
                Bir sorun yaşadın mı?
                <select className="input" value={issueType} onChange={(e) => setIssueType(e.target.value)}>
                  <option value="">Sorun yok / belirtmek istemiyorum</option>
                  {Object.entries(ISSUE_LABELS).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </select>
              </label>

              <label className="label">
                Deneyimini anlat *
                <textarea required minLength={10} className="input" rows={4} value={narrative} onChange={(e) => setNarrative(e.target.value)} />
              </label>

              <div className="rounded-2xl bg-cream p-4 text-xs leading-relaxed text-muted">
                <p className="mb-3">
                  KVKK kapsamında deneyim metniniz yalnızca ürün analizi amacıyla işlenir. Çocuk adı, tam doğum tarihi,
                  adres veya gereksiz kişisel bilgi paylaşmayın.
                </p>
                <label className="flex items-start gap-2 text-ink">
                  <input type="checkbox" checked={consentAck} onChange={(e) => setConsentAck(e.target.checked)} />
                  <span>
                    Veri işleme onayını okudum ve kabul ediyorum.{" "}
                    <Link href={`/org/${orgId}/compliance`} className="font-semibold text-forest underline">
                      Uyumluluk sayfası
                    </Link>
                  </span>
                </label>
              </div>

              {formState === "success" && (
                <p className="alert-success">Deneyiminiz alındı. Moderasyon durumu: PENDING.</p>
              )}
              {formState === "error" && <p className="alert-error">{formError}</p>}

              <button type="submit" disabled={formState === "loading"} className="btn-primary">
                {formState === "loading" ? "Gönderiliyor…" : "Deneyimi gönder"}
              </button>
            </form>

            {experiences.length === 0 ? (
              <AsyncView state="empty" empty={<p className="text-sm text-muted">Henüz Miyuna deneyimi yok.</p>} />
            ) : (
              <ul className="space-y-3 border-t border-sand pt-4">
                {experiences.map((exp) => (
                  <li key={exp.id} className="rounded-2xl bg-cream p-4 text-sm">
                    <div className="mb-2 flex flex-wrap gap-2">
                      <span className="badge-forest">Kullanıcı içeriği</span>
                      <span className="badge-muted">{exp.moderationStatus}</span>
                      <span className="text-muted">{SATISFACTION_LABELS[exp.satisfactionLevel] ?? exp.satisfactionLevel}</span>
                    </div>
                    <p>{exp.narrative}</p>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="card space-y-4">
            <div>
              <h2 className="font-display text-2xl">Satış kanalı yorumları</h2>
              <p className="mt-1 text-sm text-muted">Satış kanallarından gelen üçüncü taraf yorumlar.</p>
            </div>
            {reviews.length === 0 ? (
              <AsyncView state="empty" empty={<p className="text-sm text-muted">Henüz satış kanalı yorumu yok.</p>} />
            ) : (
              <ul className="space-y-3">
                {reviews.map((rev) => (
                  <li key={rev.id} className="rounded-2xl bg-cream p-4 text-sm">
                    <div className="mb-2 flex flex-wrap gap-2">
                      <span className="badge-muted">{rev.source}</span>
                      {rev.rating != null && <span className="text-xs">★ {rev.rating}</span>}
                      <span className="text-xs text-muted">{rev.language}</span>
                    </div>
                    <p>{rev.reviewText}</p>
                  </li>
                ))}
              </ul>
            )}
          </section>

          {summary && (
            <section className="card text-sm">
              <h2 className="font-display text-2xl">Veri seti durumu</h2>
              <dl className="mt-4 grid grid-cols-3 gap-3 text-center">
                <div className="rounded-2xl bg-cream px-2 py-3">
                  <dt className="font-display text-2xl text-forest">{summary.totalRecords}</dt>
                  <dd className="text-[11px] uppercase tracking-wide text-muted">Toplam</dd>
                </div>
                <div className="rounded-2xl bg-cream px-2 py-3">
                  <dt className="font-display text-2xl text-forest">{summary.ugcCount}</dt>
                  <dd className="text-[11px] uppercase tracking-wide text-muted">Kullanıcı</dd>
                </div>
                <div className="rounded-2xl bg-cream px-2 py-3">
                  <dt className="font-display text-2xl text-forest">{summary.marketplaceCount}</dt>
                  <dd className="text-[11px] uppercase tracking-wide text-muted">Satış kanalı</dd>
                </div>
              </dl>
              {summary.eligibilityDistribution.length > 0 && (
                <ul className="mt-4 space-y-1">
                  {summary.eligibilityDistribution.map((row) => (
                    <li key={row.eligibility} className="flex justify-between text-xs text-muted">
                      <span>{row.eligibility}</span>
                      <span>{row.count}</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}
        </div>
      )}
    </AppShell>
  );
}

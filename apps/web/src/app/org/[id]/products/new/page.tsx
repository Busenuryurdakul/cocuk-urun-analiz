"use client";

import { useParams, useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlErrorMessage, graphqlRequest } from "@/lib/graphql";

function slugify(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, "-");
}

function buildCreateProductInput(params: {
  orgId: string;
  name: string;
  brand: string;
  category: string;
  description: string;
  targetAge: string;
  materials: string;
  safetyWarnings: string;
  currentPrice: string;
  originalPrice: string;
  currency: string;
  seller: string;
  rating: string;
  reviewCount: string;
  stockStatus: string;
  sku: string;
  source: string;
  sourceProductId: string;
  sourceUrl: string;
}): Record<string, string> {
  const input: Record<string, string> = {
    organizationId: params.orgId,
    name: params.name.trim(),
    source: params.source,
    sourceProductId: params.sourceProductId.trim() || slugify(params.name),
  };

  const optional: Array<[keyof typeof params, string]> = [
    ["brand", params.brand],
    ["category", params.category],
    ["description", params.description],
    ["targetAge", params.targetAge],
    ["materials", params.materials],
    ["safetyWarnings", params.safetyWarnings],
    ["currentPrice", params.currentPrice],
    ["originalPrice", params.originalPrice],
    ["currency", params.currency],
    ["seller", params.seller],
    ["rating", params.rating],
    ["reviewCount", params.reviewCount],
    ["stockStatus", params.stockStatus],
    ["sku", params.sku],
    ["sourceUrl", params.sourceUrl],
  ];

  for (const [key, value] of optional) {
    const trimmed = value.trim();
    if (trimmed) {
      input[key] = trimmed;
    }
  }

  return input;
}

function validateProductForm(values: {
  currentPrice: string;
  originalPrice: string;
  rating: string;
  reviewCount: string;
  sourceUrl: string;
}): string | null {
  const pricePattern = /^\d+(\.\d+)?$/;
  if (values.currentPrice.trim() && !pricePattern.test(values.currentPrice.trim())) {
    return "Güncel fiyat pozitif bir sayı olmalıdır.";
  }
  if (values.originalPrice.trim() && !pricePattern.test(values.originalPrice.trim())) {
    return "Liste fiyatı pozitif bir sayı olmalıdır.";
  }
  if (values.rating.trim() && !pricePattern.test(values.rating.trim())) {
    return "Puan sayısal olmalıdır.";
  }
  if (values.reviewCount.trim() && !/^\d+$/.test(values.reviewCount.trim())) {
    return "Yorum sayısı tam sayı olmalıdır.";
  }
  if (values.sourceUrl.trim()) {
    try {
      const url = new URL(values.sourceUrl.trim());
      if (!["http:", "https:"].includes(url.protocol)) {
        return "Kaynak URL http veya https ile başlamalıdır.";
      }
    } catch {
      return "Kaynak URL geçerli bir adres olmalıdır.";
    }
  }
  return null;
}

export default function NewProductPage() {
  const params = useParams<{ id: string }>();
  const orgId = params.id;
  const router = useRouter();
  const [name, setName] = useState("");
  const [brand, setBrand] = useState("");
  const [category, setCategory] = useState("");
  const [description, setDescription] = useState("");
  const [targetAge, setTargetAge] = useState("");
  const [materials, setMaterials] = useState("");
  const [safetyWarnings, setSafetyWarnings] = useState("");
  const [currentPrice, setCurrentPrice] = useState("");
  const [originalPrice, setOriginalPrice] = useState("");
  const [currency, setCurrency] = useState("");
  const [seller, setSeller] = useState("");
  const [rating, setRating] = useState("");
  const [reviewCount, setReviewCount] = useState("");
  const [stockStatus, setStockStatus] = useState("");
  const [sku, setSku] = useState("");
  const [source, setSource] = useState("MIYUNA");
  const [sourceProductId, setSourceProductId] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [state, setState] = useState<"idle" | "loading" | "error" | "unauthorized">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    const validationError = validateProductForm({ currentPrice, originalPrice, rating, reviewCount, sourceUrl });
    if (validationError) {
      setState("error");
      setError(validationError);
      return;
    }

    setState("loading");
    setError("");
    try {
      const data = await graphqlRequest<{ createProduct: { product: { id: string } } }>(
        `mutation($input: CreateProductInput!) {
          createProduct(input: $input) {
            product { id }
          }
        }`,
        {
          input: buildCreateProductInput({
            orgId,
            name,
            brand,
            category,
            description,
            targetAge,
            materials,
            safetyWarnings,
            currentPrice,
            originalPrice,
            currency,
            seller,
            rating,
            reviewCount,
            stockStatus,
            sku,
            source,
            sourceProductId,
            sourceUrl,
          }),
        },
      );
      router.push(`/org/${orgId}/products/${data.createProduct.product.id}`);
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setState("unauthorized");
      } else {
        setState("error");
        setError(graphqlErrorMessage(err, "Kayıt başarısız"));
      }
    }
  }

  return (
    <AppShell
      title="Yeni ürün"
      kicker="Ürünler"
      orgId={orgId}
      restricted={state === "unauthorized"}
      description="Kaynakta olmayan alanları boş bırakın; sistem bunları uydurmaz."
    >
      {state === "unauthorized" && <AsyncView state="unauthorized" />}
      {state !== "unauthorized" && (
        <form onSubmit={onSubmit} className="card max-w-xl space-y-4">
          <label className="label">
            Ürün adı *
            <input required className="input" value={name} onChange={(e) => setName(e.target.value)} />
          </label>
          <label className="label">
            Marka
            <input className="input" value={brand} onChange={(e) => setBrand(e.target.value)} />
          </label>
          <label className="label">
            Kategori
            <input className="input" value={category} onChange={(e) => setCategory(e.target.value)} />
          </label>
          <label className="label">
            Açıklama
            <textarea className="input" rows={3} value={description} onChange={(e) => setDescription(e.target.value)} />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="label">
              Hedef yaş
              <input className="input" value={targetAge} onChange={(e) => setTargetAge(e.target.value)} />
            </label>
            <label className="label">
              SKU
              <input className="input" value={sku} onChange={(e) => setSku(e.target.value)} />
            </label>
            <label className="label">
              Malzeme
              <input className="input" value={materials} onChange={(e) => setMaterials(e.target.value)} />
            </label>
            <label className="label">
              Güvenlik uyarıları
              <input className="input" value={safetyWarnings} onChange={(e) => setSafetyWarnings(e.target.value)} />
            </label>
            <label className="label">
              Güncel fiyat
              <input className="input" inputMode="decimal" min="0" value={currentPrice} onChange={(e) => setCurrentPrice(e.target.value)} />
            </label>
            <label className="label">
              Liste fiyatı
              <input className="input" inputMode="decimal" min="0" value={originalPrice} onChange={(e) => setOriginalPrice(e.target.value)} />
            </label>
            <label className="label">
              Para birimi
              <input className="input" value={currency} onChange={(e) => setCurrency(e.target.value)} />
            </label>
            <label className="label">
              Satıcı
              <input className="input" value={seller} onChange={(e) => setSeller(e.target.value)} />
            </label>
            <label className="label">
              Puan
              <input className="input" inputMode="decimal" min="0" max="5" value={rating} onChange={(e) => setRating(e.target.value)} />
            </label>
            <label className="label">
              Yorum sayısı
              <input className="input" inputMode="numeric" min="0" value={reviewCount} onChange={(e) => setReviewCount(e.target.value)} />
            </label>
            <label className="label">
              Stok
              <input className="input" value={stockStatus} onChange={(e) => setStockStatus(e.target.value)} />
            </label>
          </div>
          <label className="label">
            Kaynak
            <select className="input" value={source} onChange={(e) => setSource(e.target.value)}>
              <option value="MIYUNA">Miyuna</option>
              <option value="HEPSIBURADA">Hepsiburada</option>
              <option value="TRENDYOL">Trendyol</option>
              <option value="OTHER">Diğer</option>
            </select>
          </label>
          <label className="label">
            Kaynak ürün ID
            <input className="input" value={sourceProductId} onChange={(e) => setSourceProductId(e.target.value)} />
          </label>
          <label className="label">
            Kaynak URL
            <input className="input" type="url" value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} placeholder="https://..." />
          </label>
          {state === "error" && <p className="alert-error">{error}</p>}
          <button type="submit" disabled={state === "loading"} className="btn-primary">
            {state === "loading" ? "Kaydediliyor…" : "Kaydet"}
          </button>
        </form>
      )}
    </AppShell>
  );
}

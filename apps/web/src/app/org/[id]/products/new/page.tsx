"use client";

import { useParams, useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlRequest } from "@/lib/graphql";

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
          input: {
            organizationId: orgId,
            name,
            brand: brand || null,
            category: category || null,
            description: description || null,
            targetAge: targetAge || null,
            materials: materials || null,
            safetyWarnings: safetyWarnings || null,
            currentPrice: currentPrice || null,
            originalPrice: originalPrice || null,
            currency: currency || null,
            seller: seller || null,
            rating: rating || null,
            reviewCount: reviewCount || null,
            stockStatus: stockStatus || null,
            sku: sku || null,
            source,
            sourceProductId: sourceProductId || name.toLowerCase().replace(/\s+/g, "-"),
            sourceUrl: sourceUrl || null,
          },
        },
      );
      router.push(`/org/${orgId}/products/${data.createProduct.product.id}`);
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setState("unauthorized");
      } else {
        setState("error");
        setError(err instanceof Error ? err.message : "Kayıt başarısız");
      }
    }
  }

  return (
    <AppShell
      title="Yeni ürün"
      kicker="Ürünler"
      orgId={orgId}
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
              <input className="input" value={currentPrice} onChange={(e) => setCurrentPrice(e.target.value)} />
            </label>
            <label className="label">
              Liste fiyatı
              <input className="input" value={originalPrice} onChange={(e) => setOriginalPrice(e.target.value)} />
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
              <input className="input" value={rating} onChange={(e) => setRating(e.target.value)} />
            </label>
            <label className="label">
              Yorum sayısı
              <input className="input" value={reviewCount} onChange={(e) => setReviewCount(e.target.value)} />
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
            <input className="input" value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} />
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

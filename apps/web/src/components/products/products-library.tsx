"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { MetricCard } from "@/components/miyuna/metric-card";
import { PremiumProductCard } from "@/components/miyuna/premium-product-card";
import { StatusRibbon } from "@/components/miyuna/status-ribbon";
import { FilterPill } from "@/components/ui/filter-pill";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { PackageIcon } from "@/components/ui/icons";
import type { AnalysisRun } from "@/lib/analysis-types";
import {
  computeDashboardStats,
  enrichProduct,
  formatShortDate,
  latestRunByProduct,
  parseProductImageUrl,
  productSourceLabel,
  type EnrichedProduct,
} from "@/lib/product-intelligence";
import { fieldValue, productTitle, type Product } from "@/lib/product";
import { cn } from "@/lib/cn";

type ViewMode = "grid" | "table";

type ProductsLibraryProps = {
  orgId: string;
  products: Product[];
  analysisRuns: AnalysisRun[];
};

type AnalysisFilter = "all" | "none" | "completed" | "running" | "failed";

function matchesSearch(item: EnrichedProduct, query: string): boolean {
  if (!query.trim()) return true;
  const q = query.trim().toLowerCase();
  const p = item.product;
  return [productTitle(p), fieldValue(p.brand), fieldValue(p.category), fieldValue(p.sku)]
    .filter(Boolean)
    .some((v) => v!.toLowerCase().includes(q));
}

function ProductsTable({ items, orgId }: { items: EnrichedProduct[]; orgId: string }) {
  return (
    <div className="overflow-hidden rounded-card-lg border border-border/80 bg-surface shadow-card">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="sticky top-0 z-10 border-b border-border/80 bg-secondary/80 backdrop-blur-sm">
            <tr>
              <th className="px-4 py-3 font-medium text-text-secondary">Ürün</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Marka</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Kategori</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Risk</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Recall</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Güven</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Son analiz</th>
              <th className="px-4 py-3 font-medium text-text-secondary">Aksiyon</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const imageUrl = parseProductImageUrl(item.product);
              return (
                <tr
                  key={item.product.id}
                  className="border-b border-border/50 transition-colors duration-normal last:border-0 hover:bg-secondary/40"
                >
                  <td className="px-4 py-4">
                    <div className="flex items-center gap-3">
                      <div className="h-10 w-10 shrink-0 overflow-hidden rounded-lg bg-secondary">
                        {imageUrl ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={imageUrl} alt="" className="h-full w-full object-cover" />
                        ) : (
                          <div className="flex h-full w-full items-center justify-center text-muted">
                            <PackageIcon size={16} />
                          </div>
                        )}
                      </div>
                      <span className="max-w-[200px] truncate font-medium text-ink">
                        {productTitle(item.product)}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-4 text-text-secondary">{fieldValue(item.product.brand) ?? "—"}</td>
                  <td className="px-4 py-4 text-text-secondary">{fieldValue(item.product.category) ?? "—"}</td>
                  <td className="px-4 py-4">
                    <StatusRibbon variant={item.riskRibbon} size="sm" label={item.riskLabel} />
                  </td>
                  <td className="px-4 py-4">
                    {item.recallRibbon ? (
                      <StatusRibbon variant={item.recallRibbon} size="sm" />
                    ) : (
                      <span className="text-muted">—</span>
                    )}
                  </td>
                  <td className="px-4 py-4 text-text-secondary">
                    {item.confidence !== undefined ? `${Math.round(item.confidence * 100)}%` : "—"}
                  </td>
                  <td className="px-4 py-4 text-muted">{formatShortDate(item.lastAnalysis) ?? "—"}</td>
                  <td className="px-4 py-4">
                    <Link
                      href={`/org/${orgId}/products/${item.product.id}`}
                      className="text-sm font-medium text-primary hover:text-primary-hover"
                    >
                      Detay
                    </Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function ProductsLibrary({ orgId, products, analysisRuns }: ProductsLibraryProps) {
  const [viewMode, setViewMode] = useState<ViewMode>("grid");
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [riskFilter, setRiskFilter] = useState("");
  const [recallFilter, setRecallFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [analysisFilter, setAnalysisFilter] = useState<AnalysisFilter>("all");

  const runMap = useMemo(() => latestRunByProduct(analysisRuns), [analysisRuns]);
  const enriched = useMemo(
    () => products.map((p) => enrichProduct(p, runMap.get(p.id))),
    [products, runMap],
  );
  const stats = useMemo(() => computeDashboardStats(products, analysisRuns), [products, analysisRuns]);

  const categories = useMemo(() => {
    const values = products.map((p) => fieldValue(p.category)).filter(Boolean) as string[];
    return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b, "tr"));
  }, [products]);

  const sources = useMemo(() => {
    const values = products.map(productSourceLabel).filter(Boolean) as string[];
    return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b, "tr"));
  }, [products]);

  const filtered = useMemo(() => {
    return enriched.filter((item) => {
      if (!matchesSearch(item, search)) return false;
      if (category && fieldValue(item.product.category) !== category) return false;
      if (sourceFilter && productSourceLabel(item.product) !== sourceFilter) return false;

      if (riskFilter) {
        const risk = item.run?.finalResult?.overallRisk;
        if (!risk || risk !== riskFilter) return false;
      }

      if (recallFilter === "matched") {
        if (!item.run?.recalls?.some((r) => r.matched)) return false;
      } else if (recallFilter === "none") {
        if (item.run?.recalls?.some((r) => r.matched)) return false;
      }

      if (analysisFilter === "none" && item.run) return false;
      if (analysisFilter === "completed" && item.run?.status !== "COMPLETED") return false;
      if (analysisFilter === "running" && item.run?.status !== "RUNNING" && item.run?.status !== "PENDING") {
        return false;
      }
      if (analysisFilter === "failed" && item.run?.status !== "FAILED" && item.run?.status !== "REJECTED") {
        return false;
      }

      return true;
    });
  }, [enriched, search, category, sourceFilter, riskFilter, recallFilter, analysisFilter]);

  const activeFilterCount = [category, riskFilter, recallFilter, sourceFilter].filter(Boolean).length;

  return (
    <div className="space-y-8">
      {/* Summary metrics */}
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Tüm ürünler" value={stats.totalProducts} context="Canonical katalog" />
        {stats.analyzedProducts > 0 && (
          <MetricCard label="Analiz edilenler" value={stats.analyzedProducts} context="Tamamlanan analiz" />
        )}
        {stats.reviewRequiredCount > 0 && (
          <MetricCard label="İnceleme gereken" value={stats.reviewRequiredCount} context="Karar veya recall" />
        )}
        {stats.recallMatchCount > 0 && (
          <MetricCard label="Recall eşleşmeleri" value={stats.recallMatchCount} context="Analiz kayıtları" />
        )}
      </section>

      {/* Filter toolbar */}
      <section className="rounded-card-lg border border-border/70 bg-surface p-4 shadow-card sm:p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end">
          <div className="min-w-0 flex-1">
            <Input
              label="Ara"
              placeholder="Ürün adı, marka, kategori, SKU…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="mt-0"
            />
          </div>
          <div className="grid flex-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Select label="Kategori" value={category} onChange={(e) => setCategory(e.target.value)} className="mt-0">
              <option value="">Tümü</option>
              {categories.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </Select>
            <Select label="Risk" value={riskFilter} onChange={(e) => setRiskFilter(e.target.value)} className="mt-0">
              <option value="">Tümü</option>
              <option value="LOW">Düşük</option>
              <option value="MEDIUM">Orta</option>
              <option value="HIGH">Yüksek</option>
              <option value="CRITICAL">Kritik</option>
            </Select>
            <Select
              label="Recall"
              value={recallFilter}
              onChange={(e) => setRecallFilter(e.target.value)}
              className="mt-0"
            >
              <option value="">Tümü</option>
              <option value="matched">Eşleşme var</option>
              <option value="none">Eşleşme yok</option>
            </Select>
            <Select
              label="Kaynak"
              value={sourceFilter}
              onChange={(e) => setSourceFilter(e.target.value)}
              className="mt-0"
            >
              <option value="">Tümü</option>
              {sources.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </Select>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-border/60 pt-4">
          <span className="text-xs font-medium uppercase tracking-wider text-muted">Analiz durumu</span>
          {(
            [
              ["all", "Tümü"],
              ["none", "Analiz yok"],
              ["completed", "Tamamlandı"],
              ["running", "Devam ediyor"],
              ["failed", "Başarısız"],
            ] as const
          ).map(([key, label]) => (
            <FilterPill
              key={key}
              active={analysisFilter === key}
              onClick={() => setAnalysisFilter(key)}
            >
              {label}
            </FilterPill>
          ))}
          {activeFilterCount > 0 && (
            <button
              type="button"
              className="ml-auto text-xs font-medium text-primary hover:text-primary-hover"
              onClick={() => {
                setCategory("");
                setRiskFilter("");
                setRecallFilter("");
                setSourceFilter("");
                setAnalysisFilter("all");
                setSearch("");
              }}
            >
              Filtreleri temizle
            </button>
          )}
        </div>
      </section>

      {/* View toggle + results */}
      <section>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-muted">
            {filtered.length} / {products.length} ürün
          </p>
          <div className="flex rounded-xl border border-border bg-secondary p-1">
            {(["grid", "table"] as const).map((mode) => (
              <button
                key={mode}
                type="button"
                onClick={() => setViewMode(mode)}
                className={cn(
                  "rounded-lg px-3 py-1.5 text-xs font-medium transition-all duration-normal",
                  viewMode === mode ? "bg-surface text-primary shadow-sm" : "text-text-secondary hover:text-ink",
                )}
                aria-pressed={viewMode === mode}
              >
                {mode === "grid" ? "Grid" : "Tablo"}
              </button>
            ))}
          </div>
        </div>

        {filtered.length === 0 ? (
          <div className="rounded-card-lg border border-border/70 bg-secondary/40 px-6 py-12 text-center">
            <p className="text-sm font-medium text-ink">Eşleşen ürün bulunamadı</p>
            <p className="mt-1 text-sm text-muted">Filtreleri gevşetin veya yeni ürün ekleyin.</p>
            <Link href={`/org/${orgId}/products/new`} className="btn-primary mt-4 inline-flex">
              Ürün ekle
            </Link>
          </div>
        ) : viewMode === "grid" ? (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {filtered.map((item) => (
              <PremiumProductCard
                key={item.product.id}
                productName={productTitle(item.product)}
                brand={fieldValue(item.product.brand) ?? undefined}
                category={fieldValue(item.product.category) ?? undefined}
                imageUrl={parseProductImageUrl(item.product)}
                riskLevel={item.riskRibbon}
                confidence={item.confidence !== undefined ? Math.round(item.confidence * 100) : undefined}
                recallState={item.recallRibbon}
                lastAnalysis={formatShortDate(item.lastAnalysis)}
                href={`/org/${orgId}/products/${item.product.id}`}
                actionLabel="Analizi aç"
              />
            ))}
          </div>
        ) : (
          <ProductsTable items={filtered} orgId={orgId} />
        )}
      </section>
    </div>
  );
}

export function ProductsLibrarySkeleton() {
  return (
    <div className="space-y-8" aria-busy="true">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-card bg-secondary" />
        ))}
      </div>
      <div className="h-36 animate-pulse rounded-card-lg bg-secondary" />
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="h-44 animate-pulse rounded-card-lg bg-secondary" />
        ))}
      </div>
    </div>
  );
}

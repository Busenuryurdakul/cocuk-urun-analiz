"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { ProductsLibrary, ProductsLibrarySkeleton } from "@/components/products/products-library";
import { Card } from "@/components/ui/card";
import { ANALYSIS_RUN_FIELDS, type AnalysisRun } from "@/lib/analysis-types";
import { graphqlRequest } from "@/lib/graphql";
import { PRODUCT_FIELD_SELECTION, type Product } from "@/lib/product";

export default function OrgProductsPage() {
  const params = useParams<{ id: string }>();
  const orgId = params.id;
  const [products, setProducts] = useState<Product[]>([]);
  const [analysisRuns, setAnalysisRuns] = useState<AnalysisRun[]>([]);
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized">("loading");

  const load = useCallback(async () => {
    setView("loading");
    try {
      const [productData, runsData] = await Promise.all([
        graphqlRequest<{ products: Product[] }>(
          `query($id: ID!) {
            products(organizationId: $id, limit: 100) {
              ${PRODUCT_FIELD_SELECTION}
            }
          }`,
          { id: orgId },
        ),
        graphqlRequest<{ analysisRuns: AnalysisRun[] }>(
          `query($id: ID!) {
            analysisRuns(organizationId: $id, limit: 100) { ${ANALYSIS_RUN_FIELDS} }
          }`,
          { id: orgId },
        ).catch(() => ({ analysisRuns: [] as AnalysisRun[] })),
      ]);
      setProducts(productData.products);
      setAnalysisRuns(runsData.analysisRuns);
      setView("success");
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setView("unauthorized");
      } else {
        setView("error");
      }
    }
  }, [orgId]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <AppShell
      title="Ürünler"
      kicker="Product Intelligence"
      orgId={orgId}
      restricted={view === "unauthorized"}
      description="Ürün kataloğunu izle, analiz et ve güvenlik durumlarını karşılaştır."
      actions={
        view === "success" ? (
          <Link href={`/org/${orgId}/products/new`} className="btn-accent">
            Ürün ekle
          </Link>
        ) : undefined
      }
    >
      {view === "loading" && <ProductsLibrarySkeleton />}
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
      {view === "success" && products.length === 0 && (
        <Card variant="tinted" className="mx-auto max-w-lg text-center">
          <p className="font-heading text-xl text-ink">Katalog boş</p>
          <p className="mt-2 text-sm text-text-secondary">
            İlk ürününüzü ekleyerek güvenlik analizine başlayın. Eksik alanlar uydurulmaz; kaynak verisiyle gösterilir.
          </p>
          <Link href={`/org/${orgId}/products/new`} className="btn-primary mt-6 inline-flex">
            Ürün ekle
          </Link>
        </Card>
      )}
      {view === "success" && products.length > 0 && (
        <ProductsLibrary orgId={orgId} products={products} analysisRuns={analysisRuns} />
      )}
    </AppShell>
  );
}

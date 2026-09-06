"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { ProductCardMeta } from "@/components/product-fields";
import { graphqlRequest } from "@/lib/graphql";
import { PRODUCT_FIELD_SELECTION, type Product } from "@/lib/product";

export default function OrgProductsPage() {
  const params = useParams<{ id: string }>();
  const orgId = params.id;
  const [products, setProducts] = useState<Product[]>([]);
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized">("loading");

  const load = useCallback(async () => {
    setView("loading");
    try {
      const data = await graphqlRequest<{ products: Product[] }>(
        `query($id: ID!) {
          products(organizationId: $id, limit: 100) {
            ${PRODUCT_FIELD_SELECTION}
          }
        }`,
        { id: orgId },
      );
      setProducts(data.products);
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
      kicker="Organizasyon"
      orgId={orgId}
      description="Veritabanındaki canonical ürünler — eksik alanlar uydurulmaz, kaynaklarıyla gösterilir."
      actions={
        <Link href={`/org/${orgId}/products/new`} className="btn-accent">
          Ürün ekle
        </Link>
      }
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
      {view === "success" && products.length === 0 && <AsyncView state="empty" />}
      {view === "success" && products.length > 0 && (
        <ul className="grid gap-3">
          {products.map((p) => (
            <li key={p.id}>
              <Link
                href={`/org/${orgId}/products/${p.id}`}
                className="card block transition hover:-translate-y-0.5 hover:shadow-lift"
              >
                <ProductCardMeta product={p} />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </AppShell>
  );
}

"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlRequest } from "@/lib/graphql";

type ProductField = { value?: string | null; missing: boolean };
type Product = {
  id: string;
  name: ProductField;
  brand: ProductField;
  category: ProductField;
  createdAt: string;
};

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
            id
            name { value missing }
            brand { value missing }
            category { value missing }
            createdAt
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
      description="Canonical ürünler — analiz, deneyim ve marketplace kaynakları buradan açılır."
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
                <p className="font-display text-xl">{p.name.value ?? "İsimsiz ürün"}</p>
                <p className="mt-1 text-sm text-muted">
                  {[p.brand.value, p.category.value].filter(Boolean).join(" · ") || "Detay yok"}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </AppShell>
  );
}

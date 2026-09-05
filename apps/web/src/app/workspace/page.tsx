"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { UsagePanel, type UsageDashboardData } from "@/components/llm/usage-panel";
import { ProductCardMeta } from "@/components/product-fields";
import { graphqlRequest } from "@/lib/graphql";
import { PRODUCT_FIELD_SELECTION, type Product } from "@/lib/product";
import { monthStartIsoDate } from "@/lib/llm-events";

type Workspace = {
  organizationId: string;
  name: string;
  type: string;
  role: string;
};

type Me = {
  id: string;
  email: string;
};

export default function WorkspacePage() {
  const [viewState, setViewState] = useState<"loading" | "unauthorized" | "success" | "error">("loading");
  const [me, setMe] = useState<Me | null>(null);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [productOrgId, setProductOrgId] = useState<string | null>(null);
  const [llmUsage, setLlmUsage] = useState<UsageDashboardData | null>(null);

  useEffect(() => {
    graphqlRequest<{ me: Me; myWorkspaces: Workspace[] }>(`{
      me { id email }
      myWorkspaces { organizationId name type role }
    }`)
      .then(async (data) => {
        setMe(data.me);
        setWorkspaces(data.myWorkspaces);
        const org = data.myWorkspaces.find((ws) => ws.type === "ORGANIZATION") ?? data.myWorkspaces[0];
        if (org) {
          const [productResult, usageResult] = await Promise.all([
            graphqlRequest<{ products: Product[] }>(
              `query($id: ID!) { products(organizationId: $id, limit: 8) { ${PRODUCT_FIELD_SELECTION} } }`,
              { id: org.organizationId },
            ),
            graphqlRequest<{ llmUsageDashboard: UsageDashboardData }>(
              `query($id: ID!, $fromDate: String!) {
                llmUsageDashboard(organizationId: $id, fromDate: $fromDate, recentLimit: 5) {
                  summary { callCount inputTokens outputTokens totalTokens estimatedCostUsd fallbackCount }
                  byModel { modelKey displayName callCount inputTokens outputTokens totalTokens estimatedCostUsd }
                  recentCalls { id modelKey personaKey routingReason fallbackUsed inputTokens outputTokens latencyMs status createdAt }
                }
              }`,
              { id: org.organizationId, fromDate: monthStartIsoDate() },
            ).catch(() => null),
          ]);
          setProductOrgId(org.organizationId);
          setProducts(productResult.products);
          if (usageResult) setLlmUsage(usageResult.llmUsageDashboard);
        }
        setViewState("success");
      })
      .catch((err) => {
        if (err instanceof Error && err.message === "UNAUTHORIZED") {
          setViewState("unauthorized");
        } else {
          setViewState("error");
        }
      });
  }, []);

  return (
    <AppShell
      title="Çalışma alanı"
      kicker="Miyuna"
      description="Organizasyonlarınız, rolleriniz ve ürün analizine giriş."
      accountEmail={me?.email}
      actions={
        viewState === "success" ? (
          <Link href="/org/create" className="btn-primary">
            Organizasyon oluştur
          </Link>
        ) : undefined
      }
    >
      {viewState === "loading" && <AsyncView state="loading" />}
      {viewState === "unauthorized" && (
        <AsyncView
          state="unauthorized"
          unauthorized={
            <div className="card space-y-3 text-center">
              <p className="text-sm text-muted">Oturum gerekli.</p>
              <Link href="/auth/login" className="btn-primary">
                Giriş yap
              </Link>
            </div>
          }
        />
      )}
      {viewState === "error" && <AsyncView state="error" />}
      {viewState === "success" && me && (
        <section className="space-y-6">
          <div className="card">
            <p className="kicker">Hesap</p>
            <p className="mt-2 font-display text-2xl">{me.email}</p>
          </div>
          <div className="card">
            <h2 className="font-display text-2xl">Çalışma alanları</h2>
            {workspaces.length === 0 ? (
              <div className="mt-4">
                <AsyncView state="empty" />
              </div>
            ) : (
              <ul className="mt-5 space-y-3">
                {workspaces.map((ws) => (
                  <li
                    key={ws.organizationId}
                    className="flex flex-col gap-3 rounded-2xl border border-sand bg-cream/60 px-4 py-4 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div>
                      <p className="font-semibold">{ws.name}</p>
                      <p className="text-sm text-muted">{ws.type}</p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="badge-muted">{ws.role}</span>
                      {ws.type === "ORGANIZATION" && (
                        <>
                          <Link href={`/org/${ws.organizationId}/products`} className="btn-secondary !px-3 !py-1.5 text-xs">
                            Ürünler
                          </Link>
                          <Link href={`/org/${ws.organizationId}/llm`} className="btn-ghost !px-3 !py-1.5 text-xs">
                            LLM
                          </Link>
                          <Link href={`/org/${ws.organizationId}/members`} className="btn-ghost !px-3 !py-1.5 text-xs">
                            Yönet
                          </Link>
                        </>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
          {productOrgId && llmUsage && (
            <div className="card">
              <UsagePanel orgId={productOrgId} usage={llmUsage} compact title="LLM kullanım özeti" />
            </div>
          )}
          {productOrgId && (
            <div className="card">
              <div className="flex items-center justify-between gap-3">
                <h2 className="font-display text-2xl">Veritabanındaki ürünler</h2>
                <Link href={`/org/${productOrgId}/products`} className="btn-ghost !px-3 !py-1.5 text-xs">
                  Tümünü gör
                </Link>
              </div>
              {products.length === 0 ? (
                <div className="mt-4">
                  <AsyncView state="empty" />
                </div>
              ) : (
                <ul className="mt-5 grid gap-3">
                  {products.map((product) => (
                    <li key={product.id}>
                      <Link
                        href={`/org/${productOrgId}/products/${product.id}`}
                        className="block rounded-2xl border border-sand bg-cream/60 px-4 py-4 transition hover:bg-paper"
                      >
                        <ProductCardMeta product={product} />
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </section>
      )}
    </AppShell>
  );
}

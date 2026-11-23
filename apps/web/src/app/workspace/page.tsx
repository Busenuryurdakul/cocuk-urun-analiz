"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { WorkspaceDashboard, WorkspaceDashboardSkeleton } from "@/components/dashboard/workspace-dashboard";
import { AppShell } from "@/components/layout/app-shell";
import { UsagePanel, type UsageDashboardData } from "@/components/llm/usage-panel";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ANALYSIS_RUN_FIELDS, type AnalysisRun, type MarketplaceImportRun } from "@/lib/analysis-types";
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
  const [analysisRuns, setAnalysisRuns] = useState<AnalysisRun[]>([]);
  const [importRuns, setImportRuns] = useState<MarketplaceImportRun[]>([]);
  const [activeOrg, setActiveOrg] = useState<Workspace | null>(null);
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
          setActiveOrg(org);
          const [productResult, runsResult, importsResult, usageResult] = await Promise.all([
            graphqlRequest<{ products: Product[] }>(
              `query($id: ID!) { products(organizationId: $id, limit: 100) { ${PRODUCT_FIELD_SELECTION} } }`,
              { id: org.organizationId },
            ),
            graphqlRequest<{ analysisRuns: AnalysisRun[] }>(
              `query($id: ID!) {
                analysisRuns(organizationId: $id, limit: 50) { ${ANALYSIS_RUN_FIELDS} }
              }`,
              { id: org.organizationId },
            ).catch(() => ({ analysisRuns: [] as AnalysisRun[] })),
            graphqlRequest<{ marketplaceImportRuns: MarketplaceImportRun[] }>(
              `query($id: ID!) {
                marketplaceImportRuns(organizationId: $id, limit: 10) {
                  id source sourceUrl status recordsAccepted recordsRejected createdAt
                }
              }`,
              { id: org.organizationId },
            ).catch(() => ({ marketplaceImportRuns: [] as MarketplaceImportRun[] })),
            graphqlRequest<{ llmUsageDashboard: UsageDashboardData }>(
              `query($id: ID!, $fromDate: String!) {
                llmUsageDashboard(organizationId: $id, fromDate: $fromDate, recentLimit: 12) {
                  summary { callCount inputTokens outputTokens totalTokens estimatedCostUsd fallbackCount }
                  byModel { modelKey displayName callCount inputTokens outputTokens totalTokens estimatedCostUsd }
                  recentCalls { id modelKey personaKey routingReason fallbackUsed inputTokens outputTokens latencyMs status createdAt }
                }
              }`,
              { id: org.organizationId, fromDate: monthStartIsoDate() },
            ).catch(() => null),
          ]);
          setProducts(productResult.products);
          setAnalysisRuns(runsResult.analysisRuns);
          setImportRuns(importsResult.marketplaceImportRuns);
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
      title="Intelligence merkezi"
      kicker="Miyuna"
      description="Ürün kataloğunuzdaki analiz sinyalleri, risk dağılımı ve öncelikli uyarılar."
      orgId={activeOrg?.organizationId}
      organizationLabel={activeOrg?.name}
      accountEmail={me?.email}
      actions={
        viewState === "success" ? (
          <div className="flex flex-wrap gap-2">
            {activeOrg && (
              <Link href={`/org/${activeOrg.organizationId}/products/new`}>
                <Button variant="accent" size="sm">
                  Ürün ekle
                </Button>
              </Link>
            )}
            <Link href="/org/create">
              <Button variant="secondary" size="sm">
                Organizasyon oluştur
              </Button>
            </Link>
          </div>
        ) : undefined
      }
    >
      {viewState === "loading" && <WorkspaceDashboardSkeleton />}
      {viewState === "unauthorized" && (
        <AsyncView
          state="unauthorized"
          unauthorized={
            <Card className="space-y-3 text-center">
              <p className="text-sm text-muted">Oturum gerekli.</p>
              <Link href="/auth/login" className="btn-primary inline-flex">
                Giriş yap
              </Link>
            </Card>
          }
        />
      )}
      {viewState === "error" && <AsyncView state="error" />}
      {viewState === "success" && me && (
        <div className="space-y-10">
          {activeOrg ? (
            <WorkspaceDashboard
              orgId={activeOrg.organizationId}
              orgName={activeOrg.name}
              products={products}
              analysisRuns={analysisRuns}
              importRuns={importRuns}
            />
          ) : (
            <Card variant="tinted" className="text-center">
              <p className="text-sm text-muted">Intelligence görünümü için bir organizasyon gerekli.</p>
              <Link href="/org/create" className="btn-primary mt-4 inline-flex">
                Organizasyon oluştur
              </Link>
            </Card>
          )}

          <section className="grid gap-6 lg:grid-cols-2">
            <Card padding="md">
              <p className="kicker">Hesap</p>
              <p className="mt-2 font-heading text-xl text-ink">{me.email}</p>
            </Card>
            <Card padding="md">
              <h2 className="card-title">Çalışma alanları</h2>
              {workspaces.length === 0 ? (
                <div className="mt-4">
                  <AsyncView state="empty" />
                </div>
              ) : (
                <ul className="mt-4 space-y-2">
                  {workspaces.map((ws) => (
                    <li
                      key={ws.organizationId}
                      className="flex flex-col gap-2 rounded-xl border border-border/70 bg-base px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div>
                        <p className="font-medium text-ink">{ws.name}</p>
                        <p className="text-sm text-muted">{ws.type}</p>
                      </div>
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="badge-muted">{ws.role}</span>
                        {ws.type === "ORGANIZATION" && (
                          <>
                            <Link
                              href={`/org/${ws.organizationId}/products`}
                              className="btn-secondary !px-3 !py-1.5 text-xs"
                            >
                              Ürünler
                            </Link>
                            <Link href={`/org/${ws.organizationId}/llm`} className="btn-ghost !px-3 !py-1.5 text-xs">
                              LLM
                            </Link>
                          </>
                        )}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </Card>
          </section>

          {activeOrg && llmUsage && (
            <Card padding="md">
              <UsagePanel orgId={activeOrg.organizationId} usage={llmUsage} compact title="LLM kullanım özeti" />
            </Card>
          )}
        </div>
      )}
    </AppShell>
  );
}

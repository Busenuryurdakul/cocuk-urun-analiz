"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlRequest } from "@/lib/graphql";

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

  useEffect(() => {
    graphqlRequest<{ me: Me; myWorkspaces: Workspace[] }>(`{
      me { id email }
      myWorkspaces { organizationId name type role }
    }`)
      .then((data) => {
        setMe(data.me);
        setWorkspaces(data.myWorkspaces);
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
        </section>
      )}
    </AppShell>
  );
}

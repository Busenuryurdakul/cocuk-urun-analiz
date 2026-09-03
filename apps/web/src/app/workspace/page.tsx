"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
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

  async function logout() {
    await graphqlRequest(`mutation { logout }`);
    window.location.href = "/auth/login";
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-8 px-6 py-16">
      <header className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium uppercase tracking-wide text-miyuna-600">Miyuna</p>
          <h1 className="text-2xl font-semibold text-slate-900">Workspace</h1>
        </div>
        <button
          type="button"
          onClick={logout}
          className="rounded-md border border-slate-300 px-3 py-1.5 text-sm"
        >
          Çıkış
        </button>
      </header>

      {viewState === "loading" && <AsyncView state="loading" />}
      {viewState === "unauthorized" && (
        <AsyncView
          state="unauthorized"
          unauthorized={
            <div className="space-y-3 text-center">
              <p className="text-sm">Oturum gerekli.</p>
              <Link href="/auth/login" className="text-miyuna-600 underline">
                Giriş yap
              </Link>
            </div>
          }
        />
      )}
      {viewState === "error" && <AsyncView state="error" />}
      {viewState === "success" && me && (
        <section className="space-y-6">
          <div className="rounded-xl border border-slate-200 bg-white p-6">
            <h2 className="mb-1 font-medium">Hesap</h2>
            <p className="text-sm text-slate-600">{me.email}</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-6">
            <h2 className="mb-4 font-medium">Workspace&apos;ler</h2>
            {workspaces.length === 0 ? (
              <AsyncView state="empty" />
            ) : (
              <ul className="space-y-3">
                {workspaces.map((ws) => (
                  <li
                    key={ws.organizationId}
                    className="flex items-center justify-between rounded-lg border border-slate-100 px-4 py-3 text-sm"
                  >
                    <span>
                      {ws.name} <span className="text-slate-500">({ws.type})</span>
                    </span>
                    <span className="rounded bg-slate-100 px-2 py-0.5 text-xs">{ws.role}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>
      )}

      <Link href="/" className="text-sm text-slate-500 underline">
        Ana sayfa
      </Link>
    </main>
  );
}

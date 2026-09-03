"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { graphqlRequest } from "@/lib/graphql";

type Organization = {
  id: string;
  name: string;
  complianceProfile: string;
};

export default function CreateOrganizationPage() {
  const [name, setName] = useState("");
  const [profile, setProfile] = useState("KVKK");
  const [state, setState] = useState<"idle" | "loading" | "success" | "error" | "unauthorized">("idle");
  const [org, setOrg] = useState<Organization | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setState("loading");
    try {
      const data = await graphqlRequest<{ createOrganization: Organization }>(
        `mutation($input: CreateOrganizationInput!) {
          createOrganization(input: $input) { id name complianceProfile }
        }`,
        { input: { name, complianceProfile: profile } },
      );
      setOrg(data.createOrganization);
      setState("success");
    } catch (err) {
      if (err instanceof Error && err.message === "UNAUTHORIZED") {
        setState("unauthorized");
      } else {
        setState("error");
      }
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col gap-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">Organizasyon Oluştur</h1>
      {state === "loading" && <AsyncView state="loading" />}
      {state === "unauthorized" && (
        <AsyncView state="unauthorized" unauthorized={<Link href="/auth/login" className="underline">Giriş yap</Link>} />
      )}
      {state === "error" && <AsyncView state="error" />}
      {state === "success" && org && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-4 text-sm">
          <p>Oluşturuldu: {org.name}</p>
          <Link href={`/org/${org.id}/members`} className="mt-2 inline-block text-miyuna-600 underline">
            Üyeleri yönet
          </Link>
        </div>
      )}
      {(state === "idle" || state === "error") && (
        <form onSubmit={onSubmit} className="space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <label className="block text-sm">
            Ad
            <input className="mt-1 w-full rounded border px-3 py-2" value={name} onChange={(e) => setName(e.target.value)} required />
          </label>
          <label className="block text-sm">
            Uyumluluk profili
            <select className="mt-1 w-full rounded border px-3 py-2" value={profile} onChange={(e) => setProfile(e.target.value)}>
              <option value="KVKK">KVKK</option>
              <option value="GDPR">GDPR</option>
              <option value="BOTH">BOTH</option>
            </select>
          </label>
          <button type="submit" className="rounded-md bg-miyuna-600 px-4 py-2 text-sm text-white">
            Oluştur
          </button>
        </form>
      )}
      <Link href="/workspace" className="text-sm text-slate-500 underline">Workspace</Link>
    </main>
  );
}
